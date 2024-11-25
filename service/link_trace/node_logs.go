package link_trace

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
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
		return FileNodeLogs(ctx, req.NodeType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_WEB:
		return WebReaderNodeLogs(ctx, req.NodeType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_MULTI:
		return MultiNodeLogs(ctx, req.NodeType, req.EntryID, nil, false)
	default:
		return nil, &consts.RetParamError
	}
}

func FileNodeLogs(ctx context.Context, nodeType empyrean_lens.LinkNodeTypeEnum, entryID string, node *empyrean_lens.GraphNode, refresh bool) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取文章详情
	fileInfo, err := plugin.NewFileDao().FindFileById(ctx, entryID)
	if err != nil || fileInfo == nil {
		hlog.CtxErrorf(ctx, "get file info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	// 先查数据库
	hasLog, apiLogs, bizCOde := findNodeLogsFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_FILE), entryID, nodeType)
	if bizCOde != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCOde
	}
	if (len(apiLogs) != 0 || hasLog) && !refresh {
		return &empyrean_lens.LinkNodeLogRespData{
			Logs: apiLogs,
			Cost: getNodeCost(apiLogs),
		}, nil
	}
	start := fileInfo.CreateTime.Add(-1 * time.Hour)
	end := fileInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	bizCode := &consts.BizCode{}
	if node == nil {
		node, bizCode = GetProcessNode(ctx, nodeType, fileInfo.TranslateEntryInfo(), start, end)
		if bizCode != nil || node == nil {
			hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
			return nil, bizCode
		}
		node.Type = nodeType
		if node.EnterTime == "" {
			node.EnterTime = start.Format(consts.DateTimeTemplate)
		}
	}
	// 获取日志列表
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, fileInfo.TranslateEntryInfo(), node)
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

func WebReaderNodeLogs(ctx context.Context, nodeType empyrean_lens.LinkNodeTypeEnum, entryID string, node *empyrean_lens.GraphNode, refresh bool) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取文章详情
	webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, entryID)
	if err != nil || webReaderInfo == nil {
		hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	// 先查数据库
	hasLog, apiLogs, bizCode := findNodeLogsFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_WEB), entryID, nodeType)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	if (len(apiLogs) != 0 || hasLog) && !refresh {
		return &empyrean_lens.LinkNodeLogRespData{
			Logs: apiLogs,
			Cost: getNodeCost(apiLogs),
		}, nil
	}
	start := webReaderInfo.CreateTime.Add(-1 * time.Hour)
	end := webReaderInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	bizCode = &consts.BizCode{}
	if node == nil {
		node, bizCode = GetProcessNode(ctx, nodeType, webReaderInfo.TranslateEntryInfo(), start, end)
		if bizCode != nil || node == nil {
			hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
			return nil, bizCode
		}
		node.Type = nodeType
		if node.EnterTime == "" {
			node.EnterTime = start.Format(consts.DateTimeTemplate)
		}
	}
	// 获取日志列表
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, webReaderInfo.TranslateEntryInfo(), node)
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

func MultiNodeLogs(ctx context.Context, nodeType empyrean_lens.LinkNodeTypeEnum, entryID string, node *empyrean_lens.GraphNode, refresh bool) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取多文档详情
	multiInfo, err := plugin.NewMultiDao().FindMultiById(ctx, entryID)
	if err != nil || multiInfo == nil {
		hlog.CtxErrorf(ctx, "get multi info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	// 先查数据库
	hasLog, apiLogs, bizCOde := findNodeLogsFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_MULTI), entryID, nodeType)
	if bizCOde != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCOde
	}
	if (len(apiLogs) != 0 || hasLog) && !refresh {
		return &empyrean_lens.LinkNodeLogRespData{
			Logs: apiLogs,
			Cost: getNodeCost(apiLogs),
		}, nil
	}
	start := multiInfo.CreateTime.Add(-1 * time.Hour)
	end := multiInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	bizCode := &consts.BizCode{}
	if node == nil {
		node, bizCode = GetProcessNode(ctx, nodeType, multiInfo.TranslateEntryInfo(), start, end)
		if bizCode != nil || node == nil {
			hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
			return nil, bizCode
		}
		node.Type = nodeType
		if node.EnterTime == "" {
			node.EnterTime = start.Format(consts.DateTimeTemplate)
		}
	}
	// 获取日志列表
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, multiInfo.TranslateEntryInfo(), node)
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

func findNodeLogsFromMongo(ctx context.Context, entryType int, entryID string, nodeType empyrean_lens.LinkNodeTypeEnum) (bool, []*empyrean_lens.ApiLog, *consts.BizCode) {
	// 获取node节点记录
	linkTraceGraph, bizCode := bi.NewEntryActionDao().FindByEntryTypeEntryIDAndActionType(ctx, entryType, entryID, int(nodeType))
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryTypeEntryIDAndActionType] get link trace graph failed, err: %v", bizCode)
		return false, nil, &consts.QueryRecordError
	}
	if linkTraceGraph == nil {
		return false, nil, nil
	}
	// 获取日志列表
	logs := []*empyrean_lens.ApiLog{}
	for _, log := range linkTraceGraph.ActionIOs {
		logs = append(logs, log.TranslateApiLogs(linkTraceGraph.ActionType)...)
	}
	hasLog := (linkTraceGraph.ActionStatus == int(empyrean_lens.ActionStatusEnum_SUCCESS) || linkTraceGraph.ActionStatus == int(empyrean_lens.ActionStatusEnum_FAIL))
	return hasLog, logs, nil
}

func NodeErrorAndSafeLogs(ctx context.Context, entryInfo *bi.EntryInfo, node *empyrean_lens.GraphNode) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	timeAt, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
	if node.EnterTime == "" && node.FinishTime == "" {
		return "", []*empyrean_lens.ApiLog{}, nil
	}
	// 获取traceID
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, entryInfo, node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", apiLogs)
		return "", nil, bizCode
	}
	// 获取错误日志和安全日志
	errLogs, safeLogs := getErrorAndSafeLogs(ctx, entryInfo.MultiID, entryInfo.EntryID, node, []aliyun.FileProcessLog{{
		Asctime: timeAt,
		TraceId: traceID,
	}})
	return traceID, append(errLogs, safeLogs...), nil
}

func NodeApiLogs(ctx context.Context, entryInfo *bi.EntryInfo, node *empyrean_lens.GraphNode) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	timeAt, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
	start := timeAt.Add(-24 * time.Hour)
	end := timeAt.Add(24 * time.Hour)
	switch node.Type {
	case empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:
		// 查数据库，伪造输入输出
		apiLogsInputs, apiLogsOuputs := []aliyun.FileProcessLog{}, []aliyun.FileProcessLog{}
		if entryInfo.EntryType == consts.EntryTypeWEB {
			webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, entryInfo.EntryID)
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
			fileInfo, err := plugin.NewFileDao().FindFileById(ctx, entryInfo.EntryID)
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
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInputs, apiLogsOuputs)
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		apiLogsInput, err := aliyun.CrawlerOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.CrawlerOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		apiLogsInput, err := aliyun.WcdOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.WcdOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		apiLogsInput, err := aliyun.SuqinOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.SuqinOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:
		apiLogsInput, err := aliyun.TextParseOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.TextParseOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		apiLogsInput, err := aliyun.EduParserOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.EduParserOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
		apiLogsInput, err := aliyun.AbstractModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.AbstractModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
		apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:
		apiLogsInput, err := aliyun.OutlineModelOutRequestQuery(ctx, entryInfo.EntryID, entryInfo.UserID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.OutlineModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH:
		apiLogsInput, err := aliyun.MultiSingleAnalysisModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiSingleAnalysisModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:
		apiLogsInput, err := aliyun.MultiThemeModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiThemeModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		apiLogsInput, err := aliyun.MultiOutlineModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiOutlineModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput, apiLogsOuput)
	}
	return "", []*empyrean_lens.ApiLog{}, nil
}

func getReqAndResp(ctx context.Context, multiID, entryID string, node *empyrean_lens.GraphNode, apiLogsInput, apiLogsOuput []aliyun.FileProcessLog) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	apiLogs := []*empyrean_lens.ApiLog{}
	if len(apiLogsInput) == 0 && len(apiLogsOuput) == 0 && node.TraceID == "" {
		return "", []*empyrean_lens.ApiLog{}, nil
	}
	// 错误和安全日志
	errLogs, safeLogs := getErrorAndSafeLogs(ctx, multiID, entryID, node, apiLogsInput)
	// 遍历
	for _, input := range apiLogsInput {
		output := aliyun.FileProcessLog{
			Asctime: input.Asctime,
		}
		for _, apiLogOuput := range apiLogsOuput {
			if apiLogOuput.TraceId == input.TraceId && (apiLogOuput.Asctime.After(input.Asctime) || apiLogOuput.Asctime.Equal(input.Asctime)) {
				output = apiLogOuput
				break
			}
		}
		// 输入输出
		apiLog := &empyrean_lens.ApiLog{
			HTTPCode:   200,
			EnterTime:  input.Asctime.Format(consts.DateTimeTemplate),
			FinishTime: output.Asctime.Format(consts.DateTimeTemplate),
			Input:      GetReqRespFromMsg(input.Message, "req:"),
			Output:     GetReqRespFromMsg(output.Message, "resp:"),
			TraceID:    input.TraceId,
		}
		// edu 解析输出特殊处理，asicII 转 字符串
		if node.Type == empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH {
			apiLogOutput := eduOutputTranslate(apiLog.Output)
			if apiLogOutput != "" {
				apiLog.Output = apiLogOutput
			}
		}
		apiLogs = append(apiLogs, apiLog)
		// 错误和安全日志
		for _, errLog := range errLogs {
			if errLog.EnterTime >= apiLog.EnterTime {
				apiLogs = append(apiLogs, errLog)
			}
		}
		if node.Type == empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH {
			apiLogs = append(apiLogs, safeLogs...)
		} else {
			for _, safeLog := range safeLogs {
				if safeLog.EnterTime >= apiLog.EnterTime && safeLog.EnterTime <= apiLog.FinishTime {
					apiLogs = append(apiLogs, safeLog)
				}
			}
		}
	}
	if len(apiLogs) == 0 && (len(errLogs) > 0 || len(safeLogs) > 0) {
		for _, errLog := range errLogs {
			if errLog.EnterTime >= node.EnterTime {
				apiLogs = append(apiLogs, errLog)
			}
		}
		for _, safeLog := range safeLogs {
			if safeLog.EnterTime >= node.EnterTime {
				apiLogs = append(apiLogs, safeLog)
			}
		}
	}
	if node.Type == empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH ||
		node.Type == empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH {
		for _, errLog := range errLogs {
			if node.FinishTime == "" || errLog.EnterTime <= node.FinishTime {
				apiLogs = append(apiLogs, errLog)
			}
		}
	}
	// 日志排序
	sort.Slice(apiLogs, func(i, j int) bool {
		return apiLogs[i].EnterTime > apiLogs[j].EnterTime
	})
	// traceID
	traceID := node.TraceID
	if traceID == "" && len(apiLogsInput) > 0 {
		traceID = apiLogsInput[0].TraceId
	}
	return traceID, apiLogs, nil
}

func getErrorAndSafeLogs(ctx context.Context, multiID, entryID string, node *empyrean_lens.GraphNode, apiLogsInput []aliyun.FileProcessLog) ([]*empyrean_lens.ApiLog, []*empyrean_lens.ApiLog) {
	timeAt, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
	start := timeAt.Add(-24 * time.Hour)
	end := timeAt.Add(24 * time.Hour)
	// trace id
	traceIDs := []string{}
	for _, apiLog := range apiLogsInput {
		if apiLog.Message != "" {
			traceIDs = append(traceIDs, apiLog.TraceId)
		}
	}
	if len(traceIDs) == 0 && node.TraceID != "" {
		traceIDs = append(traceIDs, node.TraceID)
	}
	// 多文档大纲，需要获取所有traceID
	if node.Type == empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH {
		traceLogs, err := aliyun.MultiOutlineErrorTraceQuery(ctx, multiID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineErrorTraceQuery] get err logs failed, err: %v", err)
			return []*empyrean_lens.ApiLog{}, []*empyrean_lens.ApiLog{}
		}
		for _, log := range traceLogs {
			traceIDs = append(traceIDs, log.TraceId)
		}
	}
	// 没有traceID，直接返回
	if len(traceIDs) == 0 {
		return []*empyrean_lens.ApiLog{}, []*empyrean_lens.ApiLog{}
	}
	// 并发获取错误日志、安全日志
	wg, errMapping, safeMapping := sync.WaitGroup{}, sync.Map{}, sync.Map{}
	wg.Add(len(traceIDs) * 2)
	for idx := range traceIDs {
		traceID := traceIDs[idx]
		// 获取error日志
		go func() {
			defer wg.Done()
			var err error
			var apiLogsError []aliyun.FileProcessLog
			if multiID != "" {
				apiLogsError, err = aliyun.MultiTraceIDErrorQuery(ctx, traceID, start, end)
			} else {
				apiLogsError, err = aliyun.SingleTraceIDErrorQuery(ctx, traceID, start, end)
			}
			if err != nil {
				hlog.CtxErrorf(ctx, "[getErrorAndSafeLogs] get err logs failed, err: %v", err)
				return
			}
			errMapping.Store(traceID, apiLogsError)
		}()
		// 获取安全日志
		go func() {
			defer wg.Done()
			var err error
			var safeLogs []aliyun.FileProcessLog
			if multiID != "" {
				safeLogs, err = aliyun.MultiIDSafeQuery(ctx, entryID, start, end)
			} else {
				safeLogs, err = aliyun.TraceIDSafeQuery(ctx, traceID, start, end)
				if len(safeLogs) != 0 {
					safeLogs, err = aliyun.TraceIDSafeQuery(ctx, safeLogs[0].TraceId, start, end)
				}
			}
			if err != nil {
				hlog.CtxErrorf(ctx, "[getErrorAndSafeLogs] get safe logs failed, err: %v", err)
				return
			}
			safeMapping.Store(traceID, safeLogs)
		}()
	}
	wg.Wait()
	errLogs := []*empyrean_lens.ApiLog{}
	errMapping.Range(func(key, value interface{}) bool {
		apiLogsError := value.([]aliyun.FileProcessLog)
		for _, apiLogError := range apiLogsError {
			apiLog := &empyrean_lens.ApiLog{
				HTTPCode:   500,
				EnterTime:  apiLogError.Asctime.Format(consts.DateTimeTemplate),
				FinishTime: apiLogError.Asctime.Format(consts.DateTimeTemplate),
				ErrorMsg:   apiLogError.Message,
				TraceID:    apiLogError.TraceId,
			}
			errLogs = append(errLogs, apiLog)
		}
		return true
	})
	safeLogs := []*empyrean_lens.ApiLog{}
	safeMapping.Range(func(key, value interface{}) bool {
		apiLogsError := value.([]aliyun.FileProcessLog)
		for _, apiLogError := range apiLogsError {
			apiLog := &empyrean_lens.ApiLog{
				HTTPCode:   500,
				EnterTime:  apiLogError.Asctime.Format(consts.DateTimeTemplate),
				FinishTime: apiLogError.Asctime.Format(consts.DateTimeTemplate),
				ErrorMsg:   apiLogError.Message,
				TraceID:    apiLogError.TraceId,
			}
			safeLogs = append(safeLogs, apiLog)
		}
		return true
	})
	return errLogs, safeLogs
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

func GetReqRespFromMsg(msg string, substr string) string {
	if strings.Contains(msg, substr) {
		resp := strings.Split(msg, substr)[1]
		output, err := utillib.DeStrGzip(resp)
		if err != nil {
			output = resp
		}
		return output
	}
	return ""
}

func eduOutputTranslate(output string) string {
	// json解析
	outputList := []interface{}{}
	err := json.Unmarshal([]byte(output), &outputList)
	if err != nil {
		return output
	}
	// 转换格式
	translateList := []string{}
	for _, outputStr := range outputList {
		outputJson := map[string]interface{}{}
		err = json.Unmarshal([]byte(outputStr.(string)), &outputJson)
		if err != nil {
			return output
		}
		data, _ := json.Marshal(outputJson)
		translateList = append(translateList, string(data))
	}
	// json编码
	data, _ := json.Marshal(translateList)
	return string(data)
}
