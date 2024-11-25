package link_trace

import (
	"context"
	"strings"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func LinkTrace(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string) (interface{}, interface{}, *consts.BizCode) {
	switch entryType {
	case empyrean_lens.EntryTypeEnum_FILE:
		return FileLinkTrace(ctx, entryID, false)
	case empyrean_lens.EntryTypeEnum_WEB:
		return WebReaderLinkTrace(ctx, entryID, false)
	case empyrean_lens.EntryTypeEnum_MULTI:
		return MultiLinkTrace(ctx, entryID, false)
	default:
		return nil, nil, &consts.RetParamError
	}
}

func FileLinkTrace(ctx context.Context, fileID string, refresh bool) (*plugin.File, *empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	fileInfo, err := plugin.NewFileDao().FindFileById(ctx, fileID)
	if err != nil || fileInfo == nil {
		hlog.CtxErrorf(ctx, "get file info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 确定需要查的节点列表和节点关系
	var pracessList []empyrean_lens.LinkNodeTypeEnum
	var pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum
	if fileInfo.MultiId != "" {
		pracessList = consts.MultiFileProcessList
		pracessMapping = consts.MultiFileProcessMapping
	} else if utils.ChannelIntToString(fileInfo.ChannelType) == "语鲸小助手" || utils.ChannelIntToString(fileInfo.ChannelType) == "语鲸小程序" {
		if fileInfo.CopyFromFildID != "" {
			pracessList = consts.SinglePluginCopiedFileProcessList
			pracessMapping = consts.SinglePluginCopiedFileProcessMapping
		} else {
			pracessList = consts.SinglePluginFileProcessList
			pracessMapping = consts.SinglePluginFileProcessMapping
		}
	} else {
		if fileInfo.CopyFromFildID != "" {
			pracessList = consts.SingleCopiedFileProcessList
			pracessMapping = consts.SingleCopiedFileProcessMapping
		} else {
			pracessList = consts.SingleFileProcessList
			pracessMapping = consts.SingleFileProcessMapping
		}
	}
	// 先查数据库
	linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_FILE), fileID, pracessList, pracessMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	// 数据库没有，查阿里云日志
	if refresh || linkTraceGraph == nil {
		start := fileInfo.CreateTime.Add(-1 * time.Hour)
		end := fileInfo.CreateTime.Add(24 * time.Hour)
		fileInfo.TranslateEntryInfo()
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, fileInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
	}
	// 返回
	return fileInfo, &empyrean_lens.DocLinkTraceRespData{
		LinkGraph: linkTraceGraph,
		Cost:      getLinkTraceCost(linkTraceGraph.Nodes),
		EntryID:   fileID,
		EntryType: empyrean_lens.EntryTypeEnum_FILE,
		Title:     "",
	}, nil
}

func WebReaderLinkTrace(ctx context.Context, webReaderID string, refresh bool) (*plugin.WebReader, *empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, webReaderID)
	if err != nil || webReaderInfo == nil {
		hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 确定需要查的节点列表和节点关系
	var pracessList []empyrean_lens.LinkNodeTypeEnum
	var pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum
	if webReaderInfo.MultiId != "" {
		pracessList = consts.MultiWebReaderProcessList
		pracessMapping = consts.MultiWebReaderProcessMapping
	} else if utils.ChannelIntToString(webReaderInfo.ChannelType) == "语鲸插件" {
		pracessList = consts.SinglePluginWebReaderProcessList
		pracessMapping = consts.SinglePluginWebReaderProcessMapping
	} else if utils.ChannelIntToString(webReaderInfo.ChannelType) == "语鲸web" {
		pracessList = consts.SingleWebReaderProcessList
		pracessMapping = consts.SingleWebReaderProcessMapping
	} else {
		pracessList = consts.SingleMiniWebReaderProcessList
		pracessMapping = consts.SingleMiniWebReaderProcessMapping
	}
	// 先查数据库
	linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_WEB), webReaderID, pracessList, pracessMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	// 数据库没有，查阿里云日志
	if refresh || linkTraceGraph == nil {
		start := webReaderInfo.CreateTime.Add(-24 * time.Hour)
		end := webReaderInfo.CreateTime.Add(24 * time.Hour)
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, webReaderInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
	}
	// 返回
	return webReaderInfo, &empyrean_lens.DocLinkTraceRespData{
		LinkGraph: linkTraceGraph,
		Cost:      getLinkTraceCost(linkTraceGraph.Nodes),
		EntryID:   webReaderID,
		EntryType: empyrean_lens.EntryTypeEnum_WEB,
		Title:     "",
	}, nil
}

func MultiLinkTrace(ctx context.Context, multiID string, refresh bool) (*plugin.MultiModel, *empyrean_lens.MultiDocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	multiInfo, err := plugin.NewMultiDao().FindMultiById(ctx, multiID)
	if err != nil || multiInfo == nil {
		hlog.CtxErrorf(ctx, "get multi info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 并发链路信息
	wg, graphMapping, multiGrap := sync.WaitGroup{}, sync.Map{}, &empyrean_lens.TraceLinkGraph{}
	wg.Add(len(multiInfo.ArticleList) + 1)
	for idx := range multiInfo.ArticleList {
		articleEntry := multiInfo.ArticleList[idx]
		go func() {
			defer wg.Done()
			if articleEntry.EntryType == consts.EntryTypePDF {
				_, articleGraph, bizCode := FileLinkTrace(ctx, articleEntry.EntryId, refresh)
				if bizCode != nil {
					hlog.CtxErrorf(ctx, "[FileLinkTrace] get article graph failed, err: %v", bizCode)
					return
				}
				graphMapping.Store(articleEntry.EntryId, articleGraph.LinkGraph)
			} else {
				_, articleGraph, bizCode := WebReaderLinkTrace(ctx, articleEntry.EntryId, refresh)
				if bizCode != nil {
					hlog.CtxErrorf(ctx, "[WebReaderLinkTrace] get article graph failed, err: %v", bizCode)
					return
				}
				graphMapping.Store(articleEntry.EntryId, articleGraph.LinkGraph)
			}
		}()
	}
	go func() {
		defer wg.Done()
		pracessList, pracessMapping := consts.MultiProcessList, consts.MultiProcessMapping
		linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_MULTI), multiInfo.ID.Hex(), pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
			return
		}
		// 数据库没有，查阿里云日志
		if refresh || linkTraceGraph == nil {
			start := multiInfo.CreateTime.Add(-24 * time.Hour)
			end := multiInfo.UpdateTime.Add(24 * time.Hour)
			linkTraceGraph, bizCode = LinkTraceGraph(ctx, multiInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
			if bizCode != nil {
				hlog.CtxErrorf(ctx, "[LinkTraceGraph] get article graph failed, err: %v", bizCode)
				return
			}
		}
		multiGrap = linkTraceGraph
	}()
	wg.Wait()
	// 整理数据
	graphs, articles := []*empyrean_lens.TraceLinkGraph{}, []*empyrean_lens.Article{}
	for _, article := range multiInfo.ArticleList {
		articleGraph, ok := graphMapping.Load(article.EntryId)
		if ok && articleGraph != nil {
			graph := articleGraph.(*empyrean_lens.TraceLinkGraph)
			graphs = append(graphs, graph)
			articles = append(articles, &empyrean_lens.Article{
				EntryType: empyrean_lens.EntryTypeEnum(article.EntryType),
				EntryID:   article.EntryId,
				StartID:   articleGraph.(*empyrean_lens.TraceLinkGraph).Nodes[0].ID,
				Graph:     graph,
			})
		}
	}
	linkTraceGraph := mergeLinkTraceGraph(graphs, multiGrap)
	// 返回
	return multiInfo, &empyrean_lens.MultiDocLinkTraceRespData{
		Graph:    multiGrap,
		Cost:     getLinkTraceCost(linkTraceGraph.Nodes),
		Articles: articles,
		Title:    "",
	}, nil
}

func findLinkTraceFromMongo(ctx context.Context, entryType int, entryID string,
	pracessList []empyrean_lens.LinkNodeTypeEnum, pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) (*empyrean_lens.TraceLinkGraph, *consts.BizCode) {
	// 先查数据库
	entryActions, err := bi.NewEntryActionDao().FindByEntryTypeEntryID(ctx, entryType, entryID)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryTypeEntryID] get entry actions failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	if len(entryActions) == 0 {
		return nil, nil
	}
	// 整理数据
	nodes := []*empyrean_lens.GraphNode{}
	nodeMappingNew := map[empyrean_lens.LinkNodeTypeEnum]*empyrean_lens.GraphNode{}
	for _, action := range entryActions {
		actionType := empyrean_lens.LinkNodeTypeEnum(action.ActionType)
		if _, ok := nodeMappingNew[actionType]; !ok {
			nodeMappingNew[actionType] = action.TranslateGraphNode()
			nodes = append(nodes, nodeMappingNew[actionType])
		}
	}
	edges := map[empyrean_lens.NodeId][]empyrean_lens.NodeId{}
	for pracessType, pracessTypeList := range pracessMapping {
		node1, ok := nodeMappingNew[pracessType]
		if !ok || node1 == nil {
			continue
		}
		edges[node1.ID] = []empyrean_lens.NodeId{}
		for _, itemType := range pracessTypeList {
			node2, ok := nodeMappingNew[itemType]
			if !ok || node2 == nil {
				continue
			}
			edges[node1.ID] = append(edges[node1.ID], node2.ID)
		}
	}
	// 返回
	return &empyrean_lens.TraceLinkGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// 获取链路追踪图
func LinkTraceGraph(ctx context.Context, entryInfo *bi.EntryInfo, start, end time.Time,
	pracessList []empyrean_lens.LinkNodeTypeEnum, pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) (*empyrean_lens.TraceLinkGraph, *consts.BizCode) {
	// 并发获取节点日志
	wg, nodeMapping := sync.WaitGroup{}, sync.Map{}
	wg.Add(len(pracessList))
	for idx := range pracessList {
		pracessType := pracessList[idx]
		go func(pracessType empyrean_lens.LinkNodeTypeEnum) {
			defer wg.Done()
			node, err := GetProcessNode(ctx, pracessType, entryInfo, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[GetProcessNode] get process logs failed, err: %v", err)
				return
			}
			if node != nil {
				nodeMapping.Store(pracessType, node)
			}
		}(pracessType)
	}
	wg.Wait()
	// 整理数据
	nodes := []*empyrean_lens.GraphNode{}
	nodeMappingNew := map[empyrean_lens.LinkNodeTypeEnum]*empyrean_lens.GraphNode{}
	for _, pracessType := range pracessList {
		value, ok := nodeMapping.Load(pracessType)
		if !ok || value == nil {
			continue
		}
		node := value.(*empyrean_lens.GraphNode)
		nodes = append(nodes, node)
		nodeMappingNew[pracessType] = node
	}
	edges := map[empyrean_lens.NodeId][]empyrean_lens.NodeId{}
	for pracessType, pracessTypeList := range pracessMapping {
		node1, ok := nodeMappingNew[pracessType]
		if !ok || node1 == nil {
			continue
		}
		edges[node1.ID] = []empyrean_lens.NodeId{}
		for _, itemType := range pracessTypeList {
			node2, ok := nodeMappingNew[itemType]
			if !ok || node2 == nil {
				continue
			}
			edges[node1.ID] = append(edges[node1.ID], node2.ID)
		}
	}
	// 修改状态，子节点成功，父节点也要成功
	hasFailedNode := false
	for _, pracessType := range pracessList {
		if node, ok := nodeMappingNew[pracessType]; ok && node != nil {
			if isFatherFail(node, nodes, edges) {
				node.Status = empyrean_lens.ActionStatusEnum_UNREACHEAD
				node.EnterTime = ""
				node.FinishTime = ""
				hasFailedNode = true
				continue
			}
			if hasFailedNode {
				node.Status = empyrean_lens.ActionStatusEnum_UNREACHEAD
				node.EnterTime = ""
				node.FinishTime = ""
				continue
			}
			if node.Status == empyrean_lens.ActionStatusEnum_WORTHLESS {
				hasFailedNode = true
				continue
			}
			if node.Status != empyrean_lens.ActionStatusEnum_SUCCESS && node.Status != empyrean_lens.ActionStatusEnum_FAIL && isChildSuccess(node, nodes, edges) {
				node.Status = empyrean_lens.ActionStatusEnum_SUCCESS
			}
			if node.Type != empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH && node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD &&
				isFatherSuccess(node, nodes, edges) && isChildAllUnReachead(node, nodes, edges) {
				hasFailedNode = true
				node.Status = empyrean_lens.ActionStatusEnum_FAIL
				node.EnterTime = ""
				node.FinishTime = ""
			}
			if node.Type == empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH && node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD &&
				isFatherSuccess(node, nodes, edges) {
				hasFailedNode = true
				node.Status = empyrean_lens.ActionStatusEnum_FAIL
				node.EnterTime = ""
				node.FinishTime = ""
			}
		}
	}
	return &empyrean_lens.TraceLinkGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// 获取节点日志
func GetProcessNode(ctx context.Context, processType empyrean_lens.LinkNodeTypeEnum, entryInfo *bi.EntryInfo, start, end time.Time) (*empyrean_lens.GraphNode, *consts.BizCode) {
	switch processType {
	case empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:
		processLogs, err := aliyun.ResourceUploadQuery(ctx, entryInfo.EntryID, consts.EntryTypeMap[entryInfo.EntryType], start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[ResourceUploadQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		processLogs, err := aliyun.CrawlerQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[CrawlerQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		// 没有抓取日志，使用输入输出兜底
		if len(processLogs) == 0 {
			apiLogsOuput, err := aliyun.CrawlerOutResponseQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			return processLogsToNode(processType, apiLogsOuput), nil
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		processLogs, err := aliyun.WcdParseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[WcdParseQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		// 没有使用输入输出兜底
		if len(processLogs) == 0 {
			apiLogsInput, err := aliyun.WcdOutRequestQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[WcdOutRequestQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(apiLogsInput) > 0 {
				errorLogs, err := aliyun.SingleTraceIDErrorQuery(ctx, apiLogsInput[0].TraceId, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[TraceIDErrorQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				return processLogsToNode(processType, append(apiLogsInput, errorLogs...)), nil
			}
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		processLogs, err := aliyun.PDFParserQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[PDFParserQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:
		apiLogsInput, err := aliyun.TextParseOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TextParseOutRequestQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.TextParseOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TextParseOutResponseQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		processLogs := apiLogsOuput
		if len(apiLogsInput) > 0 && len(apiLogsOuput) > 0 {
			processLogs[0].Cost = float64(apiLogsOuput[0].Asctime.Sub(apiLogsInput[0].Asctime).Seconds())
			return processLogsToNode(processType, processLogs), nil
		}
		if len(apiLogsInput) > 0 && len(apiLogsOuput) == 0 {
			if entryInfo.MultiID != "" {
				errLogs, err := aliyun.MultiTraceIDErrorQuery(ctx, apiLogsInput[0].TraceId, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[MultiTraceIDErrorQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				processLogs = append(apiLogsInput, errLogs...)
			} else {
				errLogs, err := aliyun.SingleTraceIDErrorQuery(ctx, apiLogsInput[0].TraceId, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[SingleTraceIDErrorQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				processLogs = append(apiLogsInput, errLogs...)
			}
		}
		node := processLogsToNode(processType, processLogs)
		if len(apiLogsInput) > 0 {
			node.TraceID = apiLogsInput[0].TraceId
		}
		return node, nil
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		var err error
		var processLogs []aliyun.FileProcessLog
		if entryInfo.MultiID != "" {
			processLogs, err = aliyun.MultiEduParseQuery(ctx, entryInfo.EntryID, start, end)
		} else {
			processLogs, err = aliyun.SingleEduParseQuery(ctx, entryInfo.EntryID, start, end)
		}
		if err != nil {
			hlog.CtxErrorf(ctx, "[EduParseQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
		processLogs1, err := aliyun.SingleViewpointBeginQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[SingleViewpointBeginQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(processLogs1) != 0 {
			processLogs2, err := aliyun.SingleViewpointEndQuery(ctx, processLogs1[0].TraceId, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[SingleViewpointEndQuery] get process logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(processLogs2) > 0 {
				return processLogsToNode(processType, append([]aliyun.FileProcessLog{processLogs1[0]}, processLogs2...)), nil
			}
		}
		// 插件没有传文章ID，导致匹配不上，使用输入输出兜底
		apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		processLogs := []aliyun.FileProcessLog{}
		if len(apiLogsInput) > 0 && len(apiLogsOuput) > 0 {
			processLogs = append(processLogs, apiLogsInput[0])
			for _, apiLogOuput := range apiLogsOuput {
				if apiLogOuput.TraceId == apiLogsInput[0].TraceId {
					processLogs = append([]aliyun.FileProcessLog{apiLogsInput[0]}, apiLogOuput)
					break
				}
			}
		}
		// 兜底，长度不够
		if len(processLogs) == 0 && entryInfo.ContentSize < 100 {
			node := makeEmptyNode(processType)
			node.Status = empyrean_lens.ActionStatusEnum_LENGTH_ERROR
			return node, nil
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
		processLogs1, err := aliyun.SingleOverviewBeginQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[SingleOverviewBeginQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(processLogs1) != 0 {
			processLogs2, err := aliyun.SingleOverviewEndQuery(ctx, processLogs1[0].TraceId, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[SingleOverviewEndQuery] get process logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(processLogs2) > 0 {
				return processLogsToNode(processType, append([]aliyun.FileProcessLog{processLogs1[0]}, processLogs2...)), nil
			}
		}
		// 插件没有传文章ID，导致匹配不上，使用输入输出兜底
		apiLogsInput, err := aliyun.AbstractModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[AbstractModelOutRequestQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.AbstractModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[AbstractModelOutResponseQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		processLogs := []aliyun.FileProcessLog{}
		if len(apiLogsInput) > 0 && len(apiLogsOuput) > 0 {
			processLogs = append(processLogs, apiLogsInput[0])
			for _, apiLogOuput := range apiLogsOuput {
				if apiLogOuput.TraceId == apiLogsInput[0].TraceId {
					processLogs = append([]aliyun.FileProcessLog{apiLogsInput[0]}, apiLogOuput)
					break
				}
			}
		}
		// 兜底，长度不够
		if len(processLogs) == 0 && entryInfo.ContentSize < 100 {
			node := makeEmptyNode(processType)
			node.Status = empyrean_lens.ActionStatusEnum_LENGTH_ERROR
			return node, nil
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:
		apiLogsInput, err := aliyun.OutlineModelOutRequestQuery(ctx, entryInfo.EntryID, entryInfo.UserID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[OutlineModelOutRequestQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.OutlineModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[OutlineModelOutResponseQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		processLogs := []aliyun.FileProcessLog{}
		if len(apiLogsInput) > 0 && len(apiLogsOuput) > 0 {
			processLogs = append(processLogs, apiLogsInput[0])
			for _, apiLogOuput := range apiLogsOuput {
				if apiLogOuput.TraceId == apiLogsInput[0].TraceId {
					processLogs = append([]aliyun.FileProcessLog{apiLogsInput[0]}, apiLogOuput)
					break
				}
			}
		}
		// 兜底，长度不够
		if len(processLogs) == 0 && entryInfo.ContentSize < 1000 {
			node := makeEmptyNode(processType)
			node.Status = empyrean_lens.ActionStatusEnum_LENGTH_ERROR
			return node, nil
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH:
		processLogs, err := aliyun.MultiItemAnalysisQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiAnalysisQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:
		processLogs, err := aliyun.MultiThemeQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiThemeQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		processLogs, err := aliyun.MultiOutlineQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		// 为空，使用错误日志兜底
		if len(processLogs) == 0 {
			traceLogs, err := aliyun.MultiOutlineErrorTraceQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[MultiOutlineErrorTraceQuery] get err logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(traceLogs) != 0 {
				processLogs, err = aliyun.MultiTraceIDErrorQuery(ctx, traceLogs[0].TraceId, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[MultiOutlineErrorTraceQuery] get err logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
			}
		}
		return processLogsToNode(processType, processLogs), nil
	}
	return makeEmptyNode(processType), nil
}

func processLogsToNode(nodeType empyrean_lens.LinkNodeTypeEnum, processLogs []aliyun.FileProcessLog) *empyrean_lens.GraphNode {
	if len(processLogs) == 0 {
		return makeEmptyNode(nodeType)
	}
	if utils.InSlice(consts.LinkNodeTypeName[nodeType], []string{"概述生成", "关键信息生成", "大纲生成"}) && len(processLogs) != 0 {
		length := len(processLogs)
		enterTime := processLogs[0].Asctime
		if processLogs[length-1].Cost != 0 {
			enterTime = processLogs[length-1].Asctime.Add(-time.Millisecond * time.Duration(processLogs[length-1].Cost*1000))
		}
		return &empyrean_lens.GraphNode{
			ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
			Type:       nodeType,
			Name:       consts.LinkNodeTypeName[nodeType],
			EnterTime:  enterTime.Format(consts.DateTimeTemplate),
			FinishTime: processLogs[length-1].Asctime.Format(consts.DateTimeTemplate),
			Status:     getActionStatus(nodeType, []aliyun.FileProcessLog{processLogs[length-1]}),
			TraceID:    processLogs[length-1].TraceId,
		}
	}
	enterTime := processLogs[0].Asctime.Add(-time.Millisecond * time.Duration(processLogs[0].Cost*1000))
	return &empyrean_lens.GraphNode{
		ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
		Type:       nodeType,
		Name:       consts.LinkNodeTypeName[nodeType],
		EnterTime:  enterTime.Format(consts.DateTimeTemplate),
		FinishTime: processLogs[0].Asctime.Format(consts.DateTimeTemplate),
		Status:     getActionStatus(nodeType, processLogs),
		TraceID:    processLogs[0].TraceId,
	}
}

func mergeLinkTraceGraph(headers []*empyrean_lens.TraceLinkGraph, tail *empyrean_lens.TraceLinkGraph) *empyrean_lens.TraceLinkGraph {
	nodes := make([]*empyrean_lens.GraphNode, 0)
	edges := make(map[empyrean_lens.NodeId][]empyrean_lens.NodeId, 0)
	for _, header := range headers {
		nodes = append(nodes, header.Nodes...)
		for _, node := range header.Nodes {
			if _, ok := header.Edges[node.ID]; ok {
				edges[node.ID] = header.Edges[node.ID]
			} else {
				edges[node.ID] = make([]empyrean_lens.NodeId, 0)
				edges[node.ID] = append(edges[node.ID], tail.Nodes[0].ID)
			}
		}
	}
	nodes = append(nodes, tail.Nodes...)
	for key, value := range tail.Edges {
		edges[key] = value
	}
	// 修改状态，子节点成功，父节点也要成功
	for _, node := range nodes {
		if node.Status != empyrean_lens.ActionStatusEnum_SUCCESS && isChildSuccess(node, nodes, edges) {
			node.Status = empyrean_lens.ActionStatusEnum_SUCCESS
		}
		if node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD && isFatherAllSuccess(node, nodes, edges) && isChildAllUnReachead(node, nodes, edges) {
			node.Status = empyrean_lens.ActionStatusEnum_FAIL
		}
	}
	return &empyrean_lens.TraceLinkGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

func getLinkTraceCost(nodes []*empyrean_lens.GraphNode) float64 {
	startAt, endAt := "", ""
	for _, node := range nodes {
		if startAt == "" || strings.Compare(startAt, node.EnterTime) > 0 {
			startAt = node.EnterTime
		}
		if endAt == "" || strings.Compare(endAt, node.FinishTime) < 0 {
			endAt = node.FinishTime
		}
	}
	startAtT, _ := time.Parse(consts.DateTimeTemplate, startAt)
	endAtT, _ := time.Parse(consts.DateTimeTemplate, endAt)
	return endAtT.Sub(startAtT).Seconds()
}

func makeEmptyNode(nodeType empyrean_lens.LinkNodeTypeEnum) *empyrean_lens.GraphNode {
	status := empyrean_lens.ActionStatusEnum_UNREACHEAD
	if nodeType == empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH {
		status = empyrean_lens.ActionStatusEnum_SUCCESS
	}
	return &empyrean_lens.GraphNode{
		ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
		Name:       consts.LinkNodeTypeName[nodeType],
		Type:       nodeType,
		EnterTime:  "",
		FinishTime: "",
		Status:     status,
	}
}

func isFatherSuccess(node *empyrean_lens.GraphNode, nodes []*empyrean_lens.GraphNode, nodeIDMapping map[empyrean_lens.NodeId][]empyrean_lens.NodeId) bool {
	nodeMapping := map[empyrean_lens.NodeId]*empyrean_lens.GraphNode{}
	for _, node := range nodes {
		nodeMapping[node.ID] = node
	}
	hasFather := false
	for fID, ids := range nodeIDMapping {
		for _, id := range ids {
			if id == node.ID {
				hasFather = true
				return nodeMapping[fID].Status == empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
	}
	return !hasFather
}

func isFatherFail(node *empyrean_lens.GraphNode, nodes []*empyrean_lens.GraphNode, nodeIDMapping map[empyrean_lens.NodeId][]empyrean_lens.NodeId) bool {
	nodeMapping := map[empyrean_lens.NodeId]*empyrean_lens.GraphNode{}
	for _, node := range nodes {
		nodeMapping[node.ID] = node
	}
	hasFather := false
	for fID, ids := range nodeIDMapping {
		for _, id := range ids {
			if id == node.ID {
				hasFather = true
				return nodeMapping[fID].Status == empyrean_lens.ActionStatusEnum_FAIL || nodeMapping[fID].Status == empyrean_lens.ActionStatusEnum_WORTHLESS
			}
		}
	}
	return hasFather
}

func isFatherAllSuccess(node *empyrean_lens.GraphNode, nodes []*empyrean_lens.GraphNode, nodeIDMapping map[empyrean_lens.NodeId][]empyrean_lens.NodeId) bool {
	nodeMapping := map[empyrean_lens.NodeId]*empyrean_lens.GraphNode{}
	for _, node := range nodes {
		nodeMapping[node.ID] = node
	}
	for fID, ids := range nodeIDMapping {
		for _, id := range ids {
			if id == node.ID && nodeMapping[fID].Status != empyrean_lens.ActionStatusEnum_SUCCESS {
				return false
			}
		}
	}
	return true
}

func isChildSuccess(node *empyrean_lens.GraphNode, nodes []*empyrean_lens.GraphNode, nodeIDMapping map[empyrean_lens.NodeId][]empyrean_lens.NodeId) bool {
	if _, ok := nodeIDMapping[node.ID]; !ok {
		return false
	}
	nodeMapping := map[empyrean_lens.NodeId]*empyrean_lens.GraphNode{}
	for _, node := range nodes {
		nodeMapping[node.ID] = node
	}
	// 广度优先遍历
	nodeIds := nodeIDMapping[node.ID]
	for len(nodeIds) > 0 {
		newNodeIds := []empyrean_lens.NodeId{}
		for _, id := range nodeIds {
			if nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_FAIL && nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_WORTHLESS && nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_UNREACHEAD && nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_LENGTH_ERROR {
				return true
			}
			newNodeIds = append(newNodeIds, nodeIDMapping[id]...)
		}
		nodeIds = newNodeIds
	}
	return false
}

func isChildAllUnReachead(node *empyrean_lens.GraphNode, nodes []*empyrean_lens.GraphNode, nodeIDMapping map[empyrean_lens.NodeId][]empyrean_lens.NodeId) bool {
	if _, ok := nodeIDMapping[node.ID]; !ok {
		return false
	}
	nodeMapping := map[empyrean_lens.NodeId]*empyrean_lens.GraphNode{}
	for _, node := range nodes {
		nodeMapping[node.ID] = node
	}
	// 广度优先遍历
	nodeIds := nodeIDMapping[node.ID]
	for len(nodeIds) > 0 {
		newNodeIds := []empyrean_lens.NodeId{}
		for _, id := range nodeIds {
			if nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_UNREACHEAD && nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_LENGTH_ERROR {
				return false
			}
			newNodeIds = append(newNodeIds, nodeIDMapping[id]...)
		}
		nodeIds = newNodeIds
	}
	return true
}

func getActionStatus(nodeType empyrean_lens.LinkNodeTypeEnum, processLogs []aliyun.FileProcessLog) empyrean_lens.ActionStatusEnum {
	if len(processLogs) == 0 {
		return empyrean_lens.ActionStatusEnum_UNREACHEAD
	}
	switch nodeType {
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		if strings.Contains(processLogs[0].Message, "成功") {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		if strings.Contains(processLogs[0].Message, "wcd text nil") || strings.Contains(processLogs[0].Message, "wcd worthless") {
			return empyrean_lens.ActionStatusEnum_WORTHLESS
		}
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "WcdRaw do req error") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "苏秦解析异常") || strings.Contains(processLog.Message, "pdf解析异常") || strings.Contains(processLog.Message, "parsing file failed") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:
		for _, processLog := range processLogs {
			if (strings.Contains(processLog.Message, "parse_edu") && strings.Contains(processLog.Message, "error")) || strings.Contains(processLog.Message, "ParseEdu error") || strings.Contains(processLog.Message, "edu parse error") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		for _, processLog := range processLogs {
			if (strings.Contains(processLog.Message, "parse_edu") && strings.Contains(processLog.Message, "error")) || strings.Contains(processLog.Message, "ParseEdu error") || strings.Contains(processLog.Message, "edu parse error") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH, empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:
		for _, processLog := range processLogs {
			if !strings.Contains(processLog.Message, "OutRequest") && (strings.Contains(processLog.Message, "error") || strings.Contains(processLog.Message, "fail")) {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
			if strings.Contains(processLog.Message, "OutRequest") && strings.Contains(processLog.Message, "resp:") {
				// 解码
				msg := GetReqRespFromMsg(processLog.Message, "resp:")
				if !strings.Contains(msg, "\\\"code\\\": 0") && !strings.Contains(msg, "data too long") {
					return empyrean_lens.ActionStatusEnum_FAIL
				}
				if strings.Contains(msg, "data too long") {
					return empyrean_lens.ActionStatusEnum_SUCCESS
				}
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "不安全") || strings.Contains(processLog.Message, "core error") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	}
	return empyrean_lens.ActionStatusEnum_SUCCESS
}
