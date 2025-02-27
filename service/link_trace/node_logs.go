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
	"empyrean_lens/tools"
	"empyrean_lens/utils"

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
	case empyrean_lens.EntryTypeEnum_SUMMARY:
		return SummaryNodeLogs(ctx, req.NodeType, req.EntryType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_OUTLINE:
		return SummaryNodeLogs(ctx, req.NodeType, req.EntryType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_VIEWPOINT:
		return SummaryNodeLogs(ctx, req.NodeType, req.EntryType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_MULTI, empyrean_lens.EntryTypeEnum_MULTI_OUTLINE:
		if entryType, ok := consts.RetryProcessToEntryType[req.NodeType]; ok {
			return MultiOutlineNodeLogs(ctx, req.NodeType, entryType, req.EntryID, nil, false)
		}
		return MultiNodeLogs(ctx, req.NodeType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB:
		return SubscribeNodeLogs(ctx, req.NodeType, req.EntryType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE:
		return SubscribeNodeLogs(ctx, req.NodeType, req.EntryType, req.EntryID, nil, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI:
		return SubscribeNodeLogs(ctx, req.NodeType, req.EntryType, req.EntryID, nil, false)
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
	hasLog, apiLogs, action, bizCode := findNodeLogsFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_FILE), entryID, nodeType)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	if (len(apiLogs) != 0 || hasLog) && !refresh {
		traceId := ""
		if len(apiLogs) != 0 {
			traceId = apiLogs[0].TraceID
		}
		return &empyrean_lens.LinkNodeLogRespData{
			Logs:       apiLogs,
			Cost:       getNodeCost(apiLogs),
			TraceID:    traceId,
			Title:      fileInfo.Name,
			UserID:     fileInfo.UserID,
			ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_WEB), fileInfo.MultiId, fileInfo.CopyFromResourceID, "", 0),
			Status:     empyrean_lens.ActionStatusEnum(action.ActionStatus),
			NodeName:   consts.LinkNodeTypeName[nodeType],
			TimeAt:     action.ActionStartTime.Format(consts.DateTimeTemplate),
		}, nil
	}
	start := fileInfo.CreateTime.Add(-1 * time.Hour)
	end := fileInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
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
	_, status := utils.GetStatusFromNode([]*empyrean_lens.GraphNode{node})
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:       apiLogs,
		Cost:       getNodeCost(apiLogs),
		TraceID:    traceID,
		Title:      fileInfo.Name,
		UserID:     fileInfo.UserID,
		ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_FILE), fileInfo.MultiId, fileInfo.CopyFromResourceID, "", 0),
		Status:     status,
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
	hasLog, apiLogs, action, bizCode := findNodeLogsFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_WEB), entryID, nodeType)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	if (len(apiLogs) != 0 || hasLog) && !refresh {
		traceId := ""
		if len(apiLogs) != 0 {
			traceId = apiLogs[0].TraceID
		}
		return &empyrean_lens.LinkNodeLogRespData{
			Logs:       apiLogs,
			Cost:       getNodeCost(apiLogs),
			TraceID:    traceId,
			Title:      webReaderInfo.Title,
			UserID:     webReaderInfo.UserID,
			ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_WEB), webReaderInfo.MultiId, webReaderInfo.CopyFromResourceID, "", 0),
			Status:     empyrean_lens.ActionStatusEnum(action.ActionStatus),
			NodeName:   consts.LinkNodeTypeName[nodeType],
			TimeAt:     action.ActionStartTime.Format(consts.DateTimeTemplate),
		}, nil
	}
	start := webReaderInfo.CreateTime.Add(-1 * time.Hour)
	end := webReaderInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
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
	_, status := utils.GetStatusFromNode([]*empyrean_lens.GraphNode{node})
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:       apiLogs,
		Cost:       getNodeCost(apiLogs),
		TraceID:    traceID,
		Title:      webReaderInfo.Title,
		UserID:     webReaderInfo.UserID,
		ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_WEB), webReaderInfo.MultiId, webReaderInfo.CopyFromResourceID, "", 0),
		Status:     status,
	}, nil
}

func SummaryNodeLogs(ctx context.Context, nodeType empyrean_lens.LinkNodeTypeEnum, entryType empyrean_lens.EntryTypeEnum, entryID string, node *empyrean_lens.GraphNode, refresh bool) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取summary信息
	summaryInfo, err := plugin.NewSummaryDao().QueryByTypeAndID(ctx, int(entryType), entryID)
	if err != nil {
		hlog.CtxErrorf(ctx, "get summary info failed, entry_id:%s, err: %v", entryID, err)
		return nil, &consts.QueryRecordError
	}
	// 查询文章的链路追踪，没有模型生成节点
	articleInfo, articleLinkTraceGroup, bizCode := SummaryArticleTrace(ctx, summaryInfo, refresh)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[SummaryArticleTrace] get article graph failed, err: %v", bizCode)
		return nil, bizCode
	}
	actionName := ""
	if articleLinkTraceGroup.EntryType == consts.EntryTypePDF {
		actionName = utils.GetActionName(int(entryType), articleInfo.(*plugin.File).MultiId, articleInfo.(*plugin.File).CopyFromResourceID, summaryInfo.SummaryLangType, summaryInfo.OutlineType)
		if _, ok := consts.RetryProcessToEntryType[nodeType]; !ok {
			return FileNodeLogs(ctx, nodeType, articleInfo.(*plugin.File).ID.Hex(), nil, false)
		}
	} else {
		actionName = utils.GetActionName(int(entryType), articleInfo.(*plugin.WebReader).MultiId, articleInfo.(*plugin.WebReader).CopyFromResourceID, summaryInfo.SummaryLangType, summaryInfo.OutlineType)
		if _, ok := consts.RetryProcessToEntryType[nodeType]; !ok {
			return WebReaderNodeLogs(ctx, nodeType, articleInfo.(*plugin.WebReader).ID.Hex(), nil, false)
		}
	}
	// 先查数据库
	hasLog, apiLogs, action, bizCode := findNodeLogsFromMongo(ctx, int(entryType), entryID, nodeType)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	if (len(apiLogs) != 0 && hasLog) && !refresh {
		return &empyrean_lens.LinkNodeLogRespData{
			Logs:       apiLogs,
			Cost:       getNodeCost(apiLogs),
			TraceID:    apiLogs[0].TraceID,
			Title:      articleLinkTraceGroup.Title,
			UserID:     summaryInfo.UserID,
			ActionName: actionName,
			Status:     empyrean_lens.ActionStatusEnum(action.ActionStatus),
			NodeName:   consts.LinkNodeTypeName[nodeType],
			TimeAt:     action.ActionStartTime.Format(consts.DateTimeTemplate),
		}, nil
	}
	// 获取节点日志
	start := summaryInfo.CreateTime.Add(-1 * time.Hour)
	end := summaryInfo.CreateTime.Add(24 * time.Hour)
	if node == nil {
		node, bizCode = GetProcessNode(ctx, nodeType, summaryInfo.TranslateEntryInfo(), start, end)
		if bizCode != nil || node == nil {
			hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
			return nil, bizCode
		}
		if node.EnterTime == "" {
			node.EnterTime = start.Format(consts.DateTimeTemplate)
		}
	}
	// 获取日志列表
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, summaryInfo.TranslateEntryInfo(), node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	// 返回数据
	_, status := utils.GetStatusFromNode([]*empyrean_lens.GraphNode{node})
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:       apiLogs,
		Cost:       getNodeCost(apiLogs),
		TraceID:    traceID,
		Title:      articleLinkTraceGroup.Title,
		UserID:     summaryInfo.UserID,
		ActionName: actionName,
		Status:     status,
	}, nil
}

func SubscribeNodeLogs(ctx context.Context, nodeType empyrean_lens.LinkNodeTypeEnum, entryType empyrean_lens.EntryTypeEnum, entryID string, node *empyrean_lens.GraphNode, refresh bool) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取文章详情
	resourceInfo, err := plugin.NewResourceDao().FindResourceById(ctx, entryID)
	if err != nil || resourceInfo == nil {
		hlog.CtxErrorf(ctx, "get resource info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	// 先查数据库
	hasLog, apiLogs, action, bizCode := findNodeLogsFromMongo(ctx, int(entryType), entryID, nodeType)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	if (len(apiLogs) != 0 && hasLog) && !refresh {
		return &empyrean_lens.LinkNodeLogRespData{
			Logs:       apiLogs,
			Cost:       getNodeCost(apiLogs),
			TraceID:    apiLogs[0].TraceID,
			Title:      resourceInfo.Title,
			UserID:     "resource_server",
			ActionName: utils.GetActionName(int(entryType), "", "", "", 0),
			Status:     empyrean_lens.ActionStatusEnum(action.ActionStatus),
			NodeName:   consts.LinkNodeTypeName[nodeType],
			TimeAt:     action.ActionStartTime.Format(consts.DateTimeTemplate),
		}, nil
	}
	start := resourceInfo.CreateTime.Add(-1 * time.Hour)
	end := resourceInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	if node == nil {
		node, bizCode = GetProcessNode(ctx, nodeType, resourceInfo.TranslateEntryInfo(), start, end)
		if bizCode != nil || node == nil {
			hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
			return nil, bizCode
		}
		node.Type = nodeType
	}
	if node.EnterTime == "" {
		node.EnterTime = start.Format(consts.DateTimeTemplate)
	}
	// 获取日志列表
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, resourceInfo.TranslateEntryInfo(), node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	_, status := utils.GetStatusFromNode([]*empyrean_lens.GraphNode{node})
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:       apiLogs,
		Cost:       getNodeCost(apiLogs),
		TraceID:    traceID,
		Title:      resourceInfo.Title,
		UserID:     resourceInfo.UserID,
		ActionName: utils.GetActionName(int(entryType), "", "", "", 0),
		Status:     status,
	}, nil
}

func MultiNodeLogs(ctx context.Context, nodeType empyrean_lens.LinkNodeTypeEnum, entryID string, node *empyrean_lens.GraphNode, refresh bool) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取多文档详情
	multiInfo, err := plugin.NewMultiDao().FindMultiById(ctx, entryID)
	if err != nil {
		hlog.CtxErrorf(ctx, "get multi info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	// ID来源于多文档大纲
	if multiInfo == nil {
		multiAigcInfo, err := plugin.NewMultiAigcDao().QueryByID(ctx, entryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "get multi aigc info failed, entry_id:%s, err: %v", entryID, err)
			return nil, &consts.QueryRecordError
		}
		multiInfo, err = plugin.NewMultiDao().FindMultiById(ctx, multiAigcInfo.MultiID)
		if err != nil {
			hlog.CtxErrorf(ctx, "get multi info failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
	}
	// 先查数据库
	hasLog, apiLogs, action, bizCode := findNodeLogsFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_MULTI), entryID, nodeType)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	if (len(apiLogs) != 0 || hasLog) && !refresh {
		traceId := ""
		if len(apiLogs) != 0 {
			traceId = apiLogs[0].TraceID
		}
		return &empyrean_lens.LinkNodeLogRespData{
			Logs:       apiLogs,
			Cost:       getNodeCost(apiLogs),
			TraceID:    traceId,
			Title:      multiInfo.Title,
			UserID:     multiInfo.UserID,
			ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_WEB), "", multiInfo.CopyFromResourceID, "", 0),
			Status:     empyrean_lens.ActionStatusEnum(action.ActionStatus),
			NodeName:   consts.LinkNodeTypeName[nodeType],
			TimeAt:     action.ActionStartTime.Format(consts.DateTimeTemplate),
		}, nil
	}
	start := multiInfo.CreateTime.Add(-1 * time.Hour)
	end := multiInfo.CreateTime.Add(24 * time.Hour)
	// 获取节点日志
	bizCode = &consts.BizCode{}
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
	_, status := utils.GetStatusFromNode([]*empyrean_lens.GraphNode{node})
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:       apiLogs,
		Cost:       getNodeCost(apiLogs),
		TraceID:    traceID,
		Title:      multiInfo.Title,
		UserID:     multiInfo.UserID,
		ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_WEB), "", multiInfo.CopyFromResourceID, "", 0),
		Status:     status,
	}, nil
}

func MultiOutlineNodeLogs(ctx context.Context, nodeType empyrean_lens.LinkNodeTypeEnum, entryType empyrean_lens.EntryTypeEnum, entryID string, node *empyrean_lens.GraphNode, refresh bool) (*empyrean_lens.LinkNodeLogRespData, *consts.BizCode) {
	// 获取summary信息
	multiAigcInfo, err := plugin.NewMultiAigcDao().QueryByID(ctx, entryID)
	if err != nil {
		hlog.CtxErrorf(ctx, "get multi aigc info failed, entry_id:%s, err: %v", entryID, err)
		return nil, &consts.QueryRecordError
	}
	// 查询文章的链路追踪，没有模型生成节点
	multiInfo, err := plugin.NewMultiDao().FindMultiById(ctx, multiAigcInfo.MultiID)
	if err != nil {
		hlog.CtxErrorf(ctx, "[MultiLinkTrace] get article graph failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	actionName := utils.GetActionName(int(entryType), multiInfo.ID.Hex(), multiInfo.CopyFromResourceID, "", 0)
	// 先查数据库
	hasLog, apiLogs, action, bizCode := findNodeLogsFromMongo(ctx, int(entryType), entryID, nodeType)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	if (len(apiLogs) != 0 && hasLog) && !refresh {
		return &empyrean_lens.LinkNodeLogRespData{
			Logs:       apiLogs,
			Cost:       getNodeCost(apiLogs),
			TraceID:    apiLogs[0].TraceID,
			Title:      multiInfo.Title,
			UserID:     multiAigcInfo.UserID,
			ActionName: actionName,
			Status:     empyrean_lens.ActionStatusEnum(action.ActionStatus),
			NodeName:   consts.LinkNodeTypeName[nodeType],
			TimeAt:     action.ActionStartTime.Format(consts.DateTimeTemplate),
		}, nil
	}
	// 获取节点日志
	start := multiAigcInfo.CreateTime.Add(-1 * time.Hour)
	end := multiAigcInfo.CreateTime.Add(24 * time.Hour)
	if node == nil {
		node, bizCode = GetProcessNode(ctx, nodeType, multiAigcInfo.TranslateEntryInfo(), start, end)
		if bizCode != nil || node == nil {
			hlog.CtxErrorf(ctx, "[GetProcessNode] get node failed, err: %v", err)
			return nil, bizCode
		}
		if node.EnterTime == "" {
			node.EnterTime = start.Format(consts.DateTimeTemplate)
		}
	}
	traceID, apiLogs, bizCode := NodeApiLogs(ctx, multiAigcInfo.TranslateEntryInfo(), node)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
		return nil, bizCode
	}
	// 返回数据
	_, status := utils.GetStatusFromNode([]*empyrean_lens.GraphNode{node})
	return &empyrean_lens.LinkNodeLogRespData{
		Logs:       apiLogs,
		Cost:       getNodeCost(apiLogs),
		TraceID:    traceID,
		Title:      multiInfo.Title,
		UserID:     multiInfo.UserID,
		ActionName: actionName,
		Status:     status,
		NodeName:   consts.LinkNodeTypeName[nodeType],
		TimeAt:     node.EnterTime,
	}, nil
}

func findNodeLogsFromMongo(ctx context.Context, entryType int, entryID string, nodeType empyrean_lens.LinkNodeTypeEnum) (bool, []*empyrean_lens.ApiLog, *bi.EntryAction, *consts.BizCode) {
	// 获取node节点记录
	entryAction, bizCode := bi.NewEntryActionDao().FindByEntryTypeEntryIDAndActionType(ctx, entryType, entryID, int(nodeType))
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryTypeEntryIDAndActionType] get link trace graph failed, err: %v", bizCode)
		return false, nil, nil, &consts.QueryRecordError
	}
	if entryAction == nil {
		return false, nil, nil, nil
	}
	// 获取日志列表
	logs := []*empyrean_lens.ApiLog{}
	for _, log := range entryAction.ActionIOs {
		logs = append(logs, log.TranslateApiLogs(entryAction.ActionType)...)
	}
	hasLog := (entryAction.ActionStatus == int(empyrean_lens.ActionStatusEnum_SUCCESS) || entryAction.ActionStatus == int(empyrean_lens.ActionStatusEnum_FAIL))
	//hasLog = hasLog && (len(logs) != 0)
	return hasLog, logs, entryAction, nil
}

func NodeApiLogs(ctx context.Context, entryInfo *bi.EntryInfo, node *empyrean_lens.GraphNode) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	// 是否是copy来的
	if entryInfo.ParentEntryID != "" && utils.IsCopyNodeType(entryInfo.ParentEntryType, int(node.Type)) {
		// 先从数据库拿日志
		hasLog, apiLogs, _, bizCode := findNodeLogsFromMongo(ctx, entryInfo.ParentEntryType, entryInfo.ParentEntryID, node.Type)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", bizCode)
			return "", nil, bizCode
		}
		if len(apiLogs) != 0 && hasLog {
			return apiLogs[0].TraceID, apiLogs, nil
		}
		// 再从ailiyun拿日志
		// 获取文章信息
		newEntryInfo, err := GetEntryInfo(ctx, empyrean_lens.EntryTypeEnum(entryInfo.ParentEntryType), entryInfo.ParentEntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "get entry info failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		// 获取节点信息
		start := newEntryInfo.EntryCreateTime.Add(-24 * time.Hour)
		end := newEntryInfo.EntryCreateTime.Add(24 * time.Hour)
		node, bizCode = GetProcessNode(ctx, node.Type, newEntryInfo, start, end)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[findNodeLogsFromMongo] get api logs failed, err: %v", bizCode)
			return "", nil, bizCode
		}
		return doNodeApiLogs(ctx, newEntryInfo, node)
	}
	// 从阿里云拿日志
	return doNodeApiLogs(ctx, entryInfo, node)
}

func doNodeApiLogs(ctx context.Context, entryInfo *bi.EntryInfo, node *empyrean_lens.GraphNode) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	start := entryInfo.EntryCreateTime.Add(-24 * time.Hour)
	end := entryInfo.EntryCreateTime.Add(24 * time.Hour)
	if node != nil && node.EnterTime != "" {
		timeAt, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
		start = timeAt.Add(-24 * time.Hour)
		end = timeAt.Add(24 * time.Hour)
	}
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInputs, apiLogsOuputs)
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		// 来自订阅
		if utils.IsSubscribe(entryInfo.EntryType) {
			// 获取订阅traceID
			query := "ResourceProcessor " + entryInfo.EntryID
			logs, err := aliyun.TraceIDQuery(ctx, query, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[TraceIDQuery] get process logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			// 获取crawler日志
			traceIDMapping := make(map[string]struct{})
			for _, log := range logs {
				if log.TraceId != "" {
					traceIDMapping[log.TraceId] = struct{}{}
				}
			}
			totalApiLogsInput := []aliyun.FileProcessLog{}
			for traceID := range traceIDMapping {
				apiLogsInput, err := aliyun.CrawlerSubscribeQuery(ctx, traceID, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[CrawlerSubscribeQuery] get process logs failed, err: %v", err)
					return "", nil, &consts.QueryRecordError
				}
				for _, log := range apiLogsInput {
					if strings.Contains(log.Message, entryInfo.EntryURL) || strings.Contains(log.Message, "ResourceCrawled") || strings.Contains(log.Message, "UrlChecked") {
						totalApiLogsInput = append(totalApiLogsInput, log)
						break
					}
				}
			}
			return getReqAndResp(ctx, entryInfo, node, totalApiLogsInput, []aliyun.FileProcessLog{})
		}
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		apiLogsInput, err := aliyun.EduParserOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		// 获取每一行日志
		var apiLogsOuput []aliyun.FileProcessLog
		for _, inputLog := range apiLogsInput {
			lineLogs, err := aliyun.EduParserOutResponseLinesQuery(ctx, entryInfo.EntryID, inputLog.TraceId, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			totalOutPut := make([]string, 0, len(lineLogs))
			for _, line := range lineLogs {
				lineStr := GetReqRespFromMsg(line.Message, "line:")
				totalOutPut = append(totalOutPut, lineStr)
			}
			if len(totalOutPut) > 0 {
				totalOutPutStr, _ := json.Marshal(totalOutPut)
				lineLogs[0].Message = "resp:" + string(totalOutPutStr)
				apiLogsOuput = append(apiLogsOuput, lineLogs[0])
			}
		}
		if len(apiLogsOuput) == 0 {
			// 如果没有逐行记录
			apiLogsOuput, err = aliyun.EduParserOutResponseQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
		}
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
		if node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD {
			return node.TraceID, []*empyrean_lens.ApiLog{}, nil
		}
		if node.TraceID != "" {
			apiLogsInput, err := aliyun.AbstractModelOutRequestQueryByTraceID(ctx, node.TraceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[AbstractModelOutRequestQueryByTraceID] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.AbstractModelOutResponseQueryByTraceID(ctx, node.TraceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[AbstractModelOutResponseQueryByTraceID] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
		}
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
		if node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD {
			return node.TraceID, []*empyrean_lens.ApiLog{}, nil
		}
		if node.TraceID != "" {
			apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, node.TraceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ViewPointModelOutRequestQuery] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, node.TraceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ViewPointModelOutResponseQuery] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
		}
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
		empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
		empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH:
		if node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD {
			return node.TraceID, []*empyrean_lens.ApiLog{}, nil
		}
		var err error
		var apiLogsInput []aliyun.FileProcessLog
		var apiLogsOuput []aliyun.FileProcessLog
		if node.TraceID != "" {
			apiLogsInput, err = aliyun.OutlineModelOutRequestQueryByTraceID(ctx, node.TraceID, "", start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[OutlineModelOutRequestQueryByTraceID] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			apiLogsOuput, err = aliyun.OutlineModelOutResponseQueryByTraceID(ctx, node.TraceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[AbstractModelOutResponseQueryByTraceID] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
		} else {
			apiLogsInput, err = aliyun.OutlineModelOutRequestQuery(ctx, entryInfo.EntryID, entryInfo.UserID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
			apiLogsOuput, err = aliyun.OutlineModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
				return "", nil, &consts.QueryRecordError
			}
		}
		newApiLogsInput := []aliyun.FileProcessLog{}
		for _, log := range apiLogsInput {
			// 解压缩
			inputStr := GetReqRespFromMsg(log.Message, "req:")
			if inputStr == "" {
				inputStr = log.Message
			}
			if node.Type == empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH {
				if strings.Contains(inputStr, "\"verbose\":true") {
					newApiLogsInput = append(newApiLogsInput, log)
				}
			} else {
				if strings.Contains(inputStr, "\"verbose\":false") {
					newApiLogsInput = append(newApiLogsInput, log)
				}
			}
		}
		newApiLogsOuput := []aliyun.FileProcessLog{}
		if len(newApiLogsInput) > 0 {
			for _, log := range apiLogsOuput {
				if log.OperationID == newApiLogsInput[0].OperationID {
					newApiLogsOuput = append(newApiLogsOuput, log)
				}
			}
		}
		return getReqAndResp(ctx, entryInfo, node, newApiLogsInput, newApiLogsOuput)
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
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:
		// 订阅来源，需要基于traceID查询
		query := entryInfo.EntryID
		if utils.IsSubscribe(int(entryInfo.EntryType)) {
			query = node.TraceID
		}
		apiLogsInput, err := aliyun.MultiThemeModelOutRequestQuery(ctx, query, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiThemeModelOutResponseQuery(ctx, query, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		// 订阅来源，需要基于traceID查询
		query := entryInfo.EntryID
		if utils.IsSubscribe(int(entryInfo.EntryType)) {
			query = node.TraceID
		}
		apiLogsInput, err := aliyun.MultiOutlineModelOutRequestQuery(ctx, query, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiOutlineModelOutResponseQuery(ctx, query, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_RETRY_FINISH:
		apiLogsInput, err := aliyun.AbstractModelOutRequestQueryByTraceID(ctx, node.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.AbstractModelOutResponseQueryByTraceID(ctx, node.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_RETRY_FINISH,
		empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_RETRY_FINISH,
		empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_RETRY_FINISH:
		apiLogsInput, err := aliyun.OutlineModelOutRequestQueryByTraceID(ctx, node.TraceID, entryInfo.UserID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		newApiLogsInput := []aliyun.FileProcessLog{}
		for _, log := range apiLogsInput {
			// 解压缩
			inputStr := GetReqRespFromMsg(log.Message, "req:")
			if inputStr == "" {
				inputStr = log.Message
			}
			if node.Type == empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_RETRY_FINISH {
				if strings.Contains(inputStr, "\"verbose\":true") {
					newApiLogsInput = append(newApiLogsInput, log)
				}
			} else {
				if strings.Contains(inputStr, "\"verbose\":false") {
					newApiLogsInput = append(newApiLogsInput, log)
				}
			}
		}
		apiLogsOuput, err := aliyun.OutlineModelOutResponseQuery(ctx, node.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		newApiLogsOuput := []aliyun.FileProcessLog{}
		if len(newApiLogsInput) > 0 {
			for _, log := range apiLogsOuput {
				if log.OperationID == newApiLogsInput[0].OperationID {
					newApiLogsOuput = append(newApiLogsOuput, log)
				}
			}
		}
		return getReqAndResp(ctx, entryInfo, node, newApiLogsInput, newApiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH:
		apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, node.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, node.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_RETRY_FINISH:
		apiLogsInput, err := aliyun.MultiOutlineModelOutRequestQuery(ctx, node.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineModelOutRequestQuery] get process logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiOutlineModelOutResponseQuery(ctx, node.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineModelOutResponseQuery] get process logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	case empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH:
		apiLogsInput, err := aliyun.NovelFormOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NovelFormOutRequestQuery] get process logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.NovelFormOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NovelFormOutResponseQuery] get process logs failed, err: %v", err)
			return "", nil, &consts.QueryRecordError
		}
		return getReqAndResp(ctx, entryInfo, node, apiLogsInput, apiLogsOuput)
	}
	return "", []*empyrean_lens.ApiLog{}, nil
}

func getReqAndResp(ctx context.Context, entryInfo *bi.EntryInfo, node *empyrean_lens.GraphNode, apiLogsInput, apiLogsOuput []aliyun.FileProcessLog) (string, []*empyrean_lens.ApiLog, *consts.BizCode) {
	apiLogs := []*empyrean_lens.ApiLog{}
	if node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD {
		return "", []*empyrean_lens.ApiLog{}, nil
	}
	if len(apiLogsInput) == 0 && len(apiLogsOuput) == 0 && node.TraceID == "" {
		return "", []*empyrean_lens.ApiLog{}, nil
	}
	// 错误和安全日志
	errLogs, safeLogs := getErrorAndSafeLogs(ctx, entryInfo.MultiID, entryInfo.EntryID, node, apiLogsInput)
	// 排序
	sort.Slice(apiLogsInput, func(i, j int) bool {
		return apiLogsInput[i].Asctime.Before(apiLogsInput[j].Asctime)
	})
	sort.Slice(apiLogsOuput, func(i, j int) bool {
		return apiLogsOuput[i].Asctime.Before(apiLogsOuput[j].Asctime)
	})
	// 遍历
	outputUsedIdx := map[int]struct{}{}
	for _, input := range apiLogsInput {
		output := aliyun.FileProcessLog{
			Asctime: input.Asctime,
		}
		for idx, apiLogOuput := range apiLogsOuput {
			if _, ok := outputUsedIdx[idx]; ok {
				continue
			}
			if apiLogOuput.TraceId == input.TraceId && !apiLogOuput.Asctime.Before(input.Asctime) && apiLogOuput.OperationID == input.OperationID {
				output = apiLogOuput
				if node.EnterTime == node.FinishTime {
					node.EnterTime = input.Asctime.Format(consts.DateTimeTemplate)
					node.FinishTime = output.Asctime.Format(consts.DateTimeTemplate)
				}
				outputUsedIdx[idx] = struct{}{}
				break
			}
		}
		// 输入输出
		inputStr := GetReqRespFromMsg(input.Message, "req:")
		if inputStr == "" {
			inputStr = input.Message
		}
		outputStr := GetReqRespFromMsg(output.Message, "resp:")
		if outputStr == "" {
			outputStr = output.Message
		}
		apiLog := &empyrean_lens.ApiLog{
			HTTPCode:    200,
			EnterTime:   input.Asctime.Format(consts.DateTimeTemplate),
			FinishTime:  output.Asctime.Format(consts.DateTimeTemplate),
			Input:       inputStr,
			Output:      outputStr,
			TraceID:     input.TraceId,
			OperationID: input.OperationID,
		}
		// wcd 记录 oss链接
		if node.Type == empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH {
			outputJson := map[string]interface{}{}
			if splitList := strings.Split(output.Message, "resp:"); len(splitList) > 1 {
				resp := strings.Split(output.Message, "resp:")[1]
				err := json.Unmarshal([]byte(resp), &outputJson)
				if err == nil {
					// 获取oss列表
					key, bucket := "", ""
					if _, ok := outputJson["oss_info"]; ok {
						ossInfo := outputJson["oss_info"].(map[string]interface{})
						if value, ok := ossInfo["key"]; ok {
							key = value.(string)
						}
						if value, ok := ossInfo["bucket"]; ok {
							bucket = value.(string)
						}
					}
					// 获取oss文件
					if key != "" && bucket != "" {
						ossOp := tools.GetOssOperator(ctx)
						file, err := ossOp.DownloadWcdOssFile(bucket, key)
						if err == nil {
							// 更新输出
							outputJson["raw_html"] = file.RawHtml
							outputJson["parsed_html"] = file.ParsedHtml
							outputJson["model_result"] = file.TextParserLabels
							outputJson["model_input_str"] = file.ModelInput
							// 删除map中的字段
							delete(outputJson, "model_result_str")
							delete(outputJson, "model_input_df")
							delete(outputJson, "readable_html")
							outputStr, _ := json.Marshal(outputJson)
							apiLog.Output = string(outputStr)
						}
					}
				}
			}
		}
		// edu 解析输出特殊处理，asicII 转 字符串
		if node.Type == empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH {
			apiLogOutput := eduOutputTranslate(apiLog.Output)
			if apiLogOutput != "" {
				apiLog.Output = apiLogOutput
			}
		}
		apiLogs = append(apiLogs, apiLog)
	}
	// 错误日志
	if len(apiLogsInput) > 0 {
		node.EnterTime = apiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
	}
	for _, errLog := range errLogs {
		if node.Status != empyrean_lens.ActionStatusEnum_SUCCESS && (node.Type == empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH || node.Type == empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH || errLog.EnterTime >= node.EnterTime) {
			apiLogs = append(apiLogs, errLog)
		}
	}
	// 安全日志
	for _, safeLog := range safeLogs {
		if safeLog.EnterTime >= node.EnterTime {
			apiLogs = append(apiLogs, safeLog)
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
	traceIDMapping := map[string]struct{}{}
	for _, apiLog := range apiLogsInput {
		if apiLog.TraceId != "" {
			traceIDMapping[apiLog.TraceId] = struct{}{}
		}
	}
	if len(traceIDMapping) == 0 && node.TraceID != "" {
		traceIDMapping[node.TraceID] = struct{}{}
	}
	// 多文档大纲，需要获取所有traceID
	if node.Type == empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH {
		traceLogs, err := aliyun.MultiOutlineErrorTraceQuery(ctx, multiID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineErrorTraceQuery] get err logs failed, err: %v", err)
			return []*empyrean_lens.ApiLog{}, []*empyrean_lens.ApiLog{}
		}
		for _, log := range traceLogs {
			traceIDMapping[log.TraceId] = struct{}{}
		}
	}
	// 没有traceID，直接返回
	if len(traceIDMapping) == 0 {
		return []*empyrean_lens.ApiLog{}, []*empyrean_lens.ApiLog{}
	}
	// 并发获取错误日志、安全日志
	wg, errMapping, safeMapping := sync.WaitGroup{}, sync.Map{}, sync.Map{}
	wg.Add(len(traceIDMapping) * 2)
	for key := range traceIDMapping {
		traceID := key
		// 获取error日志
		go func() {
			defer wg.Done()
			var err error
			var apiLogsError []aliyun.FileProcessLog
			if multiID != "" {
				apiLogsError, err = aliyun.MultiTraceIDErrorQuery(ctx, traceID, start, end)
				if len(apiLogsError) == 0 && node.Type == empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH {
					apiLogsError, err = aliyun.PDFParserFcErrorQuery(ctx, entryID, start, end)
				}
			} else {
				apiLogsError, err = aliyun.SingleTraceIDErrorQuery(ctx, traceID, start, end)
				if len(apiLogsError) == 0 && node.Type == empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH {
					apiLogsError, err = aliyun.PDFParserFcErrorQuery(ctx, entryID, start, end)
				}
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
				HTTPCode:      500,
				EnterTime:     apiLogError.Asctime.Format(consts.DateTimeTemplate),
				FinishTime:    apiLogError.Asctime.Format(consts.DateTimeTemplate),
				ErrorMsg:      apiLogError.Message,
				TraceID:       apiLogError.TraceId,
				ContainerName: apiLogError.ContainerName,
				OperationID:   apiLogError.OperationID,
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
				HTTPCode:      500,
				EnterTime:     apiLogError.Asctime.Format(consts.DateTimeTemplate),
				FinishTime:    apiLogError.Asctime.Format(consts.DateTimeTemplate),
				ErrorMsg:      apiLogError.Message,
				TraceID:       apiLogError.TraceId,
				ContainerName: apiLogError.ContainerName,
				OperationID:   apiLogError.OperationID,
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

func DeleteFailLogs(logs []*empyrean_lens.ApiLog) []*empyrean_lens.ApiLog {
	newLogs := []*empyrean_lens.ApiLog{}
	for _, log := range logs {
		if log.ErrorMsg == "" {
			newLogs = append(newLogs, log)
		}
	}
	return newLogs
}

// 日志按照重试分组
func GroupLogsByRetry(status empyrean_lens.ActionStatusEnum, logs []*empyrean_lens.ApiLog) []*empyrean_lens.ApiLogGroup {
	// 过滤输入输出
	inputLogs := []*empyrean_lens.ApiLog{}
	for _, log := range logs {
		if log.Input != "" {
			inputLogs = append(inputLogs, log)
		}
	}
	// 输入输出按照时间正序排序
	sort.Slice(inputLogs, func(i, j int) bool {
		return inputLogs[i].EnterTime < inputLogs[j].EnterTime
	})
	groups := []*empyrean_lens.ApiLogGroup{}
	// 按照输入分组将错误日志分组
	if len(inputLogs) > 0 {
		// 输入的traceID是否都不同
		inputTraceIDMapping := map[string]struct{}{}
		for _, input := range inputLogs {
			inputTraceIDMapping[input.TraceID] = struct{}{}
		}
		// 先按照traceID分组
		groupLogsMapping := map[int][]*empyrean_lens.ApiLog{}
		if len(inputTraceIDMapping) == len(inputLogs) {
			usedLogsIdx := map[int]struct{}{}
			for i := 0; i < len(inputLogs); i++ {
				groupLogs := []*empyrean_lens.ApiLog{}
				// 成功的节点，最后一次成功，所以没有错误日志
				if status != empyrean_lens.ActionStatusEnum_SUCCESS || i != len(inputLogs)-1 {
					for idx, log := range logs {
						if log.ErrorMsg != "" && log.TraceID == inputLogs[i].TraceID {
							groupLogs = append(groupLogs, log)
							usedLogsIdx[idx] = struct{}{}
						}
					}
				}
				groupLogsMapping[i] = groupLogs
			}
			// 过滤未使用的
			reaminLogs := []*empyrean_lens.ApiLog{}
			for idx := range logs {
				if _, ok := usedLogsIdx[idx]; !ok {
					reaminLogs = append(reaminLogs, logs[idx])
				}
			}
			logs = reaminLogs
		}
		// 再按照时间对剩余的日志分组
		for i := 0; i < len(inputLogs); i++ {
			groupLogs := groupLogsMapping[i]
			start, end := inputLogs[i].EnterTime, time.Now().Format(consts.DateTimeTemplate)
			if i < len(inputLogs)-1 {
				end = inputLogs[i+1].EnterTime
			}
			// 成功的节点，最后一次成功，所以没有错误日志
			if status != empyrean_lens.ActionStatusEnum_SUCCESS || i != len(inputLogs)-1 {
				for _, log := range logs {
					if log.ErrorMsg != "" && log.EnterTime >= start && log.EnterTime < end {
						groupLogs = append(groupLogs, log)
					}
				}
			}
			// 按照时间排序
			sort.Slice(groupLogs, func(i, j int) bool {
				return groupLogs[i].EnterTime < groupLogs[j].EnterTime
			})
			groupLogs = append([]*empyrean_lens.ApiLog{inputLogs[i]}, groupLogs...)
			groups = append(groups, &empyrean_lens.ApiLogGroup{
				Idx:  int32(i + 1),
				Logs: groupLogs,
			})
		}
	} else {
		// 没有输入输出，按照traceID分组
		traceIDs := []string{}
		traceMapping := map[string][]*empyrean_lens.ApiLog{}
		for _, log := range logs {
			if _, ok := traceMapping[log.TraceID]; !ok {
				traceIDs = append(traceIDs, log.TraceID)
			}
			traceMapping[log.TraceID] = append(traceMapping[log.TraceID], log)
		}
		for idx, traceID := range traceIDs {
			groups = append(groups, &empyrean_lens.ApiLogGroup{
				Idx:  int32(idx + 1),
				Logs: traceMapping[traceID],
			})
		}
	}
	return groups
}
