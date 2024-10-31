package link_trace

import (
	"context"
	"strings"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func LinkTrace(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string) (interface{}, *consts.BizCode) {
	switch entryType {
	case empyrean_lens.EntryTypeEnum_FILE:
		return FileLinkTrace(ctx, entryID)
	case empyrean_lens.EntryTypeEnum_WEB:
		return WebReaderLinkTrace(ctx, entryID)
	case empyrean_lens.EntryTypeEnum_MULTI:
		return MultiLinkTrace(ctx, entryID)
	default:
		return nil, &consts.RetParamError
	}
}

func FileLinkTrace(ctx context.Context, fileID string) (*empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	fileInfo, err := plugin.NewFileDao().FindFileById(ctx, fileID)
	if err != nil || fileInfo == nil {
		hlog.CtxErrorf(ctx, "get file info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	start := fileInfo.CreateTime.Add(-1 * time.Hour)
	end := fileInfo.CreateTime.Add(24 * time.Hour)
	// 并发获取节点列表
	linkTraceGraph, bizCode := &empyrean_lens.TraceLinkGraph{}, &consts.BizCode{}
	if fileInfo.MultiId != "" {
		fileGraph, bizCode := LinkTraceGraph(ctx, fileID, consts.PDF, start, end, consts.MultiFileProcessList, consts.MultiFileProcessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, bizCode
		}
		multi, bizCode := LinkTraceGraph(ctx, fileInfo.MultiId, consts.PDF, start, end, consts.MultiProcessList, consts.MultiProcessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, bizCode
		}
		linkTraceGraph = mergeLinkTraceGraph([]*empyrean_lens.TraceLinkGraph{fileGraph}, multi)
	} else {
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, fileID, consts.PDF, start, end, consts.SingleFileProcessList, consts.SingleFileProcessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, bizCode
		}
	}
	// 返回
	return &empyrean_lens.DocLinkTraceRespData{
		LinkGraph: linkTraceGraph,
		Cost:      getLinkTraceCost(linkTraceGraph.Nodes),
		EntryID:   fileID,
		EntryType: empyrean_lens.EntryTypeEnum_FILE,
		Title:     "",
	}, nil
}

func WebReaderLinkTrace(ctx context.Context, webReaderID string) (*empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, webReaderID)
	if err != nil || webReaderInfo == nil {
		hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	start := webReaderInfo.CreateTime.Add(-24 * time.Hour)
	end := webReaderInfo.CreateTime.Add(24 * time.Hour)
	// 并发获取节点列表
	linkTraceGraph, bizCode := &empyrean_lens.TraceLinkGraph{}, &consts.BizCode{}
	if webReaderInfo.MultiId != "" {
		webReaderGraph, bizCode := LinkTraceGraph(ctx, webReaderID, consts.URL, start, end, consts.MultiWebReaderProcessList, consts.MultiWebReaderProcessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, bizCode
		}
		multi, bizCode := LinkTraceGraph(ctx, webReaderInfo.MultiId, consts.URL, start, end, consts.MultiProcessList, consts.MultiProcessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, bizCode
		}
		linkTraceGraph = mergeLinkTraceGraph([]*empyrean_lens.TraceLinkGraph{webReaderGraph}, multi)
	} else {
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, webReaderID, consts.URL, start, end, consts.SingleWebReaderProcessList, consts.SingleWebReaderProcessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, bizCode
		}
	}
	// 返回
	return &empyrean_lens.DocLinkTraceRespData{
		LinkGraph: linkTraceGraph,
		Cost:      getLinkTraceCost(linkTraceGraph.Nodes),
		EntryID:   webReaderID,
		EntryType: empyrean_lens.EntryTypeEnum_WEB,
		Title:     "",
	}, nil
}

func MultiLinkTrace(ctx context.Context, multiID string) (*empyrean_lens.MultiDocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	multiInfo, err := plugin.NewMultiDao().FindMultiById(ctx, multiID)
	if err != nil || multiInfo == nil {
		hlog.CtxErrorf(ctx, "get multi info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	start := multiInfo.CreateTime.Add(-24 * time.Hour)
	end := multiInfo.CreateTime.Add(24 * time.Hour)
	// 并发获取节点列表
	wg, graphMapping, multiGrap := sync.WaitGroup{}, sync.Map{}, &empyrean_lens.TraceLinkGraph{}
	wg.Add(len(multiInfo.ArticleList) + 1)
	for idx := range multiInfo.ArticleList {
		articleEntry := multiInfo.ArticleList[idx]
		go func() {
			defer wg.Done()
			entrtId, entryType := articleEntry.EntryId, articleEntry.EntryType
			resourceType, processList, processMapping := consts.PDF, consts.MultiFileProcessList, consts.MultiFileProcessMapping
			if entryType == consts.EntryTypeWEB {
				resourceType, processList, processMapping = consts.URL, consts.MultiWebReaderProcessList, consts.MultiWebReaderProcessMapping
			}
			articleGraph, bizCode := LinkTraceGraph(ctx, entrtId, resourceType, start, end, processList, processMapping)
			if bizCode != nil {
				hlog.CtxErrorf(ctx, "[LinkTraceGraph] get article graph failed, err: %v", bizCode)
				return
			}
			graphMapping.Store(entrtId, articleGraph)
		}()
	}
	go func() {
		defer wg.Done()
		articleGraph, bizCode := LinkTraceGraph(ctx, multiID, consts.MULTI, start, end, consts.MultiProcessList, consts.MultiProcessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get article graph failed, err: %v", bizCode)
			return
		}
		multiGrap = articleGraph
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
	return &empyrean_lens.MultiDocLinkTraceRespData{
		Graph:    multiGrap,
		Cost:     getLinkTraceCost(linkTraceGraph.Nodes),
		Articles: articles,
		Title:    "",
	}, nil
}

// 获取链路追踪图
func LinkTraceGraph(ctx context.Context, resourceId, resourceType string, start, end time.Time,
	pracessList []empyrean_lens.LinkNodeTypeEnum, pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) (*empyrean_lens.TraceLinkGraph, *consts.BizCode) {
	// 并发获取节点日志
	wg, nodeMapping := sync.WaitGroup{}, sync.Map{}
	wg.Add(len(pracessList))
	for idx := range pracessList {
		pracessType := pracessList[idx]
		go func(pracessType empyrean_lens.LinkNodeTypeEnum) {
			defer wg.Done()
			node, err := GetProcessNode(ctx, pracessType, resourceId, resourceType, start, end)
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
	for _, pracessType := range pracessList {
		if node, ok := nodeMappingNew[pracessType]; ok && node != nil {
			if node.Status != empyrean_lens.ActionStatusEnum_SUCCESS && isChildSuccess(node, nodes, edges) {
				node.Status = empyrean_lens.ActionStatusEnum_SUCCESS
			}
			if node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD && isFatherSuccess(node, nodes, edges) && isChildAllUnReachead(node, nodes, edges) {
				node.Status = empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	}
	return &empyrean_lens.TraceLinkGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// 获取节点日志
func GetProcessNode(ctx context.Context, processType empyrean_lens.LinkNodeTypeEnum, resourceId, resourceType string, start, end time.Time) (*empyrean_lens.GraphNode, *consts.BizCode) {
	switch processType {
	case empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:
		processLogs, err := aliyun.ResourceUploadQuery(ctx, resourceId, resourceType, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[ResourceUploadQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		processLogs, err := aliyun.CrawlerQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[CrawlerQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		processLogs, err := aliyun.WcdParseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[WcdParseQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		processLogs, err := aliyun.PDFParserQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[PDFParserQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
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
		processLogs := apiLogsOuput
		if len(apiLogsInput) > 0 && len(apiLogsOuput) > 0 {
			processLogs[0].Cost = float64(apiLogsOuput[0].Asctime.Sub(apiLogsInput[0].Asctime).Seconds())
			return processLogsToNode(processType, processLogs), nil
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		processLogs, err := aliyun.EduParseQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[EduParseQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
		processLogs1, err := aliyun.SingleViewpointBeginQuery(ctx, resourceId, start, end)
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
			return processLogsToNode(processType, append(processLogs1, processLogs2...)), nil
		}
		return processLogsToNode(processType, []aliyun.FileProcessLog{}), nil
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
		processLogs1, err := aliyun.SingleOverviewBeginQuery(ctx, resourceId, start, end)
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
			return processLogsToNode(processType, append(processLogs1, processLogs2...)), nil
		}
		return processLogsToNode(processType, []aliyun.FileProcessLog{}), nil
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:
		processLogs1, err := aliyun.SingleOutlineBeginQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[SingleOutlineBeginQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(processLogs1) != 0 {
			processLogs2, err := aliyun.SingleOutlineEndQuery(ctx, processLogs1[0].TraceId, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[SingleOutlineEndQuery] get process logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			return processLogsToNode(processType, append(processLogs1, processLogs2...)), nil
		}
		return processLogsToNode(processType, []aliyun.FileProcessLog{}), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH:
		processLogs, err := aliyun.MultiItemAnalysisQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiAnalysisQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:
		processLogs, err := aliyun.MultiThemeQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiThemeQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		processLogs, err := aliyun.MultiOutlineQuery(ctx, resourceId, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
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
		enterTime := processLogs[length-1].Asctime.Add(-time.Millisecond * time.Duration(processLogs[length-1].Cost*1000))
		return &empyrean_lens.GraphNode{
			ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
			Type:       nodeType,
			Name:       consts.LinkNodeTypeName[nodeType],
			EnterTime:  enterTime.Format(consts.DateHourMinSecTemplate),
			FinishTime: processLogs[length-1].Asctime.Format(consts.DateHourMinSecTemplate),
			Status:     getActionStatus(nodeType, processLogs[length-1]),
			TraceID:    processLogs[length-1].TraceId,
		}
	}
	enterTime := processLogs[0].Asctime.Add(-time.Millisecond * time.Duration(processLogs[0].Cost*1000))
	return &empyrean_lens.GraphNode{
		ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
		Type:       nodeType,
		Name:       consts.LinkNodeTypeName[nodeType],
		EnterTime:  enterTime.Format(consts.DateHourMinSecTemplate),
		FinishTime: processLogs[0].Asctime.Format(consts.DateHourMinSecTemplate),
		Status:     getActionStatus(nodeType, processLogs[0]),
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
	startAtT, _ := time.Parse(consts.DateHourMinSecTemplate, startAt)
	endAtT, _ := time.Parse(consts.DateHourMinSecTemplate, endAt)
	return endAtT.Sub(startAtT).Seconds()
}

func makeEmptyNode(nodeType empyrean_lens.LinkNodeTypeEnum) *empyrean_lens.GraphNode {
	return &empyrean_lens.GraphNode{
		ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
		Name:       consts.LinkNodeTypeName[nodeType],
		Type:       nodeType,
		EnterTime:  "",
		FinishTime: "",
		Status:     empyrean_lens.ActionStatusEnum_UNREACHEAD,
	}
}

func isFatherSuccess(node *empyrean_lens.GraphNode, nodes []*empyrean_lens.GraphNode, nodeIDMapping map[empyrean_lens.NodeId][]empyrean_lens.NodeId) bool {
	nodeMapping := map[empyrean_lens.NodeId]*empyrean_lens.GraphNode{}
	for _, node := range nodes {
		nodeMapping[node.ID] = node
	}
	for fID, ids := range nodeIDMapping {
		for _, id := range ids {
			if id == node.ID && nodeMapping[fID].Status == empyrean_lens.ActionStatusEnum_SUCCESS {
				return true
			}
		}
	}
	return false
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
			if nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_FAIL && nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_UNREACHEAD {
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
			if nodeMapping[id].Status != empyrean_lens.ActionStatusEnum_UNREACHEAD {
				return false
			}
			newNodeIds = append(newNodeIds, nodeIDMapping[id]...)
		}
		nodeIds = newNodeIds
	}
	return true
}

func getActionStatus(nodeType empyrean_lens.LinkNodeTypeEnum, processLog aliyun.FileProcessLog) empyrean_lens.ActionStatusEnum {
	switch nodeType {
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		if strings.Contains(processLog.Message, "wcd text nil") || strings.Contains(processLog.Message, "wcd worthless") {
			return empyrean_lens.ActionStatusEnum_WORTHLESS
		}
	}
	return empyrean_lens.ActionStatusEnum_SUCCESS
}
