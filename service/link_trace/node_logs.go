package link_trace

import (
	"context"
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
	// 获取日志列表
	apiLogs, bizCode := NodeApiLogs(ctx, req.EntryID, node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	return &empyrean_lens.LinkNodeLogRespData{
		Logs: apiLogs,
		Cost: getNodeCost(apiLogs),
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
	// 获取日志列表
	apiLogs, bizCode := NodeApiLogs(ctx, req.EntryID, node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	return &empyrean_lens.LinkNodeLogRespData{
		Logs: apiLogs,
		Cost: getNodeCost(apiLogs),
	}, nil
}

func MultiNodeLogs(ctx context.Context, req empyrean_lens.LinkNodeLogReq) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取文章详情
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
	// 获取日志列表
	apiLogs, bizCode := NodeApiLogs(ctx, req.EntryID, node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	return &empyrean_lens.LinkNodeLogRespData{
		Logs: apiLogs,
		Cost: getNodeCost(apiLogs),
	}, nil
}

func NodeApiLogs(ctx context.Context, resourceId string, node *empyrean_lens.GraphNode) ([]*empyrean_lens.ApiLog, *consts.BizCode) {
	timeAt, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
	start := timeAt.Add(-24 * time.Hour)
	end := timeAt.Add(24 * time.Hour)
	switch node.Type {
	case empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:
		apiLogsInput, err := aliyun.UploadOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.UploadOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		apiLogsInput, err := aliyun.CrawlerOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.CrawlerOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		apiLogsInput, err := aliyun.WcdOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.WcdOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		apiLogsInput, err := aliyun.SuqinOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.SuqinOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:
		apiLogsInput, err := aliyun.TextParseOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.TextParseOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		apiLogsInput, err := aliyun.EduParserOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.EduParserOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
		apiLogsInput, err := aliyun.AbstractModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.AbstractModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
		apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:
		apiLogsInput, err := aliyun.OutlineModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.OutlineModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH:
		apiLogsInput, err := aliyun.MultiSingleAnalysisModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiSingleAnalysisModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:
		apiLogsInput, err := aliyun.MultiThemeModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiThemeModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		apiLogsInput, err := aliyun.MultiOutlineModelOutRequestQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiOutlineModelOutResponseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return getReqAndResp(apiLogsInput, apiLogsOuput), nil
	}
	return []*empyrean_lens.ApiLog{}, nil
}

// 只考虑一个 input 一个 output
func getReqAndResp(apiLogsInput, apiLogsOuput []aliyun.FileProcessLog) []*empyrean_lens.ApiLog {
	if len(apiLogsInput) == 0 && len(apiLogsOuput) == 0 {
		return []*empyrean_lens.ApiLog{}
	}
	apiLog := &empyrean_lens.ApiLog{}
	apiLog.HTTPCode = 200
	start, end := time.Now(), time.Unix(0, 0)
	// 获取req
	for _, log := range apiLogsInput {
		// 比较时间
		if log.Asctime.Before(start) {
			start = log.Asctime
		}
		msg := log.Message
		if strings.Contains(msg, "req") {
			// 获取req
			req := strings.Split(msg, "req:")[1]
			input, err := utillib.DeStrGzip(req)
			if err != nil {
				input = req
			}
			apiLog.Input = input
		}
	}
	apiLog.EnterTime = start.Format(consts.DateTimeTemplate)
	// 获取resp
	for _, log := range apiLogsOuput {
		// 比较时间
		if log.Asctime.After(end) {
			end = log.Asctime
		}
		msg := log.Message
		if strings.Contains(msg, "resp") {
			// 获取resp
			resp := strings.Split(msg, "resp:")[1]
			output, err := utillib.DeStrGzip(resp)
			if err != nil {
				output = resp
			}
			apiLog.Output = output
		}
	}
	if apiLog.Output == "" {
		apiLog.HTTPCode = 500
	}
	apiLog.FinishTime = end.Format(consts.DateTimeTemplate)
	return []*empyrean_lens.ApiLog{apiLog}
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
	startAtT, _ := time.Parse(consts.DateHourMinuteTemplate, startAt)
	endAtT, _ := time.Parse(consts.DateHourMinuteTemplate, endAt)
	return endAtT.Sub(startAtT).Seconds()
}
