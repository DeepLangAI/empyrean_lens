package link_trace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/dal/redis"
	"empyrean_lens/tools"
	"empyrean_lens/utils"
	"empyrean_lens/utils/gse"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Save(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string, refresh bool) *consts.BizCode {
	// 加锁，防止并发
	lockKey := fmt.Sprintf("link_trace:save:%s", entryID)
	err := redis.KeySetNx(ctx, lockKey, redis.Stop, time.Duration(3)*time.Minute)
	if err != nil {
		return &consts.RetParamError
	}
	defer redis.DelKey(ctx, lockKey)
	// 查询数据库
	entryInfo, err := bi.NewEntryInfoDao().FindByEntryIDAndEntryType(ctx, entryID, int(entryType))
	if err != nil {
		hlog.CtxErrorf(ctx, "get entry info failed, entry_id:%s, err: %v", entryID, err)
		return &consts.QueryRecordError
	}
	// 已经保存过，并且状态为成功，直接返回
	if !refresh && entryInfo != nil && (entryInfo.LinkStatus == int(empyrean_lens.ActionStatusEnum_SUCCESS)) {
		return &consts.ResSuccess
	}
	// 根据类型保存数据库
	switch entryType {
	case empyrean_lens.EntryTypeEnum_FILE:
		return SaveFile(ctx, entryID, "")
	case empyrean_lens.EntryTypeEnum_WEB:
		return SaveWebReader(ctx, entryID, "")
	case empyrean_lens.EntryTypeEnum_MULTI:
		return SaveMulti(ctx, entryID)
	case empyrean_lens.EntryTypeEnum_SUMMARY:
		return SaveSummary(ctx, entryType, entryID)
	case empyrean_lens.EntryTypeEnum_OUTLINE:
		return SaveSummary(ctx, entryType, entryID)
	case empyrean_lens.EntryTypeEnum_VIEWPOINT:
		return SaveSummary(ctx, entryType, entryID)
	case empyrean_lens.EntryTypeEnum_MULTI_OUTLINE:
		return SaveMultiOutline(ctx, entryType, entryID)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB:
		return SaveSubscribeSingle(ctx, entryType, entryID, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE:
		return SaveSubscribeSingle(ctx, entryType, entryID, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI:
		return SaveSubscribeMulti(ctx, entryType, entryID)
	default:
		return &consts.RetParamError
	}
}

func SaveWebReader(ctx context.Context, entryID, multiID string) *consts.BizCode {
	// 获取node列表
	webReaderInfo, linkTrace, bizCode := WebReaderLinkTrace(ctx, entryID, multiID, true, false)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	// 保存parent
	entryInfo := webReaderInfo.TranslateEntryInfo()
	if entryInfo.ParentEntryID != "" {
		Save(ctx, empyrean_lens.EntryTypeEnum(entryInfo.ParentEntryType), entryInfo.ParentEntryID, false)
		webReaderInfo, linkTrace, bizCode = WebReaderLinkTrace(ctx, entryID, multiID, true, false)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
	}
	// 获取node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.LinkGraph.Nodes {
		logData, bizCode := WebReaderNodeLogs(ctx, node.Type, entryID, node, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
		nodeLogMapping[node.ID] = logData
	}
	// 保存到数据库
	bizCode = SingleSaveToMongo(ctx, webReaderInfo, linkTrace, nodeLogMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	return nil
}

func SaveFile(ctx context.Context, entryID, multiID string) *consts.BizCode {
	// 获取node列表
	fileInfo, linkTrace, bizCode := FileLinkTrace(ctx, entryID, multiID, true, false)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	// 保存parent
	entryInfo := fileInfo.TranslateEntryInfo()
	if entryInfo.ParentEntryID != "" {
		Save(ctx, empyrean_lens.EntryTypeEnum(entryInfo.ParentEntryType), entryInfo.ParentEntryID, false)
		fileInfo, linkTrace, bizCode = FileLinkTrace(ctx, entryID, multiID, true, false)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
	}
	// 获取node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.LinkGraph.Nodes {
		logData, bizCode := FileNodeLogs(ctx, node.Type, entryID, node, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get node log failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
		nodeLogMapping[node.ID] = logData
	}
	// 保存到数据库
	bizCode = SingleSaveToMongo(ctx, fileInfo, linkTrace, nodeLogMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	return nil
}

func SaveSubscribeSingle(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string, isFromMulti bool) *consts.BizCode {
	// 获取node列表
	resourceInfo, linkTrace, bizCode := SubscribeSingleLinkTrace(ctx, entryType, entryID, isFromMulti, true, false)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	resourceInfo.UserID = "resource_server"
	// 获取node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.LinkGraph.Nodes {
		logData, bizCode := SubscribeNodeLogs(ctx, node.Type, entryType, entryID, node, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get node log failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
		nodeLogMapping[node.ID] = logData
	}
	// 保存到数据库
	bizCode = SingleSaveToMongo(ctx, resourceInfo, linkTrace, nodeLogMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	return nil
}

func SaveMulti(ctx context.Context, entryID string) *consts.BizCode {
	// 获取node列表
	multiInfo, linkTrace, bizCode := MultiLinkTrace(ctx, entryID, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	// 保存parent
	entryInfo := multiInfo.TranslateEntryInfo()
	if entryInfo.ParentEntryID != "" {
		Save(ctx, empyrean_lens.EntryTypeEnum(entryInfo.ParentEntryType), entryInfo.ParentEntryID, false)
	}
	// 获取multi node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.Graph.Nodes {
		logData, bizCode := MultiNodeLogs(ctx, node.Type, entryID, node, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get node log failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
		nodeLogMapping[node.ID] = logData
	}
	// 保存子文档信息
	for _, article := range multiInfo.ArticleList {
		if article.EntryType == consts.EntryTypeWEB {
			SaveWebReader(ctx, article.EntryId, multiInfo.ID.Hex())
		} else {
			SaveFile(ctx, article.EntryId, multiInfo.ID.Hex())
		}
	}
	// 保存多文档信息到数据库
	bizCode = MultiSaveToMongo(ctx, multiInfo, linkTrace, nodeLogMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	return nil
}

func SaveSubscribeMulti(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string) *consts.BizCode {
	// 获取node列表
	resourceInfo, linkTrace, bizCode := SubscriMultibeLinkTrace(ctx, entryType, entryID, true, false)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	resourceInfo.UserID = "resource_server"
	// 获取multi node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.Graph.Nodes {
		logData, bizCode := SubscribeNodeLogs(ctx, node.Type, entryType, entryID, node, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get node log failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
		nodeLogMapping[node.ID] = logData
	}
	// 保存子文档信息
	for _, article := range resourceInfo.ArticleList {
		SaveSubscribeSingle(ctx, empyrean_lens.EntryTypeEnum(article.EntryType), article.EntryId, true)
	}
	// 保存多文档信息到数据库
	bizCode = MultiSaveToMongo(ctx, resourceInfo, linkTrace, nodeLogMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	return nil
}

func SaveSummary(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string) *consts.BizCode {
	// 获取node列表
	summaryInfo, linkTrace, bizCode := SummaryTrace(ctx, entryType, entryID, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	if linkTrace == nil || len(linkTrace.LinkGraph.Nodes) == 0 {
		return nil
	}
	// 保存parent
	if summaryInfo.SourceEntryID != "" {
		Save(ctx, empyrean_lens.EntryTypeEnum(summaryInfo.SourceEntryType), summaryInfo.SourceEntryID, false)
	}
	// 去除非summary节点
	lenNodes := len(linkTrace.LinkGraph.Nodes)
	linkTrace.LinkGraph.Nodes = linkTrace.LinkGraph.Nodes[lenNodes-1:]
	linkTrace.LinkGraph.Edges = map[string][]string{}
	node := linkTrace.LinkGraph.Nodes[0]
	// 获取node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	logData, bizCode := SummaryNodeLogs(ctx, node.Type, entryType, entryID, node, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get node log failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	nodeLogMapping[node.ID] = logData
	// 保存到数据库
	bizCode = SummarySaveToMongo(ctx, summaryInfo, node, nodeLogMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	return nil
}

func SaveMultiOutline(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string) *consts.BizCode {
	// 获取node列表
	aigcInfo, linkTrace, bizCode := MultiOutlineLinkTrace(ctx, entryType, entryID, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	if aigcInfo == nil || linkTrace == nil {
		return nil
	}
	// 保存parent
	if aigcInfo.MultiID != "" {
		Save(ctx, empyrean_lens.EntryTypeEnum_MULTI, aigcInfo.MultiID, false)
	}
	// 去除非summary节点
	lenNodes := len(linkTrace.Graph.Nodes)
	linkTrace.Graph.Nodes = linkTrace.Graph.Nodes[lenNodes-1:]
	linkTrace.Graph.Edges = map[string][]string{}
	node := linkTrace.Graph.Nodes[0]
	// 获取node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	logData, bizCode := MultiOutlineNodeLogs(ctx, node.Type, entryType, entryID, node, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get node log failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	nodeLogMapping[node.ID] = logData
	// 保存到数据库
	bizCode = MultiOutLineSaveToMongo(ctx, aigcInfo, node, nodeLogMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	return nil
}

func SingleSaveToMongo(ctx context.Context, articleInfo interface{}, linkTrace *empyrean_lens.DocLinkTraceRespData, nodeLogMapping map[string]*empyrean_lens.LinkNodeLogRespData) *consts.BizCode {
	// 获取entryInfo
	entryInfo := makeEntryInfo(ctx, linkTrace.EntryType, articleInfo, linkTrace)
	// 获取entryAction
	entryActions := makeEntryActions(entryInfo, linkTrace.LinkGraph.Nodes, nodeLogMapping)
	// 记录错误原因
	for _, node := range linkTrace.LinkGraph.Nodes {
		if node.Status == empyrean_lens.ActionStatusEnum_FAIL {
			entryInfo.FailedAction = node.Name
			break
		}
	}
	hlog.CtxDebugf(ctx, "entryInfo: %+v", len(entryActions))
	// 保存到数据库
	err := bi.NewEntryInfoDao().SaveEntryInfo(ctx, entryInfo)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	err = bi.NewEntryActionDao().SaveBatchEntryAction(ctx, entryActions)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry action failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	return nil
}

func SummarySaveToMongo(ctx context.Context, summaryInfo *plugin.Summary, node *empyrean_lens.GraphNode, nodeLogMapping map[string]*empyrean_lens.LinkNodeLogRespData) *consts.BizCode {
	// 获取entryInfo
	entryInfo := makeEntryInfo(ctx, empyrean_lens.EntryTypeEnum(summaryInfo.EntryType), summaryInfo, node)
	// 获取entryAction
	entryActions := makeEntryActions(entryInfo, []*empyrean_lens.GraphNode{node}, nodeLogMapping)
	if node.Status == empyrean_lens.ActionStatusEnum_FAIL {
		entryInfo.FailedAction = node.Name
	}
	hlog.CtxDebugf(ctx, "entryInfo: %+v", len(entryActions))
	// 保存到数据库
	err := bi.NewEntryInfoDao().SaveEntryInfo(ctx, entryInfo)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	err = bi.NewEntryActionDao().SaveBatchEntryAction(ctx, entryActions)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry action failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	return nil
}

func MultiOutLineSaveToMongo(ctx context.Context, aigcInfo *plugin.MultiAigc, node *empyrean_lens.GraphNode, nodeLogMapping map[string]*empyrean_lens.LinkNodeLogRespData) *consts.BizCode {
	// 获取entryInfo
	entryInfo := makeEntryInfo(ctx, empyrean_lens.EntryTypeEnum(empyrean_lens.EntryTypeEnum_MULTI_OUTLINE), aigcInfo, node)
	// 获取entryAction
	entryActions := makeEntryActions(entryInfo, []*empyrean_lens.GraphNode{node}, nodeLogMapping)
	if node.Status == empyrean_lens.ActionStatusEnum_FAIL {
		entryInfo.FailedAction = node.Name
	}
	hlog.CtxDebugf(ctx, "entryInfo: %+v", len(entryActions))
	// 保存到数据库
	err := bi.NewEntryInfoDao().SaveEntryInfo(ctx, entryInfo)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	err = bi.NewEntryActionDao().SaveBatchEntryAction(ctx, entryActions)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry action failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	return nil
}

func MultiSaveToMongo(ctx context.Context, multiInfo interface{}, linkTrace *empyrean_lens.MultiDocLinkTraceRespData, nodeLogMapping map[string]*empyrean_lens.LinkNodeLogRespData) *consts.BizCode {
	// 获取entryInfo
	entryInfo := makeEntryInfo(ctx, linkTrace.EntryType, multiInfo, linkTrace)
	// 获取entryAction
	entryActions := makeEntryActions(entryInfo, linkTrace.Graph.Nodes, nodeLogMapping)
	// 记录错误原因
	for _, node := range linkTrace.Graph.Nodes {
		if node.Status == empyrean_lens.ActionStatusEnum_FAIL {
			entryInfo.FailedAction = node.Name
			break
		}
	}
	hlog.CtxDebugf(ctx, "entryInfo: %+v", len(entryActions))
	// 保存到数据库
	err := bi.NewEntryInfoDao().SaveEntryInfo(ctx, entryInfo)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry info failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	err = bi.NewEntryActionDao().SaveBatchEntryAction(ctx, entryActions)
	if err != nil {
		hlog.CtxErrorf(ctx, "save entry action failed, entry_id:%s, err: %v", entryInfo.EntryID, err)
		return &consts.WriteDbError
	}
	return nil
}

func makeEntryInfo(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, articleInfo interface{}, linkTrace interface{}) *bi.EntryInfo {
	copyFromEntryId := ""
	entryInfo, nodes := &bi.EntryInfo{}, []*empyrean_lens.GraphNode{}
	switch entryType {
	case empyrean_lens.EntryTypeEnum_WEB:
		webReaderInfo := articleInfo.(*plugin.WebReader)
		linkTrace := linkTrace.(*empyrean_lens.DocLinkTraceRespData)
		nodes = linkTrace.LinkGraph.Nodes
		entryInfo = webReaderInfo.TranslateEntryInfo()
		copyFromEntryId = webReaderInfo.CopyFromUrlID
	case empyrean_lens.EntryTypeEnum_FILE:
		fileInfo := articleInfo.(*plugin.File)
		linkTrace := linkTrace.(*empyrean_lens.DocLinkTraceRespData)
		nodes = linkTrace.LinkGraph.Nodes
		entryInfo = fileInfo.TranslateEntryInfo()
		copyFromEntryId = fileInfo.CopyFromFildID
	case empyrean_lens.EntryTypeEnum_MULTI:
		multiInfo := articleInfo.(*plugin.MultiModel)
		linkTrace := linkTrace.(*empyrean_lens.MultiDocLinkTraceRespData)
		nodes = linkTrace.Graph.Nodes
		for _, article := range linkTrace.Articles {
			nodes = append(nodes, article.Graph.Nodes...)
		}
		entryInfo = multiInfo.TranslateEntryInfo()
		copyFromEntryId = multiInfo.CopyFromMultiID
	case empyrean_lens.EntryTypeEnum_SUMMARY,
		empyrean_lens.EntryTypeEnum_OUTLINE,
		empyrean_lens.EntryTypeEnum_VIEWPOINT:
		summaryInfo := articleInfo.(*plugin.Summary)
		nodes = []*empyrean_lens.GraphNode{linkTrace.(*empyrean_lens.GraphNode)}
		entryInfo = summaryInfo.TranslateEntryInfo()
		copyFromEntryId = summaryInfo.CopyFromSummaryID
	case empyrean_lens.EntryTypeEnum_MULTI_OUTLINE:
		aigcInfo := articleInfo.(*plugin.MultiAigc)
		nodes = []*empyrean_lens.GraphNode{linkTrace.(*empyrean_lens.GraphNode)}
		entryInfo = aigcInfo.TranslateEntryInfo()
		copyFromEntryId = aigcInfo.CopyFromResourceID
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB,
		empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE:
		resourceInfo := articleInfo.(*plugin.Resource)
		linkTrace := linkTrace.(*empyrean_lens.DocLinkTraceRespData)
		nodes = linkTrace.LinkGraph.Nodes
		entryInfo = resourceInfo.TranslateEntryInfo()
		entryInfo.UserID = "resource_server"
		copyFromEntryId = ""
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI:
		resourceInfo := articleInfo.(*plugin.Resource)
		linkTrace := linkTrace.(*empyrean_lens.MultiDocLinkTraceRespData)
		nodes = linkTrace.Graph.Nodes
		entryInfo = resourceInfo.TranslateEntryInfo()
		entryInfo.UserID = "resource_server"
		copyFromEntryId = ""
	}
	// 用户类型
	userInfos, err := bi.NewUserInfoDao().FindFileByUids(ctx, []string{entryInfo.UserID})
	if err != nil {
		hlog.CtxErrorf(ctx, "[makeEntryInfo] faind user type error: %+v", err)
	}
	for _, userInfo := range userInfos {
		if userInfo.UserID == entryInfo.UserID {
			entryInfo.UserType = int(userInfo.UserType)
		}
	}
	// 计数耗时
	entryInfo.Cost = utils.GetCostFromNodes(nodes)
	// 获取状态，失败原因
	fileAction, status := utils.GetStatusFromNode(nodes)
	entryInfo.FailedAction, entryInfo.LinkStatus = fileAction, int(status)
	// 是否是拷贝来的
	if entryType == empyrean_lens.EntryTypeEnum_WEB || entryType == empyrean_lens.EntryTypeEnum_FILE {
		if entryInfo.LinkStatus == int(empyrean_lens.ActionStatusEnum_FAIL) {
			if entryInfo.ParentEntryID == "" && len(nodes) > 2 && nodes[1].Status == empyrean_lens.ActionStatusEnum_FAIL {
				entryInfo.ParentEntryID = copyFromEntryId
			}
		}
	}
	if entryType == empyrean_lens.EntryTypeEnum_SUMMARY || entryType == empyrean_lens.EntryTypeEnum_OUTLINE || entryType == empyrean_lens.EntryTypeEnum_VIEWPOINT {
		if copyFromEntryId != "" {
			entryInfo.ParentEntryID = copyFromEntryId
		}
	}
	// content index
	contentList := []string{}
	contentList = append(contentList, entryInfo.UserID)
	contentList = append(contentList, entryInfo.Title)
	contentList = append(contentList, entryInfo.EntryID)
	contentList = append(contentList, entryInfo.EntryURL)
	contentList = append(contentList, entryInfo.Content)
	for _, article := range entryInfo.MultiArticles {
		contentList = append(contentList, article.EntryId)
	}
	tokens := gse.InitGse().CutTextV1(strings.Join(contentList, "\n"))
	entryInfo.ContentIndex = strings.Join(tokens, " ")
	return entryInfo
}

func makeEntryActions(entryInfo *bi.EntryInfo, nodes []*empyrean_lens.GraphNode, nodeLogMapping map[string]*empyrean_lens.LinkNodeLogRespData) []*bi.EntryAction {
	var processList []empyrean_lens.LinkNodeTypeEnum
	if entryInfo.EntryType == int(empyrean_lens.EntryTypeEnum_MULTI) {
		processList = consts.MultiProcessList
	} else if entryInfo.EntryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI) {
		processList = consts.SubscribeMultiProcessList
	} else {
		processList = consts.TotalProcessList
	}
	// 找到对应的日志
	entryActions := []*bi.EntryAction{}
	for _, nodeType := range processList {
		for _, node := range nodes {
			if node.Type != nodeType {
				continue
			}
			nodeID, _ := primitive.ObjectIDFromHex(node.ID)
			if node.Status == empyrean_lens.ActionStatusEnum_FAIL {
				node.EnterTime = ""
				node.FinishTime = ""
			}
			startTime, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
			endTime, _ := time.Parse(consts.DateTimeTemplate, node.FinishTime)
			// 节点日志
			actionIOs := []*bi.ActionIO{}
			traceIDMapping := map[string]struct{}{}
			// 输入输出
			for _, nodeLog := range nodeLogMapping[node.ID].Logs {
				if nodeLog.ErrorMsg != "" {
					continue
				}
				input, _ := utillib.StrGzip(nodeLog.Input)
				output, _ := utillib.StrGzip(nodeLog.Output)
				actionIO := &bi.ActionIO{
					TraceID:      nodeLog.TraceID,
					ActionInput:  input,
					InputAt:      nodeLog.EnterTime,
					OutputAt:     nodeLog.FinishTime,
					ActionOutput: output,
					ActionError:  []any{},
				}
				actionIOs = append(actionIOs, actionIO)
			}
			// 错误日志
			for _, errNodeLog := range nodeLogMapping[node.ID].Logs {
				if _, ok := traceIDMapping[errNodeLog.TraceID]; !ok && errNodeLog.ErrorMsg != "" {
					errorMsgList := []any{}
					for _, errNodeLog2 := range nodeLogMapping[node.ID].Logs {
						if errNodeLog2.TraceID == errNodeLog.TraceID && errNodeLog2.ErrorMsg != "" {
							logStr, _ := json.Marshal(errNodeLog2)
							errorMsgList = append(errorMsgList, string(logStr))
						}
					}
					actionIOs = append(actionIOs, &bi.ActionIO{
						TraceID:      errNodeLog.TraceID,
						ActionInput:  "",
						ActionOutput: "",
						ActionError:  errorMsgList,
						InputAt:      errNodeLog.EnterTime,
						OutputAt:     errNodeLog.FinishTime,
						OperationID:  errNodeLog.OperationID,
					})
					traceIDMapping[errNodeLog.TraceID] = struct{}{}
				}
			}
			entryActions = append(entryActions, &bi.EntryAction{
				ID:              nodeID,
				EntryID:         entryInfo.EntryID,
				ActionChannel:   entryInfo.EntryType,
				ActionType:      int(node.Type),
				ActionIOs:       actionIOs,
				ActionStatus:    int(node.Status),
				DataVersion:     consts.LingowhelaBiDataVersion,
				Cost:            int(endTime.Sub(startTime).Milliseconds()),
				ActionStartTime: startTime,
				ActionEndTime:   endTime,
				CreateTime:      time.Now(),
			})
		}
	}
	return entryActions
}

func BatchSave(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, beginAt, endAt int64) *consts.BizCode {
	// 开始时间，结束时间
	begin, end := time.Unix(beginAt, 0), time.Unix(endAt, 0)
	// 获取文件记录
	eventList := []*bi.EntryInfo{}
	switch entryType {
	case empyrean_lens.EntryTypeEnum_WEB:
		webReaderInfos, err := plugin.NewWebReaderDao().FindWebReaderByTimeRangeForSave(ctx, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get web reader infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, webReaderInfo := range webReaderInfos {
			eventList = append(eventList, webReaderInfo.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_FILE:
		fileInfos, err := plugin.NewFileDao().FindFileByTimeRangeForSave(ctx, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get file infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, fileInfo := range fileInfos {
			eventList = append(eventList, fileInfo.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_MULTI:
		multiInfos, err := plugin.NewMultiDao().FindMultiByTimeRangeForSave(ctx, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get multi infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, multiInfo := range multiInfos {
			eventList = append(eventList, multiInfo.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_SUMMARY:
		summaries, err := plugin.NewSummaryDao().FindSummaryByTimeRangeForSave(ctx, int(empyrean_lens.EntryTypeEnum_SUMMARY), begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get summary infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, summary := range summaries {
			eventList = append(eventList, summary.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_OUTLINE:
		summaries, err := plugin.NewSummaryDao().FindSummaryByTimeRangeForSave(ctx, int(empyrean_lens.EntryTypeEnum_OUTLINE), begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get outline infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, summary := range summaries {
			eventList = append(eventList, summary.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_VIEWPOINT:
		summaries, err := plugin.NewSummaryDao().FindSummaryByTimeRangeForSave(ctx, int(empyrean_lens.EntryTypeEnum_VIEWPOINT), begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get viewpoint infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, summary := range summaries {
			eventList = append(eventList, summary.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_MULTI_OUTLINE:
		multiAigcs, err := plugin.NewMultiAigcDao().FindByTimeRangeForSave(ctx, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get multi outline infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, multiAigc := range multiAigcs {
			eventList = append(eventList, multiAigc.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB:
		subscribeWebs, err := plugin.NewResourceDao().FindResourceByTimeRangeForSave(ctx, int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB), begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get subscribe web infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, subscribeWeb := range subscribeWebs {
			eventList = append(eventList, subscribeWeb.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE:
		subscribeFiles, err := plugin.NewResourceDao().FindResourceByTimeRangeForSave(ctx, int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE), begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get subscribe web infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, subscribeFile := range subscribeFiles {
			eventList = append(eventList, subscribeFile.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI:
		subscribeMultis, err := plugin.NewResourceDao().FindResourceByTimeRangeForSave(ctx, int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI), begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get subscribe multi infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, subscribeMulti := range subscribeMultis {
			eventList = append(eventList, subscribeMulti.TranslateEntryInfo())
		}
	}
	// 上报消息队列
	msgList := []bi.ArticleEntry{}
	for _, event := range eventList {
		msgList = append(msgList, bi.ArticleEntry{
			EntryId:   event.EntryID,
			EntryType: consts.EntryType(event.EntryType),
		})
	}
	sliceI := make([]interface{}, len(msgList))
	for i, val := range msgList {
		sliceI[i] = val
	}
	err := tools.BatchSendMsg(conf.GetConfig().MnsConfig.QueueName, sliceI)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch send msg failed, err: %v", err)
		return &consts.WriteDbError
	}
	// 删除content idx索引
	go func() {
		ctx := context.Background()
		// 加锁，防止并发
		lockKey := "link_trace:delete_content_idx"
		err := redis.KeySetNx(ctx, lockKey, redis.Stop, time.Duration(3)*time.Minute)
		if err != nil {
			hlog.CtxErrorf(ctx, "batch send msg failed, err: %v", err)
			return
		}
		defer redis.DelKey(ctx, lockKey)
		beginTime := time.Now().Add(-consts.ContentIdxExpire * time.Second)
		endTime := time.Now().Add((-consts.ContentIdxExpire + 3600) * time.Second)
		err = bi.NewEntryInfoDao().DeleteContentByTimeRange(ctx, beginTime, endTime)
		if err != nil {
			hlog.CtxErrorf(ctx, "delete content idx failed, err: %v", err)
		}
	}()
	return nil
}
