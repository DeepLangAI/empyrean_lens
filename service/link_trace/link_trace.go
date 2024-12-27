package link_trace

import (
	"context"
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
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func LinkTrace(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string) (interface{}, interface{}, *consts.BizCode) {
	switch entryType {
	case empyrean_lens.EntryTypeEnum_FILE:
		return FileLinkTrace(ctx, entryID, "", false, false)
	case empyrean_lens.EntryTypeEnum_WEB:
		return WebReaderLinkTrace(ctx, entryID, "", false, false)
	case empyrean_lens.EntryTypeEnum_MULTI:
		return MultiLinkTrace(ctx, entryID, false)
	case empyrean_lens.EntryTypeEnum_SUMMARY:
		return SummaryTrace(ctx, entryType, entryID, false)
	case empyrean_lens.EntryTypeEnum_OUTLINE:
		return SummaryTrace(ctx, entryType, entryID, false)
	case empyrean_lens.EntryTypeEnum_VIEWPOINT:
		return SummaryTrace(ctx, entryType, entryID, false)
	case empyrean_lens.EntryTypeEnum_MULTI_OUTLINE:
		return MultiOutlineLinkTrace(ctx, entryType, entryID, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB:
		return SubscribeSingleLinkTrace(ctx, entryType, entryID, false, false, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE:
		return SubscribeSingleLinkTrace(ctx, entryType, entryID, false, false, false)
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI:
		return SubscriMultibeLinkTrace(ctx, entryType, entryID, false, false)
	default:
		return nil, nil, &consts.RetParamError
	}
}

func FileLinkTrace(ctx context.Context, fileID, multiID string, refresh, withoutSummary bool) (*plugin.File, *empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	fileInfo, err := plugin.NewFileDao().FindFileById(ctx, fileID)
	if err != nil || fileInfo == nil {
		hlog.CtxErrorf(ctx, "get file info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	if fileInfo.RealChannelType >= int(empyrean_lens.ChannelType_IosUrl) {
		fileInfo.ChannelType = fileInfo.RealChannelType
	}
	if multiID != "" {
		fileInfo.MultiId = multiID
	}
	// 确定需要查的节点列表和节点关系
	pracessList, pracessMapping := getFileLinkTracePracessConfig(ctx, fileInfo, withoutSummary)
	// 先查数据库
	linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(empyrean_lens.EntryTypeEnum_FILE), fileID, pracessList, pracessMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	// 数据库没有，查阿里云日志
	if refresh || linkTraceGraph == nil {
		start := fileInfo.CreateTime.Add(-24 * time.Hour)
		end := fileInfo.CreateTime.Add(24 * time.Hour)
		fileInfo.TranslateEntryInfo()
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, fileInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
	}
	// 返回
	_, status := utils.GetStatusFromNode(linkTraceGraph.Nodes)
	return fileInfo, &empyrean_lens.DocLinkTraceRespData{
		LinkGraph:  linkTraceGraph,
		Cost:       float64(utils.GetCostFromNodes(linkTraceGraph.Nodes)),
		EntryID:    fileID,
		EntryType:  empyrean_lens.EntryTypeEnum_FILE,
		Title:      fileInfo.Name,
		UserID:     fileInfo.UserID,
		ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_FILE), fileInfo.MultiId, fileInfo.CopyFromResourceID, "", 0),
		Status:     status,
		TimeAt:     fileInfo.CreateTime.Format(consts.DateTimeTemplate),
	}, nil
}

func WebReaderLinkTrace(ctx context.Context, webReaderID, multiID string, refresh, withoutSummary bool) (*plugin.WebReader, *empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, webReaderID)
	if err != nil || webReaderInfo == nil {
		hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	if webReaderInfo.RealChannelType >= int(empyrean_lens.ChannelType_IosUrl) {
		webReaderInfo.ChannelType = webReaderInfo.RealChannelType
	}
	if multiID != "" {
		webReaderInfo.MultiId = multiID
	}
	// 确定需要查的节点列表和节点关系
	pracessList, pracessMapping := getWebReaderLinkTracePracessConfig(ctx, webReaderInfo, withoutSummary)
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
	_, status := utils.GetStatusFromNode(linkTraceGraph.Nodes)
	return webReaderInfo, &empyrean_lens.DocLinkTraceRespData{
		LinkGraph:  linkTraceGraph,
		Cost:       float64(utils.GetCostFromNodes(linkTraceGraph.Nodes)),
		EntryID:    webReaderID,
		EntryType:  empyrean_lens.EntryTypeEnum_WEB,
		Title:      webReaderInfo.Title,
		UserID:     webReaderInfo.UserID,
		ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_WEB), webReaderInfo.MultiId, webReaderInfo.CopyFromResourceID, "", 0),
		Status:     status,
		TimeAt:     webReaderInfo.CreateTime.Format(consts.DateTimeTemplate),
	}, nil
}

func SubscribeSingleLinkTrace(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string, isFromMulti, refresh, withoutSummary bool) (*plugin.Resource, *empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	resourceInfo, err := plugin.NewResourceDao().FindResourceById(ctx, entryID)
	if err != nil || resourceInfo == nil {
		hlog.CtxErrorf(ctx, "get resource info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 确定需要查的节点列表和节点关系
	pracessList, pracessMapping := getSubscribeLinkTracePracessConfig(int(entryType), resourceInfo.NovelFormID, isFromMulti, withoutSummary)
	// 先查数据库
	linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(entryType), entryID, pracessList, pracessMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	// 数据库没有，查阿里云日志
	if refresh || linkTraceGraph == nil {
		start := resourceInfo.CreateTime.Add(-24 * time.Hour)
		end := resourceInfo.CreateTime.Add(24 * time.Hour)
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, resourceInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
	}
	// 返回
	_, status := utils.GetStatusFromNode(linkTraceGraph.Nodes)
	return resourceInfo, &empyrean_lens.DocLinkTraceRespData{
		LinkGraph:  linkTraceGraph,
		Cost:       float64(utils.GetCostFromNodes(linkTraceGraph.Nodes)),
		EntryID:    entryID,
		EntryType:  entryType,
		UserID:     resourceInfo.UserID,
		Title:      resourceInfo.Title,
		ActionName: utils.GetActionName(int(entryType), "", "", "", 0),
		Status:     status,
		TimeAt:     resourceInfo.CreateTime.Format(consts.DateTimeTemplate),
	}, nil
}

func SummaryTrace(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string, refresh bool) (*plugin.Summary, *empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	// 获取summary详情
	summaryInfo, err := plugin.NewSummaryDao().QueryByTypeAndID(ctx, int(entryType), entryID)
	if err != nil || summaryInfo == nil {
		hlog.CtxErrorf(ctx, "get summary info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 查询文章的链路追踪，不包含模型生成节点
	articleInfo, articleLinkTraceGroup, bizCode := SummaryArticleTrace(ctx, summaryInfo, refresh)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[SummaryArticleTrace] get article graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	if articleInfo == nil && articleLinkTraceGroup == nil {
		hlog.CtxInfof(ctx, "is not retry summary, entryType: %v, entryID: %v", summaryInfo.EntryType, summaryInfo.ID)
		return nil, nil, nil
	}
	// 确定需要查的节点列表和节点关系
	pracessList, pracessMapping := getSummaryLinkTracePracessConfig(ctx, summaryInfo)
	// 先查数据库
	linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(entryType), entryID, pracessList, pracessMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	// 数据库没有，查阿里云日志
	if refresh || linkTraceGraph == nil {
		start := summaryInfo.CreateTime.Add(-24 * time.Hour)
		end := summaryInfo.CreateTime.Add(24 * time.Hour)
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, summaryInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
		if linkTraceGraph == nil || len(linkTraceGraph.Nodes) == 0 {
			return nil, nil, &consts.QueryRecordError
		}
	}
	// 将重新生成放到解析节点后面
	lenNode := len(articleLinkTraceGroup.LinkGraph.Nodes)
	articleLinkTraceGroup.LinkGraph.Edges[articleLinkTraceGroup.LinkGraph.Nodes[lenNode-1].ID] = []string{linkTraceGraph.Nodes[0].ID}
	articleLinkTraceGroup.LinkGraph.Nodes = append(articleLinkTraceGroup.LinkGraph.Nodes, linkTraceGraph.Nodes...)
	// 返回数据
	actionName := ""
	if articleLinkTraceGroup.EntryType == consts.EntryTypePDF {
		actionName = utils.GetActionName(int(entryType), articleInfo.(*plugin.File).MultiId, articleInfo.(*plugin.File).CopyFromResourceID, summaryInfo.SummaryLangType, summaryInfo.OutlineType)
		summaryInfo.SourceEntryID = articleInfo.(*plugin.File).ID.Hex()
		summaryInfo.SourceEntryType = consts.EntryTypePDF
		summaryInfo.SourceTitle = articleInfo.(*plugin.File).Name
	} else {
		actionName = utils.GetActionName(int(entryType), articleInfo.(*plugin.WebReader).MultiId, articleInfo.(*plugin.WebReader).CopyFromResourceID, summaryInfo.SummaryLangType, summaryInfo.OutlineType)
		summaryInfo.SourceEntryID = articleInfo.(*plugin.WebReader).ID.Hex()
		summaryInfo.SourceEntryType = consts.EntryTypeWEB
		summaryInfo.SourceTitle = articleInfo.(*plugin.WebReader).Title
	}
	return summaryInfo, &empyrean_lens.DocLinkTraceRespData{
		LinkGraph:  articleLinkTraceGroup.LinkGraph,
		Cost:       float64(utils.GetCostFromNodes(linkTraceGraph.Nodes)),
		EntryID:    summaryInfo.ID.Hex(),
		EntryType:  empyrean_lens.EntryTypeEnum(summaryInfo.EntryType),
		Title:      articleLinkTraceGroup.Title,
		UserID:     articleLinkTraceGroup.UserID,
		ActionName: actionName,
		Status:     linkTraceGraph.Nodes[0].Status,
		TimeAt:     linkTraceGraph.Nodes[0].EnterTime,
	}, nil
}

func MultiLinkTrace(ctx context.Context, multiID string, refresh bool) (*plugin.MultiModel, *empyrean_lens.MultiDocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	multiInfo, err := plugin.NewMultiDao().FindMultiById(ctx, multiID)
	if err != nil || multiInfo == nil {
		hlog.CtxErrorf(ctx, "get multi info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 获取子文档信息
	resources := []*empyrean_lens.ResourceInfo{}
	for _, article := range multiInfo.ArticleList {
		resources = append(resources, &empyrean_lens.ResourceInfo{
			EntryType: empyrean_lens.EntryTypeEnum(article.EntryType),
			EntryID:   article.EntryId,
		})
	}
	resourceMapping, bizCode := GetResourceInfo(ctx, resources)
	if bizCode != nil {
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
				_, articleGraph, bizCode := FileLinkTrace(ctx, articleEntry.EntryId, multiID, refresh, false)
				if bizCode != nil {
					hlog.CtxErrorf(ctx, "[FileLinkTrace] get article graph failed, err: %v", bizCode)
					return
				}
				graphMapping.Store(articleEntry.EntryId, articleGraph.LinkGraph)
			} else {
				_, articleGraph, bizCode := WebReaderLinkTrace(ctx, articleEntry.EntryId, multiID, refresh, false)
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
		pracessList, pracessMapping := getMultiLinkTracePracessConfig(ctx, multiInfo)
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
		key := fmt.Sprintf("%d_%s", article.EntryType, article.EntryId)
		if _, ok := resourceMapping[key]; !ok {
			continue
		}
		articleGraph, ok := graphMapping.Load(article.EntryId)
		if ok && articleGraph != nil {
			graph := articleGraph.(*empyrean_lens.TraceLinkGraph)
			graphs = append(graphs, graph)
			articles = append(articles, &empyrean_lens.Article{
				EntryType: empyrean_lens.EntryTypeEnum(article.EntryType),
				EntryID:   article.EntryId,
				StartID:   articleGraph.(*empyrean_lens.TraceLinkGraph).Nodes[0].ID,
				Title:     resourceMapping[key].Title,
				Graph:     graph,
			})
		}
	}
	linkTraceGraph := mergeLinkTraceGraph(graphs, multiGrap)
	// 返回
	_, status := utils.GetStatusFromNode(linkTraceGraph.Nodes)
	return multiInfo, &empyrean_lens.MultiDocLinkTraceRespData{
		Graph:      multiGrap,
		Cost:       float64(utils.GetCostFromNodes(linkTraceGraph.Nodes)),
		Articles:   articles,
		EntryID:    multiID,
		EntryType:  empyrean_lens.EntryTypeEnum_MULTI,
		Title:      multiInfo.Title,
		UserID:     multiInfo.UserID,
		ActionName: utils.GetActionName(int(empyrean_lens.EntryTypeEnum_MULTI), "", multiInfo.CopyFromResourceID, "", 0),
		Status:     status,
		TimeAt:     multiInfo.CreateTime.Format(consts.DateTimeTemplate),
	}, nil
}

func SubscriMultibeLinkTrace(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string, refresh, withoutSummary bool) (*plugin.Resource, *empyrean_lens.MultiDocLinkTraceRespData, *consts.BizCode) {
	// 获取文章详情
	resourceInfo, err := plugin.NewResourceDao().FindResourceById(ctx, entryID)
	if err != nil || resourceInfo == nil {
		hlog.CtxErrorf(ctx, "get resource info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 获取多文档对应的子文档
	novelFormInfo, err := plugin.NewResourceNovelFormDao().FindResourceNovelFormById(ctx, resourceInfo.NovelFormID)
	if err != nil || novelFormInfo == nil {
		hlog.CtxErrorf(ctx, "get novel form info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	resourceInfo.ArticleList = novelFormInfo.RelatedEntry
	// 并发链路信息
	wg, graphMapping, multiGrap := sync.WaitGroup{}, sync.Map{}, &empyrean_lens.TraceLinkGraph{}
	wg.Add(len(resourceInfo.ArticleList) + 1)
	for idx := range resourceInfo.ArticleList {
		articleEntry := resourceInfo.ArticleList[idx]
		go func() {
			defer wg.Done()
			_, articleGraph, bizCode := SubscribeSingleLinkTrace(ctx, empyrean_lens.EntryTypeEnum(articleEntry.EntryType), articleEntry.EntryId, true, refresh, true)
			if bizCode != nil {
				hlog.CtxErrorf(ctx, "[SubscribeSingleLinkTrace] get article graph failed, err: %v", bizCode)
				return
			}
			graphMapping.Store(articleEntry.EntryId, articleGraph.LinkGraph)
		}()
	}
	go func() {
		defer wg.Done()
		// 确定需要查的节点列表和节点关系
		pracessList, pracessMapping := getSubscribeLinkTracePracessConfig(int(entryType), resourceInfo.NovelFormID, true, withoutSummary)
		// 先查数据库
		linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(entryType), entryID, pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
			return
		}
		// 数据库没有，查阿里云日志
		if refresh || linkTraceGraph == nil {
			start := resourceInfo.CreateTime.Add(-24 * time.Hour)
			end := resourceInfo.CreateTime.Add(24 * time.Hour)
			linkTraceGraph, bizCode = LinkTraceGraph(ctx, resourceInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
			if bizCode != nil {
				hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
				return
			}
		}
		multiGrap = linkTraceGraph
	}()
	wg.Wait()
	// 整理数据
	graphs, articles := []*empyrean_lens.TraceLinkGraph{}, []*empyrean_lens.Article{}
	for _, article := range resourceInfo.ArticleList {
		articleGraph, ok := graphMapping.Load(article.EntryId)
		if ok && articleGraph != nil {
			graph := articleGraph.(*empyrean_lens.TraceLinkGraph)
			graphs = append(graphs, graph)
			articles = append(articles, &empyrean_lens.Article{
				EntryType: empyrean_lens.EntryTypeEnum(article.EntryType),
				EntryID:   article.EntryId,
				StartID:   articleGraph.(*empyrean_lens.TraceLinkGraph).Nodes[0].ID,
				Title:     article.Title,
				Graph:     graph,
			})
		}
	}
	linkTraceGraph := mergeLinkTraceGraph(graphs, multiGrap)
	// 返回
	_, status := utils.GetStatusFromNode(linkTraceGraph.Nodes)
	return resourceInfo, &empyrean_lens.MultiDocLinkTraceRespData{
		Graph:      multiGrap,
		Cost:       float64(utils.GetCostFromNodes(linkTraceGraph.Nodes)),
		Articles:   articles,
		EntryID:    entryID,
		EntryType:  entryType,
		UserID:     resourceInfo.UserID,
		Title:      resourceInfo.Title,
		ActionName: utils.GetActionName(int(entryType), "", "", "", 0),
		Status:     status,
		TimeAt:     resourceInfo.CreateTime.Format(consts.DateTimeTemplate),
	}, nil
}

func MultiOutlineLinkTrace(ctx context.Context, entryType empyrean_lens.EntryTypeEnum, entryID string, refresh bool) (*plugin.MultiAigc, *empyrean_lens.MultiDocLinkTraceRespData, *consts.BizCode) {
	// 获取多文档大纲记录
	multiAgicInfo, err := plugin.NewMultiAigcDao().QueryByID(ctx, entryID)
	if err != nil || multiAgicInfo == nil {
		hlog.CtxErrorf(ctx, "get multi aigc info failed, err: %v", err)
		return nil, nil, &consts.QueryRecordError
	}
	// 判断是否是重新生成
	if !IsRetryMultiOutline(ctx, multiAgicInfo) {
		hlog.CtxInfof(ctx, "is not retry aigc, entryType: %v, entryID: %v", entryType, entryID)
		return nil, nil, nil
	}
	// 确定需要查的节点列表和节点关系
	pracessList := []empyrean_lens.LinkNodeTypeEnum{empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_RETRY_FINISH}
	pracessMapping := map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{}
	// 先查数据库
	linkTraceGraph, bizCode := findLinkTraceFromMongo(ctx, int(entryType), entryID, pracessList, pracessMapping)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[findLinkTraceFromMongo] get link trace graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	// 数据库没有，查阿里云日志
	if refresh || linkTraceGraph == nil {
		start := multiAgicInfo.CreateTime.Add(-24 * time.Hour)
		end := multiAgicInfo.CreateTime.Add(24 * time.Hour)
		linkTraceGraph, bizCode = LinkTraceGraph(ctx, multiAgicInfo.TranslateEntryInfo(), start, end, pracessList, pracessMapping)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[LinkTraceGraph] get link trace graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
		if linkTraceGraph == nil || len(linkTraceGraph.Nodes) == 0 {
			return nil, nil, &consts.QueryRecordError
		}
	}
	// 查询多文档的链路追踪，不包含模型生成节点
	multiInfo, multiLinkTraceGroup, bizCode := MultiLinkTrace(ctx, multiAgicInfo.MultiID, refresh)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "[MultiLinkTrace] get multi graph failed, err: %v", bizCode)
		return nil, nil, bizCode
	}
	multiAgicInfo.UserID = multiInfo.UserID
	multiAgicInfo.ArticleList = multiInfo.ArticleList
	multiAgicInfo.ChannelType = multiInfo.ChannelType
	multiAgicInfo.CopyFromResourceID = multiInfo.CopyFromResourceID
	multiAgicInfo.Title = multiInfo.Title
	// 整理数据
	multiLinkTraceGroup.Graph.Nodes[1] = linkTraceGraph.Nodes[0]
	themeNodeID := multiLinkTraceGroup.Graph.Nodes[0].ID
	multiLinkTraceGroup.GetGraph().Edges[themeNodeID] = []string{linkTraceGraph.Nodes[0].ID}
	return multiAgicInfo, &empyrean_lens.MultiDocLinkTraceRespData{
		Graph:      multiLinkTraceGroup.Graph,
		Cost:       float64(utils.GetCostFromNodes(linkTraceGraph.Nodes)),
		Articles:   multiLinkTraceGroup.Articles,
		Title:      multiInfo.Title,
		UserID:     multiInfo.UserID,
		ActionName: utils.GetActionName(int(entryType), multiAgicInfo.MultiID, multiInfo.CopyFromResourceID, "", 0),
		Status:     linkTraceGraph.Nodes[0].Status,
		TimeAt:     linkTraceGraph.Nodes[0].EnterTime,
	}, nil
}

func SummaryArticleTrace(ctx context.Context, summaryInfo *plugin.Summary, refresh bool) (interface{}, *empyrean_lens.DocLinkTraceRespData, *consts.BizCode) {
	if summaryInfo.FileID != "" {
		fileInfo, err := plugin.NewFileDao().FindFileById(ctx, summaryInfo.FileID)
		if err != nil || fileInfo == nil {
			hlog.CtxErrorf(ctx, "get file info failed, err: %v", err)
			return nil, nil, &consts.QueryRecordError
		}
		// 判断是否是重新生成
		if !IsRetrySummary(ctx, summaryInfo, fileInfo.MultiId, fileInfo.CopyFromResourceID) {
			// 重新更新文章链路
			Save(ctx, empyrean_lens.EntryTypeEnum_FILE, fileInfo.ID.Hex())
			hlog.CtxInfof(ctx, "is not retry summary, entryType: %v, entryID: %v", summaryInfo.EntryType, summaryInfo.ID)
			return nil, nil, nil
		}
		// save file
		Save(ctx, empyrean_lens.EntryTypeEnum_FILE, fileInfo.ID.Hex())
		info, articleGraph, bizCode := FileLinkTrace(ctx, fileInfo.ID.Hex(), "", false, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[FileLinkTrace] get article graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
		return info, articleGraph, nil
	} else {
		webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderByUserIDAndUrl(ctx, summaryInfo.UserID, summaryInfo.Url)
		if err != nil || webReaderInfo == nil {
			hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
			return nil, nil, &consts.QueryRecordError
		}
		// 判断是否是重新生成
		if !IsRetrySummary(ctx, summaryInfo, webReaderInfo.MultiId, webReaderInfo.CopyFromResourceID) {
			// 重新更新文章链路
			Save(ctx, empyrean_lens.EntryTypeEnum_WEB, webReaderInfo.ID.Hex())
			hlog.CtxInfof(ctx, "is not retry summary, entryType: %v, entryID: %v", summaryInfo.EntryType, summaryInfo.ID)
			return nil, nil, nil
		}
		// save web reader
		Save(ctx, empyrean_lens.EntryTypeEnum_WEB, webReaderInfo.ID.Hex())
		info, articleGraph, bizCode := WebReaderLinkTrace(ctx, webReaderInfo.ID.Hex(), "", false, true)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[WebReaderLinkTrace] get article graph failed, err: %v", bizCode)
			return nil, nil, bizCode
		}
		return info, articleGraph, nil
	}
}

func IsRetrySummary(ctx context.Context, summaryInfo *plugin.Summary, multiID, copyFromResource string) bool {
	// 判断是否是重试summary
	// 根据生成时间判断
	firstSummary, err := plugin.NewSummaryDao().QueryFirstSummary(ctx, summaryInfo.EntryType, summaryInfo.UserID, summaryInfo.Url)
	if err != nil || firstSummary == nil {
		hlog.CtxErrorf(ctx, "get first summary failed, err: %v", err)
		return false
	}
	// 英文大纲，判断是否是重试
	if summaryInfo.EntryType == int(empyrean_lens.EntryTypeEnum_OUTLINE) {
		// 语言不同，肯定是重试
		if summaryInfo.SummaryLangType != firstSummary.SummaryLangType {
			return true
		}
	}
	if firstSummary.ID != summaryInfo.ID && firstSummary.PairID != summaryInfo.PairID {
		return true
	}
	return false
}

func IsRetryMultiOutline(ctx context.Context, multiAigcInfo *plugin.MultiAigc) bool {
	// 根据生成时间判断
	theme := ""
	for key := range multiAigcInfo.AigcResult {
		if multiAigcInfo.AigcResult[key] != nil {
			theme = key
			break
		}
	}
	firstAigc, err := plugin.NewMultiAigcDao().QueryFirstAigc(ctx, multiAigcInfo.MultiID, theme)
	if err != nil || firstAigc == nil {
		hlog.CtxErrorf(ctx, "get first summary failed, err: %v", err)
		return false
	}
	if multiAigcInfo.ID != firstAigc.ID {
		return true
	}
	return false
}

func getWebReaderLinkTracePracessConfig(ctx context.Context, webReaderInfo *plugin.WebReader, withoutSummary bool) ([]empyrean_lens.LinkNodeTypeEnum, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) {
	pracessList := consts.SingleWebReaderProcessList
	pracessMapping := consts.SingleWebReaderProcessMapping
	if webReaderInfo.CopyFromResourceID == "" && webReaderInfo.MultiId != "" {
		pracessList = consts.MultiWebReaderProcessList
		pracessMapping = consts.MultiWebReaderProcessMapping
	}
	if webReaderInfo.CopyFromResourceID != "" {
		pracessList = consts.SubscribeWebReaderProcessList
		pracessMapping = consts.SubscribeWebReaderProcessMapping
	}
	noNeedNodeType := []empyrean_lens.LinkNodeTypeEnum{}
	// 订阅来源，没有crawler节点
	if webReaderInfo.CopyFromResourceID != "" {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH)
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH)
		// 有没有新内容形态
		resourceInfo, _ := plugin.NewResourceDao().FindResourceById(ctx, webReaderInfo.CopyFromResourceID)
		if resourceInfo != nil && resourceInfo.NovelFormID == "" {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH)
		}
	}
	// copy来源，不需要上传节点
	if webReaderInfo.CopyFromUrlID != "" {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH)
	}
	// 插件来源，不需要crawler节点
	if utils.ChannelIntToString(webReaderInfo.ChannelType) == "语鲸插件" {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH)
	}
	// 是否需要模型生成
	if withoutSummary {
		noNeedNodeType = append(noNeedNodeType, []empyrean_lens.LinkNodeTypeEnum{
			empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
			empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
			empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH,
		}...)
		return filterUnNeedNodeType(noNeedNodeType, pracessList, pracessMapping)
	}
	// 需要模型生成，根据模型生成时间判断
	pracessList, pracessMapping = filterUnNeedNodeType(noNeedNodeType, pracessList, pracessMapping)
	return getLinkTraceSummaryPracessConfig(ctx, webReaderInfo.ChannelType, webReaderInfo.UserID, webReaderInfo.URL, pracessList, pracessMapping)
}

func getFileLinkTracePracessConfig(ctx context.Context, fileInfo *plugin.File, withoutSummary bool) ([]empyrean_lens.LinkNodeTypeEnum, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) {
	pracessList := consts.SingleFileProcessList
	pracessMapping := consts.SingleFileProcessMapping
	// 多文档不需要模型生成
	if fileInfo.CopyFromResourceID == "" && fileInfo.MultiId != "" {
		pracessList = consts.MultiFileProcessList
		pracessMapping = consts.MultiFileProcessMapping
	}
	if fileInfo.CopyFromResourceID != "" {
		pracessList = consts.SubscribeFileProcessList
		pracessMapping = consts.SubscribeFileProcessMapping
	}
	noNeedNodeType := []empyrean_lens.LinkNodeTypeEnum{}
	// copy来源，不需要上传节点
	if fileInfo.CopyFromResourceID != "" {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH)
		// 有没有新内容形态
		resourceInfo, _ := plugin.NewResourceDao().FindResourceById(ctx, fileInfo.CopyFromResourceID)
		if resourceInfo != nil && resourceInfo.NovelFormID == "" {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH)
		}
	}
	// copy来源不需要，不需要苏秦节点
	if fileInfo.CopyFromFildID != "" {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH)
	}
	// 是否需要模型生成
	if withoutSummary {
		noNeedNodeType = append(noNeedNodeType, []empyrean_lens.LinkNodeTypeEnum{
			empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
			empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
			empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH,
		}...)
		return filterUnNeedNodeType(noNeedNodeType, pracessList, pracessMapping)
	}
	// 需要模型生成，根据模型生成时间判断
	pracessList, pracessMapping = filterUnNeedNodeType(noNeedNodeType, pracessList, pracessMapping)
	return getLinkTraceSummaryPracessConfig(ctx, fileInfo.ChannelType, fileInfo.UserID, fileInfo.FileURL, pracessList, pracessMapping)
}

func getLinkTraceSummaryPracessConfig(ctx context.Context, channelType int, userID, url string,
	pracessList []empyrean_lens.LinkNodeTypeEnum, pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) ([]empyrean_lens.LinkNodeTypeEnum, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) {
	noNeedNodeType := []empyrean_lens.LinkNodeTypeEnum{}
	// 模型生成时间判断
	firstOutlineLanguage := ""
	summaryNodeTypeMapping := map[empyrean_lens.EntryTypeEnum]struct{}{}
	summaries, err := plugin.NewSummaryDao().FindByUserIDAndUrl(ctx, userID, url)
	if err != nil || len(summaries) == 0 {
		hlog.CtxErrorf(ctx, "get summary failed, err: %v", err)
	} else {
		for _, summary := range summaries {
			summaryType := empyrean_lens.EntryTypeEnum(summary.EntryType)
			summaryNodeTypeMapping[summaryType] = struct{}{}
			if firstOutlineLanguage == "" && summaryType == empyrean_lens.EntryTypeEnum_OUTLINE {
				firstOutlineLanguage = summary.SummaryLangType
				if firstOutlineLanguage == "" {
					firstOutlineLanguage = "zh"
				}
			}
		}
	}
	switch utils.ChannelIntToString(channelType) {
	case "语鲸插件":
		// 插件来源，没有生成，不需要显示生成节点
		if len(summaryNodeTypeMapping) == 0 {
			noNeedNodeType = append(noNeedNodeType, []empyrean_lens.LinkNodeTypeEnum{
				empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
				empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
				empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
				empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
				empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
			}...)
		}
		// 没有概述，不需要概述节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_SUMMARY]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH)
		}
		// 没有关键观点，不需要关键观点节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_VIEWPOINT]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH)
		}
	case "语鲸web":
		// 语鲸web端，没有生成，不需要显示生成节点
		if len(summaryNodeTypeMapping) == 0 {
			noNeedNodeType = append(noNeedNodeType, []empyrean_lens.LinkNodeTypeEnum{
				empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
				empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
				empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
				empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
				empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
				empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH,
			}...)
		}
		// 没有概述，不需要概述节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_SUMMARY]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH)
		}
		// 没有关键观点，不需要关键观点节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_VIEWPOINT]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH)
		}
	case "语鲸小助手", "语鲸小程序":
		// 小助手，小程序没有概述，不需要概述节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_SUMMARY]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH)
		}
		// 小助手，小程序没有生成关键信息，不需要关键信息节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_VIEWPOINT]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH)
		}
	case "语鲸app", "语鲸h5":
		// app，h5没有生成概述，不需要概述节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_SUMMARY]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH)
		}
		// app，h5没有生成关键信息，不需要关键信息节点
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_VIEWPOINT]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH)
		}
	}
	// 生成英文大纲，不需要默认、详细大纲节点，否则不需要默认大纲节点
	if firstOutlineLanguage == "zh" || firstOutlineLanguage == "" {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH)
		if _, ok := summaryNodeTypeMapping[empyrean_lens.EntryTypeEnum_OUTLINE]; !ok {
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH)
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH)
		}
	} else {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH)
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH)
	}
	// 过滤不需要的节点
	return filterUnNeedNodeType(noNeedNodeType, pracessList, pracessMapping)
}

func getSubscribeLinkTracePracessConfig(entryType int, novelFormID string, isFromMulti, withoutSummary bool) ([]empyrean_lens.LinkNodeTypeEnum, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) {
	pracessList := []empyrean_lens.LinkNodeTypeEnum{}
	pracessMapping := map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{}
	noNeedNodeType := []empyrean_lens.LinkNodeTypeEnum{}
	switch entryType {
	case int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB):
		if !isFromMulti {
			pracessList, pracessMapping = consts.SubscribeWebReaderProcessList, consts.SubscribeWebReaderProcessMapping
		} else {
			pracessList, pracessMapping = consts.MultiWebReaderProcessList, consts.MultiWebReaderProcessMapping
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH)
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH)
		}
	case int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE):
		if !isFromMulti {
			pracessList, pracessMapping = consts.SubscribeFileProcessList, consts.SubscribeFileProcessMapping
		} else {
			pracessList, pracessMapping = consts.MultiFileProcessList, consts.MultiFileProcessMapping
			noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH)
		}
	case int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI):
		pracessList, pracessMapping = consts.SubscribeMultiProcessList, consts.SubscribeMultiProcessMapping
	}
	// 过滤不需要的节点
	if withoutSummary {
		noNeedNodeType = append(noNeedNodeType, []empyrean_lens.LinkNodeTypeEnum{
			empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
			empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
			empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH,
			empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH,
		}...)
	}
	// 是否有新内容形态
	if novelFormID == "" {
		noNeedNodeType = append(noNeedNodeType, empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH)
	}
	return filterUnNeedNodeType(noNeedNodeType, pracessList, pracessMapping)
}

func getSummaryLinkTracePracessConfig(ctx context.Context, summaryInfo *plugin.Summary) ([]empyrean_lens.LinkNodeTypeEnum, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) {
	pracessList := []empyrean_lens.LinkNodeTypeEnum{}
	pracessMapping := map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{}
	switch summaryInfo.EntryType {
	case int(empyrean_lens.EntryTypeEnum_SUMMARY):
		pracessList = append(pracessList, empyrean_lens.LinkNodeTypeEnum_SUMMARY_RETRY_FINISH)
	case int(empyrean_lens.EntryTypeEnum_OUTLINE):
		if summaryInfo.OutlineType == 2 {
			pracessList = append(pracessList, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_RETRY_FINISH)
		} else if summaryInfo.OutlineType == 1 {
			pracessList = append(pracessList, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_RETRY_FINISH)
		} else {
			pracessList = append(pracessList, empyrean_lens.LinkNodeTypeEnum_OUTLINE_RETRY_FINISH)
		}
	case int(empyrean_lens.EntryTypeEnum_VIEWPOINT):
		pracessList = append(pracessList, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH)
	}
	return pracessList, pracessMapping
}

func getMultiLinkTracePracessConfig(ctx context.Context, multiInfo *plugin.MultiModel) ([]empyrean_lens.LinkNodeTypeEnum, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) {
	pracessList, pracessMapping := consts.MultiProcessList, consts.MultiProcessMapping
	if multiInfo.CopyFromMultiID != "" || multiInfo.CopyFromResourceID != "" {
		return []empyrean_lens.LinkNodeTypeEnum{
			empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH,
		}, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{}
	}
	return pracessList, pracessMapping
}

func filterUnNeedNodeType(noNeedNodeType []empyrean_lens.LinkNodeTypeEnum, pracessList []empyrean_lens.LinkNodeTypeEnum, pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) ([]empyrean_lens.LinkNodeTypeEnum, map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) {
	newPracessList := []empyrean_lens.LinkNodeTypeEnum{}
	for _, pracessType := range pracessList {
		if !utils.Contains(noNeedNodeType, pracessType) {
			newPracessList = append(newPracessList, pracessType)
		}
	}
	newPracessMapping := map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{}
	for _, pracessType := range newPracessList {
		if _, ok := pracessMapping[pracessType]; !ok {
			continue
		}
		children := []empyrean_lens.LinkNodeTypeEnum{pracessType}
		for len(children) > 0 {
			nextChildren := []empyrean_lens.LinkNodeTypeEnum{}
			for _, child := range children {
				if _, ok := pracessMapping[child]; ok {
					nextChildren = append(nextChildren, pracessMapping[child]...)
				}
			}
			needNextChildren := []empyrean_lens.LinkNodeTypeEnum{}
			for _, nextChild := range nextChildren {
				if !utils.Contains(noNeedNodeType, nextChild) {
					needNextChildren = append(needNextChildren, nextChild)
				}
			}
			if len(needNextChildren) > 0 {
				newPracessMapping[pracessType] = needNextChildren
				break
			}
			children = nextChildren
		}
	}
	return newPracessList, newPracessMapping
}

func findLinkTraceFromMongo(ctx context.Context, entryType int, entryID string,
	pracessList []empyrean_lens.LinkNodeTypeEnum, pracessMapping map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum) (*empyrean_lens.TraceLinkGraph, *consts.BizCode) {
	if len(pracessList) == 0 {
		return &empyrean_lens.TraceLinkGraph{
			Nodes: []*empyrean_lens.GraphNode{},
			Edges: map[string][]string{},
		}, nil
	}
	// 先查数据库
	entryInfo, err := bi.NewEntryInfoDao().FindByEntryIDAndEntryType(ctx, entryID, entryType)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryIDAndEntryType] get entry actions failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	if entryInfo == nil {
		return nil, nil
	}
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
		if utils.Contains(pracessList, empyrean_lens.LinkNodeTypeEnum(action.ActionType)) {
			nodeMappingNew[actionType] = action.TranslateGraphNode()
			nodes = append(nodes, nodeMappingNew[actionType])
		} else {
			if action.ActionType == int(empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH) {
				if _, ok := nodeMappingNew[empyrean_lens.LinkNodeTypeEnum(empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH)]; !ok &&
					utils.Contains(pracessList, empyrean_lens.LinkNodeTypeEnum(empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH)) {
					node := action.TranslateGraphNode()
					node.Type = empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH
					node.Name = consts.LinkNodeTypeName[empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH]
					nodeMappingNew[empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH] = node
					nodes = append(nodes, node)
				}
			}
		}
	}
	edges := map[empyrean_lens.NodeId][]empyrean_lens.NodeId{}
	for pracessType, pracessTypeList := range pracessMapping {
		node1, ok := nodeMappingNew[pracessType]
		if !ok || node1 == nil {
			node1 = makeEmptyNode(pracessType, entryInfo)
		}
		edges[node1.ID] = []empyrean_lens.NodeId{}
		for _, itemType := range pracessTypeList {
			node2, ok := nodeMappingNew[itemType]
			if !ok || node2 == nil {
				node2 = makeEmptyNode(pracessType, entryInfo)
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
	if len(pracessList) == 0 {
		return &empyrean_lens.TraceLinkGraph{
			Nodes: []*empyrean_lens.GraphNode{},
			Edges: map[string][]string{},
		}, nil
	}
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
			if node.Status == empyrean_lens.ActionStatusEnum_WORTHLESS || node.Status == empyrean_lens.ActionStatusEnum_NO_LOG {
				hasFailedNode = true
				continue
			}
			if node.Status != empyrean_lens.ActionStatusEnum_SUCCESS && node.Status != empyrean_lens.ActionStatusEnum_FAIL &&
				node.Status != empyrean_lens.ActionStatusEnum_NO_LOG && isChildSuccess(node, nodes, edges) {
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

func GetProcessNode(ctx context.Context, processType empyrean_lens.LinkNodeTypeEnum, entryInfo *bi.EntryInfo, start, end time.Time) (*empyrean_lens.GraphNode, *consts.BizCode) {
	if entryInfo.ParentEntryID != "" && utils.IsCopyNodeType(entryInfo.ParentEntryType, int(processType)) {
		// 先从数据库拿日志
		entryActions, err := bi.NewEntryActionDao().FindByEntryTypeEntryIDNodeType(ctx, entryInfo.ParentEntryType, int(processType), entryInfo.ParentEntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindByEntryTypeEntryIDNodeType] get entry actions failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(entryActions) != 0 && len(entryActions[0].ActionIOs) != 0 {
			name := entryActions[0].TranslateGraphNode().Name
			return &empyrean_lens.GraphNode{
				ID:         primitive.NewObjectID().Hex(),
				Name:       name,
				Type:       processType,
				Status:     empyrean_lens.ActionStatusEnum(entryActions[0].ActionStatus),
				EnterTime:  entryActions[0].ActionStartTime.Format(consts.DateTimeTemplate),
				FinishTime: entryActions[0].ActionEndTime.Format(consts.DateTimeTemplate),
				TraceID:    entryActions[0].ActionIOs[0].TraceID,
			}, nil
		}
		// 获取文章信息
		newEntryInfo, bizCode := GetEntryInfo(ctx, empyrean_lens.EntryTypeEnum(entryInfo.ParentEntryType), entryInfo.ParentEntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "get entry info failed, err: %v", bizCode)
			return nil, &consts.QueryRecordError
		}
		start := newEntryInfo.EntryCreateTime.Add(-24 * time.Hour)
		end := newEntryInfo.EntryCreateTime.Add(24 * time.Hour)
		return doGetProcessNode(ctx, processType, newEntryInfo, start, end)
	}
	return doGetProcessNode(ctx, processType, entryInfo, start, end)
}

// 获取节点日志
func doGetProcessNode(ctx context.Context, processType empyrean_lens.LinkNodeTypeEnum, entryInfo *bi.EntryInfo, start, end time.Time) (*empyrean_lens.GraphNode, *consts.BizCode) {
	switch processType {
	case empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:
		processLogs, err := aliyun.ResourceUploadQuery(ctx, entryInfo.EntryID, consts.EntryTypeMap[entryInfo.EntryType], start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[ResourceUploadQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		processLogs, err := aliyun.CrawlerQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[CrawlerQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		// 没有抓取日志，使用输入输出兜底
		if len(processLogs) == 0 {
			apiLogsInput, err := aliyun.CrawlerOutRequestQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[CrawlerOutRequestQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.CrawlerOutResponseQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[CrawlerOutResponseQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			return processLogsToNode(processType, append(apiLogsInput, apiLogsOuput...), entryInfo, nil), nil
		}
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
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
					hlog.CtxErrorf(ctx, "[SingleTraceIDErrorQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				return processLogsToNode(processType, append(apiLogsInput, errorLogs...), entryInfo, nil), nil
			}
		}
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		// 查询copy日志
		processLogs, err := aliyun.PDFParserCopyQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[PDFParserCopyQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(processLogs) > 0 {
			return makeEmptyNode(processType, entryInfo), nil
		}
		processLogs, err = aliyun.PDFParserQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[PDFParserQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		// 兜底查fc日志
		if len(processLogs) == 0 {
			processLogs, err = aliyun.PDFParserFcErrorQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[PDFParserFcErrorQuery] get process logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
		}
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
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
			return processLogsToNode(processType, processLogs, entryInfo, nil), nil
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
		node := processLogsToNode(processType, processLogs, entryInfo, nil)
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
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
		// 查数据库，没有记录，说明未执行/长度不够
		summaryID, pairID, createAt, bizCode := FindSummaryID(ctx, entryInfo, processType)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[FindSummaryID] find summary fail, err: %v", bizCode)
			// return nil, &consts.QueryRecordError
		}
		if summaryID == "" {
			node := makeEmptyNode(processType, entryInfo)
			// 兜底，未触发
			node.Status = empyrean_lens.ActionStatusEnum_UNREACHEAD
			return node, nil
		}
		// 输入输出判断
		query := pairID
		if query == "" {
			query = summaryID
		}
		start, end := createAt.Add(-24*time.Hour), createAt.Add(24*time.Hour)
		traceIDLogs, err := aliyun.TraceIDQuery(ctx, summaryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		processLogs, extra := []aliyun.FileProcessLog{}, map[string]string{}
		querys, queryMapping := []string{}, map[string]struct{}{}
		for _, log := range traceIDLogs {
			if _, ok := queryMapping[log.TraceId]; !ok {
				queryMapping[log.TraceId] = struct{}{}
				querys = append(querys, log.TraceId)
			}
		}
		querys = append(querys, entryInfo.EntryID)
		for _, query := range querys {
			apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, query, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ViewPointModelOutRequestQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, query, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ViewPointModelOutResponseQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(apiLogsInput) > 0 && len(apiLogsOuput) > 0 {
				processLogs = append(apiLogsInput, apiLogsOuput...)
				extra["summary_id"] = summaryID
				break
			}
		}
		return processLogsToNode(processType, processLogs, entryInfo, extra), nil
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
		// 查数据库，没有记录，说明未执行/长度不够
		summaryID, pairID, createAt, bizCode := FindSummaryID(ctx, entryInfo, processType)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[FindSummaryID] find summary fail, err: %v", bizCode)
			// return nil, &consts.QueryRecordError
		}
		if summaryID == "" {
			node := makeEmptyNode(processType, entryInfo)
			// 兜底，长度不够
			if entryInfo.ContentSize < 1000 {
				node.Status = empyrean_lens.ActionStatusEnum_LENGTH_ERROR
			} else {
				node.Status = empyrean_lens.ActionStatusEnum_UNREACHEAD
			}
			return node, nil
		}
		// 输入输出判断
		query := pairID
		if query == "" {
			query = summaryID
		}
		start, end := createAt.Add(-24*time.Hour), createAt.Add(24*time.Hour)
		traceIDLogs, err := aliyun.TraceIDQuery(ctx, summaryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		processLogs, extra := []aliyun.FileProcessLog{}, map[string]string{}
		querys, queryMapping := []string{}, map[string]struct{}{}
		for _, log := range traceIDLogs {
			if _, ok := queryMapping[log.TraceId]; !ok {
				queryMapping[log.TraceId] = struct{}{}
				querys = append(querys, log.TraceId)
			}
		}
		querys = append(querys, entryInfo.EntryID)
		for _, query := range querys {
			apiLogsInput, err := aliyun.AbstractModelOutRequestQuery(ctx, query, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[AbstractModelOutRequestQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.AbstractModelOutResponseQuery(ctx, query, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[AbstractModelOutResponseQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(apiLogsInput) > 0 && len(apiLogsOuput) > 0 {
				processLogs = append(apiLogsInput, apiLogsOuput...)
				break
			}
		}
		if len(processLogs) == 0 && utils.IsSubscribe(entryInfo.EntryType) {
			node := makeEmptyNode(processType, entryInfo)
			// 兜底，长度不够
			if entryInfo.ContentSize < 1000 {
				node.Status = empyrean_lens.ActionStatusEnum_LENGTH_ERROR
			} else {
				node.Status = empyrean_lens.ActionStatusEnum_UNREACHEAD
			}
			return node, nil
		}
		return processLogsToNode(processType, processLogs, entryInfo, extra), nil
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
		empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
		empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH:
		// 查数据库，没有记录，说明未执行/长度不够
		summaryID, pairID, createAt, bizCode := FindSummaryID(ctx, entryInfo, processType)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "[FindSummaryID] find summary fail, err: %v", bizCode)
			// return nil, &consts.QueryRecordError
		}
		if summaryID == "" {
			node := makeEmptyNode(processType, entryInfo)
			// 兜底，长度不够
			if entryInfo.ContentSize < 1000 {
				node.Status = empyrean_lens.ActionStatusEnum_LENGTH_ERROR
			} else {
				node.Status = empyrean_lens.ActionStatusEnum_UNREACHEAD
			}
			return node, nil
		}
		// 输入输出判断
		query := pairID
		if query == "" {
			query = summaryID
		}
		if utils.IsSubscribe(entryInfo.EntryType) {
			query = entryInfo.EntryID + " " + "outline_model"
		}
		start, end := createAt.Add(-24*time.Hour), createAt.Add(24*time.Hour)
		traceIDLogs, err := aliyun.TraceIDQuery(ctx, query, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQuery] get api logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		processLogs, extra := []aliyun.FileProcessLog{}, map[string]string{}
		querys, queryMapping := []string{}, map[string]struct{}{}
		for _, log := range traceIDLogs {
			if _, ok := queryMapping[log.TraceId]; !ok {
				queryMapping[log.TraceId] = struct{}{}
				querys = append(querys, log.TraceId)
			}
		}
		querys = append(querys, entryInfo.EntryID)
		for _, query := range querys {
			apiLogsInput, err := aliyun.OutlineModelOutRequestQueryByTraceID(ctx, query, "", start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[OutlineModelOutRequestQueryByTraceID] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			newApiLogsInput := []aliyun.FileProcessLog{}
			for _, log := range apiLogsInput {
				// 解压缩
				inputStr := GetReqRespFromMsg(log.Message, "req:")
				if inputStr == "" {
					inputStr = log.Message
				}
				if processType == empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH {
					if strings.Contains(inputStr, "\"verbose\":true") {
						newApiLogsInput = append(newApiLogsInput, log)
					}
				} else {
					if strings.Contains(inputStr, "\"verbose\":false") {
						newApiLogsInput = append(newApiLogsInput, log)
					}
				}
			}
			apiLogsOuput, err := aliyun.OutlineModelOutResponseQueryByTraceID(ctx, query, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[OutlineModelOutResponseQueryByTraceID] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			newApiLogsOuput := []aliyun.FileProcessLog{}
			if len(newApiLogsInput) > 0 {
				for _, log := range apiLogsOuput {
					if log.OperationID == newApiLogsInput[0].OperationID {
						newApiLogsOuput = append(newApiLogsOuput, log)
					}
				}
			}
			if len(newApiLogsInput) > 0 && len(newApiLogsOuput) > 0 {
				processLogs = append(newApiLogsInput, newApiLogsOuput...)
				extra["summary_id"] = summaryID
				break
			}
		}
		return processLogsToNode(processType, processLogs, entryInfo, extra), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH:
		processLogs, err := aliyun.MultiItemAnalysisQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiAnalysisQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(processLogs) == 0 {
			apiLogsInput, err := aliyun.MultiSingleAnalysisModelOutRequestQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[MultiSingleAnalysisModelOutRequestQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.MultiSingleAnalysisModelOutResponseQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[MultiSingleAnalysisModelOutResponseQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			return processLogsToNode(processType, append(apiLogsInput, apiLogsOuput...), entryInfo, nil), nil
		}
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:
		processLogs, err := aliyun.MultiThemeQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiThemeQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		// 订阅来源，需要查询traceID
		if len(processLogs) == 0 || utils.IsSubscribe(int(entryInfo.EntryType)) {
			traceLogs, err := aliyun.ResourceTraceIDQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ResourceTraceIDQuery] get trace logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(traceLogs) != 0 {
				traceID := traceLogs[0].TraceId
				apiLogsInput, err := aliyun.MultiThemeModelOutRequestQuery(ctx, traceID, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[MultiThemeModelOutRequestQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				apiLogsOuput, err := aliyun.MultiThemeModelOutResponseQuery(ctx, traceID, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[MultiThemeModelOutResponseQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				processLogs = append(apiLogsInput, apiLogsOuput...)
			}
		}
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:
		// 订阅来源，需要查询traceID
		processLogs, err := aliyun.MultiOutlineQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		// 订阅来源，需要查询traceID
		if len(processLogs) == 0 && utils.IsSubscribe(int(entryInfo.EntryType)) {
			traceLogs, err := aliyun.ResourceTraceIDQuery(ctx, entryInfo.EntryID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ResourceTraceIDQuery] get trace logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			if len(traceLogs) != 0 {
				traceID := traceLogs[0].TraceId
				apiLogsInput, err := aliyun.MultiOutlineModelOutRequestQuery(ctx, traceID, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[MultiOutlineModelOutRequestQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				apiLogsOuput, err := aliyun.MultiOutlineModelOutResponseQuery(ctx, traceID, start, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "[MultiOutlineModelOutResponseQuery] get api logs failed, err: %v", err)
					return nil, &consts.QueryRecordError
				}
				processLogs = append(apiLogsInput, apiLogsOuput...)
			}
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
		return processLogsToNode(processType, processLogs, entryInfo, nil), nil
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_RETRY_FINISH:
		node := makeEmptyNode(processType, entryInfo)
		// 获取summary记录
		summaryInfo, err := plugin.NewSummaryDao().QueryByTypeAndID(ctx, int(empyrean_lens.EntryTypeEnum_SUMMARY), entryInfo.EntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "[QueryByTypeAndID] get summary info failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if summaryInfo.Content == "" {
			node.Status = empyrean_lens.ActionStatusEnum_FAIL
		} else {
			node.Status = empyrean_lens.ActionStatusEnum_SUCCESS
		}
		// 获取traceID
		logs1, err := aliyun.TraceIDQuery(ctx, summaryInfo.PairID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQuery] get traceID failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		logs2, err := aliyun.TraceIDQueryByUserID(ctx, summaryInfo.UserID, "abstract", start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQueryByUserID] get traceID failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		queryMapping := map[string]struct{}{}
		for _, log := range append(logs1, logs2...) {
			traceID := log.TraceId
			if _, ok := queryMapping[traceID]; ok {
				continue
			}
			queryMapping[traceID] = struct{}{}
			// 查输入输出
			apiLogsInput, err := aliyun.AbstractModelOutRequestQueryByTraceID(ctx, traceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[AbstractModelOutRequestQueryByTraceID] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.AbstractModelOutResponseQueryByTraceID(ctx, traceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[AbstractModelOutResponseQueryByTraceID] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			// apiLogsInput 倒序排序
			sort.Slice(apiLogsInput, func(i, j int) bool {
				return apiLogsInput[i].Asctime.After(apiLogsInput[j].Asctime)
			})
			newApiLogsInput := []aliyun.FileProcessLog{}
			for _, log := range apiLogsInput {
				// 解压缩
				inputStr := GetReqRespFromMsg(log.Message, "req:")
				if inputStr == "" {
					inputStr = log.Message
				}
				// 语言判断
				if summaryInfo.SummaryLangType == "en" && !strings.Contains(inputStr, "\"en_mode\":true") {
					continue
				}
				if summaryInfo.SummaryLangType == "zh" && !strings.Contains(inputStr, "\"en_mode\":false") {
					continue
				}
				// 时间判断
				if summaryInfo.CreateTime.Add(5*time.Second).Format(consts.DateTimeTemplate) > log.Asctime.Format(consts.DateTimeTemplate) {
					newApiLogsInput = append(newApiLogsInput, log)
				}
			}
			if len(newApiLogsInput) > 0 && len(apiLogsOuput) > 0 {
				node.EnterTime = newApiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
				node.FinishTime = newApiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
				node.TraceID = traceID
				return node, nil
			}
			if len(newApiLogsInput) > 0 {
				node.TraceID = traceID
			}
		}
		node.EnterTime = summaryInfo.CreateTime.Format(consts.DateTimeTemplate)
		node.FinishTime = summaryInfo.CreateTime.Format(consts.DateTimeTemplate)
		return node, nil
	case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH:
		node := makeEmptyNode(processType, entryInfo)
		// 获取summary记录
		summaryInfo, err := plugin.NewSummaryDao().QueryByTypeAndID(ctx, int(empyrean_lens.EntryTypeEnum_VIEWPOINT), entryInfo.EntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "[QueryByTypeAndID] get summary info failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if summaryInfo.Content == "" {
			node.Status = empyrean_lens.ActionStatusEnum_FAIL
		} else {
			node.Status = empyrean_lens.ActionStatusEnum_SUCCESS
		}
		// 获取traceID
		logs1, err := aliyun.TraceIDQuery(ctx, summaryInfo.PairID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQuery] get traceID failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		logs2, err := aliyun.TraceIDQueryByUserID(ctx, summaryInfo.UserID, "viewpoint", start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQueryByUserID] get traceID failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		queryMapping := map[string]struct{}{}
		for _, log := range append(logs1, logs2...) {
			traceID := log.TraceId
			if _, ok := queryMapping[traceID]; ok {
				continue
			}
			queryMapping[traceID] = struct{}{}
			apiLogsInput, err := aliyun.ViewPointModelOutRequestQuery(ctx, traceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ViewPointModelOutRequestQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.ViewPointModelOutResponseQuery(ctx, traceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[ViewPointModelOutResponseQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			// apiLogsInput 倒序排序
			sort.Slice(apiLogsInput, func(i, j int) bool {
				return apiLogsInput[i].Asctime.After(apiLogsInput[j].Asctime)
			})
			newApiLogsInput := []aliyun.FileProcessLog{}
			for _, log := range apiLogsInput {
				// 解压缩
				inputStr := GetReqRespFromMsg(log.Message, "req:")
				if inputStr == "" {
					inputStr = log.Message
				}
				// 语言判断
				if summaryInfo.SummaryLangType == "en" && !strings.Contains(inputStr, "\"en_mode\":true") {
					continue
				}
				if summaryInfo.SummaryLangType == "zh" && !strings.Contains(inputStr, "\"en_mode\":false") {
					continue
				}
				// 时间判断
				if summaryInfo.CreateTime.Add(5*time.Second).Format(consts.DateTimeTemplate) > log.Asctime.Format(consts.DateTimeTemplate) {
					newApiLogsInput = append(newApiLogsInput, log)
				}
			}
			if len(newApiLogsInput) > 0 && len(apiLogsOuput) > 0 {
				node.EnterTime = newApiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
				node.FinishTime = newApiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
				node.TraceID = traceID
				return node, nil
			}
			if len(newApiLogsInput) > 0 {
				node.TraceID = traceID
			}
		}
		node.EnterTime = summaryInfo.CreateTime.Format(consts.DateTimeTemplate)
		node.FinishTime = summaryInfo.CreateTime.Format(consts.DateTimeTemplate)
		return node, nil
	case empyrean_lens.LinkNodeTypeEnum_OUTLINE_RETRY_FINISH,
		empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_RETRY_FINISH,
		empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_RETRY_FINISH:
		node := makeEmptyNode(processType, entryInfo)
		// 获取summary记录
		summaryInfo, err := plugin.NewSummaryDao().QueryByTypeAndID(ctx, int(empyrean_lens.EntryTypeEnum_OUTLINE), entryInfo.EntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "[QueryByTypeAndID] get summary info failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if summaryInfo.Content == "" {
			node.Status = empyrean_lens.ActionStatusEnum_FAIL
		} else {
			node.Status = empyrean_lens.ActionStatusEnum_SUCCESS
		}
		// 获取traceID
		logs1, err := aliyun.TraceIDQuery(ctx, summaryInfo.PairID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQuery] get traceID failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		logs2, err := aliyun.TraceIDQueryByUserID(ctx, summaryInfo.UserID, "outline", start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[TraceIDQueryByUserID] get traceID failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		queryMapping := map[string]struct{}{}
		for _, log := range append(logs1, logs2...) {
			traceID := log.TraceId
			if _, ok := queryMapping[traceID]; ok {
				continue
			}
			queryMapping[traceID] = struct{}{}
			apiLogsInput, err := aliyun.OutlineModelOutRequestQueryByTraceID(ctx, traceID, entryInfo.UserID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[OutlineModelOutRequestQueryByTraceID] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			apiLogsOuput, err := aliyun.OutlineModelOutResponseQuery(ctx, traceID, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "[OutlineModelOutResponseQuery] get api logs failed, err: %v", err)
				return nil, &consts.QueryRecordError
			}
			// apiLogsInput 倒序排序
			sort.Slice(apiLogsInput, func(i, j int) bool {
				return apiLogsInput[i].Asctime.After(apiLogsInput[j].Asctime)
			})
			newApiLogsInput := []aliyun.FileProcessLog{}
			for _, log := range apiLogsInput {
				// 解压缩
				inputStr := GetReqRespFromMsg(log.Message, "req:")
				if inputStr == "" {
					inputStr = log.Message
				}
				// 语言判断
				if summaryInfo.SummaryLangType == "en" && !strings.Contains(inputStr, "\"en_mode\":true") {
					continue
				}
				if summaryInfo.SummaryLangType == "zh" && !strings.Contains(inputStr, "\"en_mode\":false") {
					continue
				}
				// 简单详细判断
				if summaryInfo.OutlineType == 1 && !strings.Contains(inputStr, "\"verbose\":false") {
					continue
				}
				if summaryInfo.OutlineType == 2 && !strings.Contains(inputStr, "\"verbose\":true") {
					continue
				}
				// 时间判断
				if summaryInfo.CreateTime.Add(5*time.Second).Format(consts.DateTimeTemplate) > log.Asctime.Format(consts.DateTimeTemplate) {
					newApiLogsInput = append(newApiLogsInput, log)
				}
			}
			if len(newApiLogsInput) > 0 && len(apiLogsOuput) > 0 {
				node.EnterTime = newApiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
				node.FinishTime = newApiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
				node.TraceID = traceID
				return node, nil
			}
			if len(newApiLogsInput) > 0 {
				node.TraceID = traceID
			}
		}
		node.EnterTime = summaryInfo.CreateTime.Format(consts.DateTimeTemplate)
		node.FinishTime = summaryInfo.CreateTime.Format(consts.DateTimeTemplate)
		return node, nil
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_RETRY_FINISH:
		node := makeEmptyNode(processType, entryInfo)
		// 获取traceID
		logs, err := aliyun.TraceIDQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil || len(logs) == 0 {
			hlog.CtxErrorf(ctx, "[NodeApiLogs] get traceID failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		traceID := logs[0].TraceId
		apiLogsInput, err := aliyun.MultiOutlineModelOutRequestQuery(ctx, traceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineModelOutRequestQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.MultiOutlineModelOutResponseQuery(ctx, traceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[MultiOutlineModelOutResponseQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		node.EnterTime = logs[0].Asctime.Format(consts.DateTimeTemplate)
		if len(apiLogsInput) > 0 {
			node.EnterTime = apiLogsInput[0].Asctime.Format(consts.DateTimeTemplate)
		}
		node.FinishTime = logs[0].Asctime.Format(consts.DateTimeTemplate)
		if len(apiLogsOuput) > 0 {
			node.FinishTime = apiLogsOuput[0].Asctime.Format(consts.DateTimeTemplate)
		}
		node.Status = empyrean_lens.ActionStatusEnum_SUCCESS
		node.TraceID = traceID
		return node, nil
	case empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH:
		// 新内容形态可能间隔很久
		end := time.Now()
		apiLogsInput, err := aliyun.NovelFormOutRequestQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NovelFormOutRequestQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		apiLogsOuput, err := aliyun.NovelFormOutResponseQuery(ctx, entryInfo.EntryID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "[NovelFormOutResponseQuery] get process logs failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		return processLogsToNode(processType, append(apiLogsInput, apiLogsOuput...), entryInfo, nil), nil
	}
	return makeEmptyNode(processType, entryInfo), nil
}

func FindSummaryID(ctx context.Context, articleInfo *bi.EntryInfo, processType empyrean_lens.LinkNodeTypeEnum) (string, string, *time.Time, *consts.BizCode) {
	if utils.IsSubscribe(articleInfo.EntryType) {
		// 订阅的summary记录
		summaryType := 0
		switch processType {
		case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
			summaryType = int(plugin.SummaryTypeSummary)
		case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
			summaryType = int(plugin.SummaryTypeViewPoint)
		case empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH:
			summaryType = int(plugin.SummaryTypeSimpleOutline)
		case empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH:
			summaryType = int(plugin.SummaryTypeDetailOutline)
		}
		summaryInfo, err := plugin.NewResourceSummaryDao().FindByEntryTypeAndEntryIDAndSummaryType(ctx, articleInfo.EntryType, articleInfo.EntryID, summaryType)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindByEntryTypeAndEntryIDAndSummaryType] get summary info failed, err: %v", err)
			return "", "", nil, &consts.QueryRecordError
		}
		if summaryInfo == nil {
			return "", "", nil, nil
		}
		return summaryInfo.ID.Hex(), "", &summaryInfo.CreateTime, nil
	} else {
		// 非订阅的summary记录
		outlineType, entryType := 0, 0
		switch processType {
		case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:
			outlineType, entryType = 0, 5
		case empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:
			outlineType, entryType = 0, 6
		case empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH:
			outlineType, entryType = 1, 6
		case empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH:
			outlineType, entryType = 2, 6
		case empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:
			outlineType, entryType = 0, 11
		}
		summaryInfo, err := plugin.NewSummaryDao().FindByUserIDAndUrlAndType(ctx, articleInfo.UserID, articleInfo.EntryURL, entryType, outlineType)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindByUserIDAndUrlAndType] get summary info failed, err: %v", err)
			return "", "", nil, &consts.QueryRecordError
		}
		if summaryInfo.CopyFromSummaryID != "" || utils.IsSubscribe(articleInfo.ParentEntryType) {
			return FindSummaryIDByCopyID(ctx, entryType, outlineType, summaryInfo.ID.Hex(), summaryInfo.PairID, &summaryInfo.CreateTime, articleInfo.ParentEntryType, articleInfo.ParentEntryID)
		}
		if summaryInfo == nil {
			return "", "", nil, nil
		}
		return summaryInfo.ID.Hex(), summaryInfo.PairID, &summaryInfo.CreateTime, nil
	}
}

func FindSummaryIDByCopyID(ctx context.Context, summartType, outlineType int, summaryID, pairID string, summaryCreateAt *time.Time, parentEntryType int, parentEntryID string) (string, string, *time.Time, *consts.BizCode) {
	// parent article info
	parentArticleInfo, err := GetEntryInfo(ctx, empyrean_lens.EntryTypeEnum(parentEntryType), parentEntryID)
	if err != nil {
		return summaryID, pairID, summaryCreateAt, &consts.QueryRecordError
	}
	if parentArticleInfo != nil {
		summaryInfo, err := plugin.NewSummaryDao().FindByUserIDAndTypeAndPairID(ctx, parentArticleInfo.UserID, summartType, outlineType, pairID)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindByUserIDAndTypeAndPairID] get summary info failed, err: %v", err)
			return summaryID, pairID, summaryCreateAt, &consts.QueryRecordError
		}
		if parentArticleInfo.ParentEntryID != "" {
			return FindSummaryIDByCopyID(ctx, summartType, outlineType, summaryInfo.ID.Hex(), pairID, &summaryInfo.CreateTime, parentArticleInfo.ParentEntryType, parentArticleInfo.ParentEntryID)
		}
		return summaryInfo.ID.Hex(), summaryInfo.PairID, &summaryInfo.CreateTime, nil
	}
	return summaryID, pairID, summaryCreateAt, nil
}

func processLogsToNode(nodeType empyrean_lens.LinkNodeTypeEnum, processLogs []aliyun.FileProcessLog, entryInfo *bi.EntryInfo, extra map[string]string) *empyrean_lens.GraphNode {
	if len(processLogs) == 0 {
		return makeEmptyNode(nodeType, entryInfo)
	}
	enterTime := processLogs[0].Asctime.Add(-time.Millisecond * time.Duration(processLogs[0].Cost*1000))
	return &empyrean_lens.GraphNode{
		ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
		Type:       nodeType,
		Name:       consts.LinkNodeTypeName[nodeType],
		EnterTime:  enterTime.Format(consts.DateTimeTemplate),
		FinishTime: processLogs[0].Asctime.Format(consts.DateTimeTemplate),
		Status:     getActionStatus(nodeType, processLogs, entryInfo),
		TraceID:    processLogs[0].TraceId,
		Extra:      extra,
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
				if len(tail.Nodes) > 0 {
					edges[node.ID] = append(edges[node.ID], tail.Nodes[0].ID)
				}
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

func makeEmptyNode(nodeType empyrean_lens.LinkNodeTypeEnum, entryInfo *bi.EntryInfo) *empyrean_lens.GraphNode {
	status := empyrean_lens.ActionStatusEnum_UNREACHEAD
	if nodeType == empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH {
		status = empyrean_lens.ActionStatusEnum_SUCCESS
	}
	staryAt, _ := time.Parse(consts.DateTemplate, consts.LinkTraceStartDate)
	if entryInfo.EntryCreateTime.Before(staryAt) {
		status = empyrean_lens.ActionStatusEnum_NO_LOG
	}
	return &empyrean_lens.GraphNode{
		ID:         empyrean_lens.NodeId(primitive.NewObjectID().Hex()),
		Name:       consts.LinkNodeTypeName[nodeType],
		Type:       nodeType,
		EnterTime:  entryInfo.EntryCreateTime.Format(consts.DateTimeTemplate),
		FinishTime: entryInfo.EntryCreateTime.Format(consts.DateTimeTemplate),
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
				return nodeMapping[fID].Status == empyrean_lens.ActionStatusEnum_FAIL || nodeMapping[fID].Status == empyrean_lens.ActionStatusEnum_WORTHLESS || nodeMapping[fID].Status == empyrean_lens.ActionStatusEnum_NO_LOG
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

func getActionStatus(nodeType empyrean_lens.LinkNodeTypeEnum, processLogs []aliyun.FileProcessLog, entryInfo *bi.EntryInfo) empyrean_lens.ActionStatusEnum {
	if len(processLogs) == 0 {
		return empyrean_lens.ActionStatusEnum_UNREACHEAD
	}
	staryAt, _ := time.Parse(consts.DateTemplate, consts.LinkTraceStartDate)
	if entryInfo != nil && entryInfo.EntryCreateTime.Before(staryAt) {
		return empyrean_lens.ActionStatusEnum_NO_LOG
	}
	switch nodeType {
	case empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:
		// 正排判断是否成功
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "成功") || strings.Contains(processLog.Message, "\"status\": \"ok\"") {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	case empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "ParseEduNode wcd label") {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.After(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "WcdRaw do req error") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
			if strings.Contains(processLog.Message, "wcd text nil") || strings.Contains(processLog.Message, "wcd worthless") {
				return empyrean_lens.ActionStatusEnum_WORTHLESS
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "苏秦解析完成") || strings.Contains(processLog.Message, "PDF解析完成") || strings.Contains(processLog.Message, "pdf解析成功") {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.After(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "苏秦解析异常") || strings.Contains(processLog.Message, "pdf解析异常") || strings.Contains(processLog.Message, "parsing file failed") || strings.Contains(processLog.Message, "请求异常") || strings.Contains(processLog.Message, "read pdf fail") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "OutRequest text_parser resp") {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.After(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if (strings.Contains(processLog.Message, "parse_edu") && strings.Contains(processLog.Message, "error")) || strings.Contains(processLog.Message, "ParseEdu error") || strings.Contains(processLog.Message, "edu parse error") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "ParseEduNode parse end entryId") || (strings.Contains(processLog.Message, "OutRequest edu_parser") && strings.Contains(processLog.Message, "resp") && !strings.Contains(processLog.Message, "edu input filter without sentence.")) {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.After(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "edu parse error") || strings.Contains(processLog.Message, "edu parse fail") || strings.Contains(processLog.Message, "ParseEdu error") || strings.Contains(processLog.Message, "edu input filter without sentence.") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH, empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
		empyrean_lens.LinkNodeTypeEnum_SUMMARY_RETRY_FINISH, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH, empyrean_lens.LinkNodeTypeEnum_OUTLINE_RETRY_FINISH:
		// 正排判断是否成功
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "OutRequest") && strings.Contains(processLog.Message, "resp:") {
				// 解码
				msg := GetReqRespFromMsg(processLog.Message, "resp:")
				if strings.Contains(msg, "data too long") || strings.Contains(msg, "\\\"code\\\": 0") {
					return empyrean_lens.ActionStatusEnum_SUCCESS
				}
			} else if strings.Contains(processLog.Message, "core core_name:") {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	case empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_RETRY_FINISH:
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "multi core node node_name:THEME_ALL_SUMMARY") {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.After(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "不安全") || strings.Contains(processLog.Message, "core error") {
				return empyrean_lens.ActionStatusEnum_FAIL
			}
		}
	case empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH:
		// 正排判断是否成功
		sort.Slice(processLogs, func(i, j int) bool {
			return processLogs[i].Asctime.Before(processLogs[j].Asctime)
		})
		for _, processLog := range processLogs {
			if strings.Contains(processLog.Message, "status Ready") {
				return empyrean_lens.ActionStatusEnum_SUCCESS
			}
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	}
	return empyrean_lens.ActionStatusEnum_SUCCESS
}
