package link_trace

import (
	"context"
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

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Save(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string) *consts.BizCode {
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
	if entryInfo != nil && (entryInfo.LinkStatus == int(empyrean_lens.ActionStatusEnum_SUCCESS)) {
		return &consts.ResSuccess
	}
	// 根据类型保存数据库
	switch entryType {
	case empyrean_lens.EntryTypeEnum_FILE:
		return SaveFile(ctx, entryID)
	case empyrean_lens.EntryTypeEnum_WEB:
		return SaveWebReader(ctx, entryID)
	case empyrean_lens.EntryTypeEnum_MULTI:
		return SaveMulti(ctx, entryID)
	default:
		return &consts.RetParamError
	}
}

func SaveWebReader(ctx context.Context, entryID string) *consts.BizCode {
	// 获取node列表
	webReaderInfo, linkTrace, bizCode := WebReaderLinkTrace(ctx, entryID, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	// 获取node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.LinkGraph.Nodes {
		logData, bizCode := WebReaderNodeLogs(ctx, node.Type, entryID, nil, true)
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

func SaveFile(ctx context.Context, entryID string) *consts.BizCode {
	// 获取node列表
	fileInfo, linkTrace, bizCode := FileLinkTrace(ctx, entryID, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	// 获取node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.LinkGraph.Nodes {
		logData, bizCode := FileNodeLogs(ctx, node.Type, entryID, nil, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
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

func SaveMulti(ctx context.Context, entryID string) *consts.BizCode {
	// 获取node列表
	multiInfo, linkTrace, bizCode := MultiLinkTrace(ctx, entryID, true)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
		return bizCode
	}
	// 获取multi node日志
	nodeLogMapping := map[string]*empyrean_lens.LinkNodeLogRespData{}
	for _, node := range linkTrace.Graph.Nodes {
		logData, bizCode := MultiNodeLogs(ctx, node.Type, entryID, nil, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get link trace failed, entry_id:%s, err: %v", entryID, bizCode)
			return bizCode
		}
		nodeLogMapping[node.ID] = logData
	}
	// 保存子文档信息
	for _, article := range multiInfo.ArticleList {
		if article.EntryType == consts.EntryTypeWEB {
			SaveWebReader(ctx, article.EntryId)
		} else {
			SaveFile(ctx, article.EntryId)
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

func MultiSaveToMongo(ctx context.Context, multiInfo *plugin.MultiModel, linkTrace *empyrean_lens.MultiDocLinkTraceRespData, nodeLogMapping map[string]*empyrean_lens.LinkNodeLogRespData) *consts.BizCode {
	// 获取entryInfo
	entryInfo := makeEntryInfo(ctx, empyrean_lens.EntryTypeEnum_MULTI, multiInfo, linkTrace)
	// 获取entryAction
	nodes := []*empyrean_lens.GraphNode{}
	for _, article := range linkTrace.Articles {
		nodes = append(nodes, article.Graph.Nodes...)
	}
	nodes = append(nodes, linkTrace.Graph.Nodes...)
	entryActions := makeEntryActions(entryInfo, nodes, nodeLogMapping)
	// 记录错误原因
	for _, node := range nodes {
		if node.Status != empyrean_lens.ActionStatusEnum_FAIL {
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
	entryInfo, nodes := &bi.EntryInfo{}, []*empyrean_lens.GraphNode{}
	switch entryType {
	case empyrean_lens.EntryTypeEnum_WEB:
		webReaderInfo := articleInfo.(*plugin.WebReader)
		linkTrace := linkTrace.(*empyrean_lens.DocLinkTraceRespData)
		nodes = linkTrace.LinkGraph.Nodes
		entryInfo = webReaderInfo.TranslateEntryInfo()
	case empyrean_lens.EntryTypeEnum_FILE:
		fileInfo := articleInfo.(*plugin.File)
		linkTrace := linkTrace.(*empyrean_lens.DocLinkTraceRespData)
		nodes = linkTrace.LinkGraph.Nodes
		entryInfo = fileInfo.TranslateEntryInfo()
	case empyrean_lens.EntryTypeEnum_MULTI:
		multiInfo := articleInfo.(*plugin.MultiModel)
		linkTrace := linkTrace.(*empyrean_lens.MultiDocLinkTraceRespData)
		nodes = linkTrace.Graph.Nodes
		for _, article := range linkTrace.Articles {
			nodes = append(nodes, article.Graph.Nodes...)
		}
		entryInfo = multiInfo.TranslateEntryInfo()
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
	// web端上传的，需要考虑模型生成
	if !utils.IsWebChannel(entryInfo.ChannelType) {
		newNodes := []*empyrean_lens.GraphNode{}
		for _, node := range nodes {
			if !utils.Contains([]empyrean_lens.LinkNodeTypeEnum{
				empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
				empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
				empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
			}, node.Type) {
				newNodes = append(newNodes, node)
			}
		}
		nodes = newNodes
	}
	// 计数耗时
	start, end := "", ""
	for _, node := range nodes {
		if start == "" || (node.EnterTime != "" && strings.Compare(start, node.EnterTime) > 0) {
			start = node.EnterTime
		}
		if node.FinishTime != "" && strings.Compare(end, node.FinishTime) < 0 {
			end = node.FinishTime
		}
	}
	if start != "" && end != "" {
		startTime, _ := time.Parse(consts.DateTimeTemplate, start)
		endTime, _ := time.Parse(consts.DateTimeTemplate, end)
		entryInfo.Cost = int(endTime.Sub(startTime).Milliseconds())
	}
	// 获取状态，失败原因
	entryInfo.LinkStatus = int(empyrean_lens.ActionStatusEnum_SUCCESS)
	for _, node := range nodes {
		if node.Status == empyrean_lens.ActionStatusEnum_FAIL {
			entryInfo.FailedAction = node.Name
			entryInfo.LinkStatus = int(empyrean_lens.ActionStatusEnum_FAIL)
			break
		}
		if node.Status == empyrean_lens.ActionStatusEnum_WORTHLESS {
			entryInfo.FailedAction = node.Name
			entryInfo.LinkStatus = int(empyrean_lens.ActionStatusEnum_WORTHLESS)
			break
		}
		if node.Status == empyrean_lens.ActionStatusEnum_LENGTH_ERROR {
			entryInfo.FailedAction = node.Name
			entryInfo.LinkStatus = int(empyrean_lens.ActionStatusEnum_LENGTH_ERROR)
			break
		}
	}
	// 是否是拷贝来的
	if entryType == empyrean_lens.EntryTypeEnum_WEB || entryType == empyrean_lens.EntryTypeEnum_FILE {
		if entryInfo.LinkStatus == int(empyrean_lens.ActionStatusEnum_FAIL) {
			if len(nodes) > 2 && nodes[1].Status != empyrean_lens.ActionStatusEnum_FAIL {
				entryInfo.ParentEntryID = ""
			}
		}
	}
	return entryInfo
}

func makeEntryActions(entryInfo *bi.EntryInfo, nodes []*empyrean_lens.GraphNode, nodeLogMapping map[string]*empyrean_lens.LinkNodeLogRespData) []*bi.EntryAction {
	processList := consts.MultiProcessList
	if entryInfo.EntryType != int(empyrean_lens.EntryTypeEnum_MULTI) {
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
			startTime, _ := time.Parse(consts.DateTimeTemplate, node.EnterTime)
			endTime, _ := time.Parse(consts.DateTimeTemplate, node.FinishTime)
			// 节点日志
			actionIOs := []*bi.ActionIO{}
			traceIDMapping := map[string]struct{}{}
			for _, nodeLog := range nodeLogMapping[node.ID].Logs {
				if nodeLog.ErrorMsg != "" {
					continue
				}
				input, _ := utillib.StrGzip(nodeLog.Input)
				output, _ := utillib.StrGzip(nodeLog.Output)
				actionIO := &bi.ActionIO{
					TraceID:      nodeLog.TraceID,
					ActionInput:  input,
					ActionOutput: output,
					ActionError:  []any{},
				}
				for _, errNodeLog := range nodeLogMapping[node.ID].Logs {
					if errNodeLog.ErrorMsg != "" && errNodeLog.TraceID == nodeLog.TraceID {
						actionIO.ActionError = append(actionIO.ActionError, errNodeLog.ErrorMsg)
						traceIDMapping[errNodeLog.TraceID] = struct{}{}
					}
				}
				actionIOs = append(actionIOs, actionIO)
			}
			for _, errNodeLog := range nodeLogMapping[node.ID].Logs {
				if _, ok := traceIDMapping[errNodeLog.TraceID]; !ok && errNodeLog.ErrorMsg != "" {
					errorMsgList := []any{errNodeLog.ErrorMsg}
					for _, errNodeLog2 := range nodeLogMapping[node.ID].Logs {
						if errNodeLog2.TraceID == errNodeLog.TraceID {
							errorMsgList = append(errorMsgList, errNodeLog2.ErrorMsg)
						}
					}
					actionIOs = append(actionIOs, &bi.ActionIO{
						TraceID:      errNodeLog.TraceID,
						ActionInput:  "",
						ActionOutput: "",
						ActionError:  errorMsgList,
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
	articleList := []*bi.EntryInfo{}
	switch entryType {
	case empyrean_lens.EntryTypeEnum_WEB:
		webReaderInfos, err := plugin.NewWebReaderDao().FindWebReaderByTimeRangeForSave(ctx, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get web reader infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, webReaderInfo := range webReaderInfos {
			articleList = append(articleList, webReaderInfo.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_FILE:
		fileInfos, err := plugin.NewFileDao().FindFileByTimeRangeForSave(ctx, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get file infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, fileInfo := range fileInfos {
			articleList = append(articleList, fileInfo.TranslateEntryInfo())
		}
	case empyrean_lens.EntryTypeEnum_MULTI:
		multiInfos, err := plugin.NewMultiDao().FindMultiByTimeRangeForSave(ctx, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get multi infos failed, err: %v", err)
			return &consts.QueryRecordError
		}
		for _, multiInfo := range multiInfos {
			articleList = append(articleList, multiInfo.TranslateEntryInfo())
		}
	}
	// 上报消息队列
	msgList := []plugin.ArticleEntry{}
	for _, article := range articleList {
		msgList = append(msgList, plugin.ArticleEntry{
			EntryId:   article.EntryID,
			EntryType: consts.EntryType(article.EntryType),
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
	return nil
}
