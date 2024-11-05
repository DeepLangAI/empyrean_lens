package link_trace

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/plugin"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func NodeLogs(ctx context.Context, req empyrean_lens.LinkNodeLogReq) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 由于网页链路中，wcd目前入参并没有entry_id，所以无法查到text-parser日志。
	// 当entry_type为web时，node_type为text-parser时，将node_type改为wcd-parser，以便查询到日志
	if req.EntryType == consts.EntryTypeWEB && req.NodeType == empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH {
		req.NodeType = empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH
	}
	switch req.EntryType {
	case empyrean_lens.EntryTypeEnum_FILE:
		return FileNodeLogs(ctx, req)
	case empyrean_lens.EntryTypeEnum_WEB:
		return WebReaderNodeLogs(ctx, req)
	case empyrean_lens.EntryTypeEnum_MULTI:
		return MultiNodeLogs(ctx, req)
	default:
		return nil, &consts.RetParamError
	}
}

func FileNodeLogs(ctx context.Context, req empyrean_lens.LinkNodeLogReq) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取文章详情
	fileInfo, err := plugin.NewFileDao().FindFileById(ctx, req.EntryID)
	if err != nil || fileInfo == nil {
		hlog.CtxErrorf(ctx, "get file info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	start := fileInfo.CreateTime.Add(-1 * time.Hour)
	end := fileInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	node, bizCode := GetProcessNode(ctx, req.NodeType, req.EntryID, consts.PDF, start, end)
	if bizCode != nil || node == nil {
		hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
		return nil, bizCode
	}
	node.Type = req.NodeType
	if node.EnterTime == "" {
		node.EnterTime = start.Format(consts.DateTimeTemplate)
	}
	// 获取日志列表
	article := plugin.ArticleEntry{
		EntryId:   req.EntryID,
		EntryType: consts.EntryTypePDF,
	}
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, req.EntryID, fileInfo.UserID, article, node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:    apiLogs,
		Cost:    getNodeCost(apiLogs),
		TraceID: traceID,
	}, nil
}

func WebReaderNodeLogs(ctx context.Context, req empyrean_lens.LinkNodeLogReq) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取文章详情
	webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, req.EntryID)
	if err != nil || webReaderInfo == nil {
		hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	start := webReaderInfo.CreateTime.Add(-1 * time.Hour)
	end := webReaderInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	node, bizCode := GetProcessNode(ctx, req.NodeType, req.EntryID, consts.URL, start, end)
	if bizCode != nil || node == nil {
		hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
		return nil, bizCode
	}
	node.Type = req.NodeType
	if node.EnterTime == "" {
		node.EnterTime = start.Format(consts.DateTimeTemplate)
	}
	// 获取日志列表
	article := plugin.ArticleEntry{
		EntryId:   req.EntryID,
		EntryType: consts.EntryTypeWEB,
	}
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, req.EntryID, webReaderInfo.UserID, article, node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:    apiLogs,
		Cost:    getNodeCost(apiLogs),
		TraceID: traceID,
	}, nil
}

func MultiNodeLogs(ctx context.Context, req empyrean_lens.LinkNodeLogReq) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取多文档详情
	multiInfo, err := plugin.NewMultiDao().FindMultiById(ctx, req.EntryID)
	if err != nil || multiInfo == nil {
		hlog.CtxErrorf(ctx, "get multi info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	start := multiInfo.CreateTime.Add(-1 * time.Hour)
	end := multiInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	node, bizCode := GetProcessNode(ctx, req.NodeType, req.EntryID, consts.MULTI, start, end)
	if bizCode != nil || node == nil {
		hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
		return nil, bizCode
	}
	node.Type = req.NodeType
	if node.EnterTime == "" {
		node.EnterTime = start.Format(consts.DateTimeTemplate)
	}
	// 获取日志列表
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, req.EntryID, multiInfo.UserID, plugin.ArticleEntry{}, node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:    apiLogs,
		Cost:    getNodeCost(apiLogs),
		TraceID: traceID,
	}, nil
}

func NodeApiLogs(ctx context.Context, resourceId, userID string, article plugin.ArticleEntry, node *empyrean_lens.GraphNode) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	timeAt, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
	start := timeAt.Add(-24 * time.Hour)
	end := timeAt.Add(24 * time.Hour)
	switch node.Type {
	case empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:
		// 查数据库，伪造输入输出
		apiLogsInputs, apiLogsOuputs := []aliyun.FileProcessLog{}, []aliyun.FileProcessLog{}
		if article.EntryType == consts.EntryTypeWEB {
			webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, article.EntryId)
			if err != nil || webReaderInfo == nil {
				hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			msg, _ := json.Marshal(webReaderInfo)
			apiLogsInputs = append(apiLogsInputs, aliyun.FileProcessLog{
				Asctime: webReaderInfo.CreateTime,
				Message: fmt.Sprintf("req:{\"url\": \"%s\"}", webReaderInfo.URL),
				UserId:  webReaderInfo.UserID,
			})
			apiLogsOuputs = append(apiLogsOuputs, aliyun.FileProcessLog{
				Asctime: webReaderInfo.CreateTime,
				Message: fmt.Sprintf("resp:%s", string(msg)),
				UserId:  webReaderInfo.UserID,
			})
		} else {
			fileInfo, err := plugin.NewFileDao().FindFileById(ctx, article.EntryId)
			if err != nil || fileInfo == nil {
				hlog.CtxErrorf(ctx, "get file info failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			msg, _ := json.Marshal(fileInfo)
			apiLogsInputs = append(apiLogsInputs, aliyun.FileProcessLog{
				Asctime: fileInfo.CreateTime,
				Message: fmt.Sprintf("req:{\"url\": \"%s\"}", fileInfo.FileURL),
				UserId:  fileInfo.UserID,
			})
			apiLogsOuputs = append(apiLogsOuputs, aliyun.FileProcessLog{
				Asctime: fileInfo.CreateTime,
				Message: fmt.Sprintf("resp:%s", string(msg)),
				UserId:  fileInfo.UserID,
			})
		}
		return getReqAndResp(ctx, node, apiLogsInputs, apiLogsOuputs)
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		apiLogsInput, err := aliyun.CrawlerOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.CrawlerOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		apiLogsInput, err := aliyun.WcdOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.WcdOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		apiLogsInput, err := aliyun.SuqinOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.SuqinOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:
		apiLogsInput, err := aliyun.TextParseOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.TextParseOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		apiLogsInput, err := aliyun.EduParserOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.EduParserOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
		apiLogsInput, err := aliyun.AbstractModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.AbstractModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
		apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:
		apiLogsInput, err := aliyun.OutlineModelOutRequestQuery(ctx, resourceId, userID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.OutlineModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH:
		apiLogsInput, err := aliyun.MultiSingleAnalysisModelOutRequestQuery(ctx, article.EntryId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiSingleAnalysisModelOutResponseQuery(ctx, article.EntryId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:
		apiLogsInput, err := aliyun.MultiThemeModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiThemeModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		apiLogsInput, err := aliyun.MultiOutlineModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiOutlineModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, node, apiLogsInput, apiLogsOuput)
	}
	return "", []*empyrean_lens.ApiLog{}, nil
}

func getReqAndResp(ctx context.Context, node *empyrean_lens.GraphNode, apiLogsInput, apiLogsOuput []aliyun.FileProcessLog) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	apiLogs := []*empyrean_lens.ApiLog{}
	if len(apiLogsInput) == 0 && len(apiLogsOuput) == 0 && node.TraceID == "" {
		return "", []*empyrean_lens.ApiLog{}, nil
	}
	maxLex := int(math.Max(float64(len(apiLogsInput)), float64(len(apiLogsOuput))))
	// 输入输出
	for idx := 0; idx < maxLex; idx++ {
		input := aliyun.FileProcessLog{}
		if idx < len(apiLogsInput) {
			input = apiLogsInput[idx]
		}
		output := aliyun.FileProcessLog{}
		for _, apiLogOuput := range apiLogsOuput {
			if apiLogOuput.TraceId == input.TraceId {
				output = apiLogOuput
				break
			}
		}
		apiLog := &empyrean_lens.ApiLog{}
		apiLog.HTTPCode = 200
		apiLog.EnterTime = input.Asctime.Format(consts.DateTimeTemplate)
		apiLog.FinishTime = output.Asctime.Format(consts.DateTimeTemplate)
		// 输入
		if strings.Contains(input.Message, "req") {
			// 获取req
			req := strings.Split(input.Message, "req:")[1]
			input, err := utillib.DeStrGzip(req)
			if err != nil {
				input = req
			}
			apiLog.Input = input
		}
		// 输出
		if strings.Contains(output.Message, "resp") {
			// 获取req
			resp := strings.Split(output.Message, "resp:")[1]
			output, err := utillib.DeStrGzip(resp)
			if err != nil {
				output = resp
			}
			apiLog.Output = output
		}
		apiLogs = append(apiLogs, apiLog)
	}
	// error 日志
	traceID := node.TraceID
	if traceID != "" && len(apiLogsInput) > 0 {
		traceID = apiLogsInput[0].TraceId
	}
	timeAt, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
	start, end := timeAt.Add(-24*time.Hour), timeAt.Add(24*time.Hour)
	if node.Status != empyrean_lens.ActionStatusEnum_SUCCESS && traceID != "" {
		apiLogsError, err := aliyun.TraceIDErrorQuery(ctx, traceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get err logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		for _, logError := range apiLogsError {
			apiLog := &empyrean_lens.ApiLog{
				HTTPCode:   500,
				EnterTime:  logError.Asctime.Format(consts.DateTimeTemplate),
				FinishTime: logError.Asctime.Format(consts.DateTimeTemplate),
				ErrorMsg:   logError.Message,
			}
			apiLogs = append(apiLogs, apiLog)
		}
	}
	return traceID, apiLogs, nil
}

func getNodeCost(apiLogs []*empyrean_lens.ApiLog) float64 {
	startAt, endAt := "", ""
	for _, apiLog := range apiLogs {
		if startAt == "" || strings.Compare(startAt, apiLog.EnterTime) > 0 {
			startAt = apiLog.EnterTime
		}
		if endAt == "" || strings.Compare(endAt, apiLog.FinishTime) < 0 {
			endAt = apiLog.FinishTime
		}
	}
	if startAt == "" || endAt == "" {
		return 0
	}
	startAtT, _ := time.Parse(consts.DateTimeTemplate, startAt)
	endAtT, _ := time.Parse(consts.DateTimeTemplate, endAt)
	return endAtT.Sub(startAtT).Seconds()
}
