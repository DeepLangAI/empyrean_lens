package link_trace

import (
	"context"
	"strings"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/utils/gse"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
)

func UpdateEntryInfo(ctx context.Context, req empyrean_lens.UpdateEntryInfoReq) *consts.BizCode {
	if req.WebSite != "" {
		if err := UpdateEntryInfoWebSite(ctx, req); err != nil {
			hlog.CtxErrorf(ctx, "UpdateEntryInfoWebSite err: %v", err)
			return &consts.QueryRecordError
		}
	} else if req.ActionName != "" {
		if err := UpdateEntryInfoActionName(ctx, req); err != nil {
			hlog.CtxErrorf(ctx, "UpdateActionName err: %v", err)
			return &consts.QueryRecordError
		}
	} else if req.EntryID != "" {
		if err := UpdateEntryUrl(ctx, req.EntryID, int(req.EntryType)); err != nil {
			hlog.CtxErrorf(ctx, "UpdateEntryUrl err: %v", err)
			return &consts.QueryRecordError
		}
	} else {
		if err := UpdateSubscriptionUserID(ctx); err != nil {
			hlog.CtxErrorf(ctx, "UpdateSubscriptionUserID err: %v", err)
			return &consts.QueryRecordError
		}
	}

	return nil
}

func UpdateEntryInfoWebSite(ctx context.Context, req empyrean_lens.UpdateEntryInfoReq) error {
	switch req.WebSite {
	case empyrean_lens.WebSiteLingowhalePlugin: // 插件
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_PdfPlugin),
			int32(empyrean_lens.ChannelType_UrlPlugin),
			int32(empyrean_lens.ChannelType_UrlPluginMenu),
		})
	case empyrean_lens.WebSiteLingowhaleWeb: // web端
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_PdfPc),
			int32(empyrean_lens.ChannelType_PdfReader),
			int32(empyrean_lens.ChannelType_PdfPcDb),
			int32(empyrean_lens.ChannelType_PdfWebReader),
			int32(empyrean_lens.ChannelType_UrlPc),
			int32(empyrean_lens.ChannelType_UrlPcDb),
			int32(empyrean_lens.ChannelType_UrlReader),
			int32(empyrean_lens.ChannelType_WebLingoUrl),
			int32(empyrean_lens.ChannelType_WebLingoPdf),
			int32(empyrean_lens.ChannelType_WebLingoMulti),
		})
	case empyrean_lens.WebSiteLingowhaleMini: // 小程序
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_MiniUrl),
			int32(empyrean_lens.ChannelType_MiniPdf),
		})
	case empyrean_lens.WebSiteLingowhaleHelper: // 小助手
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_WechatUrl),
			int32(empyrean_lens.ChannelType_WechatPdf),
		})
	case empyrean_lens.WebSiteLingowhaleAndroid: // 语鲸app-android
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_AndroidUrl),
			int32(empyrean_lens.ChannelType_AndroidPdf),
			int32(empyrean_lens.ChannelType_AndroidMulti),
			int32(empyrean_lens.ChannelType_AndroidFeedback),
		})
	case empyrean_lens.WebSiteLingowhaleIos: // 语鲸app-ios
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_IosUrl),
			int32(empyrean_lens.ChannelType_IosPdf),
			int32(empyrean_lens.ChannelType_IosMulti),
			int32(empyrean_lens.ChannelType_IosFeedback),
		})
	case empyrean_lens.WebSiteLingowhaleH5: // 语鲸h5
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_H5Page),
		})
	case empyrean_lens.WebSiteLingowhaleDesktopWin: // 桌面端-win
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_DesktopUrl),
			int32(empyrean_lens.ChannelType_DesktopPdf),
		})
	case empyrean_lens.WebSiteLingowhaleSubscribe: // 订阅
		return bi.NewEntryInfoDao().UpdateWebSite(ctx, req.WebSite, []int32{
			int32(empyrean_lens.ChannelType_All),
		})
	}
	return nil
}

func UpdateEntryInfoActionName(ctx context.Context, req empyrean_lens.UpdateEntryInfoReq) error {
	switch req.ActionName {
	case empyrean_lens.ActionNameSingleWeb:
		// entry_type = 7, multi_id = "", or(parent_entry_type = 7, parent_entry_id = "")
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 7},
			{"$or": []bson.M{{"multi_id": bson.M{"$exists": false}}, {"multi_id": bson.M{"$eq": ""}}}},
			{"$or": []bson.M{{"parent_entry_type": 7}, {"parent_entry_id": ""}}},
		}})
	case empyrean_lens.ActionNameSinglePdf:
		// entry_type = 10, multi_id = "", or(parent_entry_type = 10, parent_entry_id = "")
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 10},
			{"$or": []bson.M{{"multi_id": bson.M{"$exists": false}}, {"multi_id": bson.M{"$eq": ""}}}},
			{"$or": []bson.M{{"parent_entry_type": 10}, {"parent_entry_id": ""}}},
		}})
	case empyrean_lens.ActionNameMulti:
		// entry_type = 12, multi_id = "", parent_entry_type != 121
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 12},
			{"$or": []bson.M{{"multi_id": bson.M{"$exists": false}}, {"multi_id": bson.M{"$eq": ""}}}},
			{"parent_entry_type": bson.M{"$ne": 121}},
		}})
	case empyrean_lens.ActionNameMultiWeb:
		// entry_type = 7, multi_id != ""
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 7},
			{"multi_id": bson.M{"$ne": ""}},
		}})
	case empyrean_lens.ActionNameMultiPdf:
		// entry_type = 10, multi_id != ""
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 10},
			{"multi_id": bson.M{"$ne": ""}},
		}})
	case empyrean_lens.ActionNameSummaryCH:
		// entry_type = 5, summary_language != en
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 5},
			{"$or": []bson.M{{"summary_language": bson.M{"$exists": false}}, {"summary_language": bson.M{"$ne": "en"}}}},
		}})
	case empyrean_lens.ActionNameSimpleOutlineCH:
		// entry_type = 6, summary_language != en, outline_type != 2
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 6},
			{"$or": []bson.M{{"summary_language": bson.M{"$exists": false}}, {"summary_language": bson.M{"$ne": "en"}}}},
			{"$or": []bson.M{{"outline_type": bson.M{"$exists": false}}, {"outline_type": bson.M{"$ne": 2}}}},
		}})
	case empyrean_lens.ActionNameDetailOutlineCH:
		// entry_type = 6, summary_language != en, outline_type = 2
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 6},
			{"$or": []bson.M{{"summary_language": bson.M{"$exists": false}}, {"summary_language": bson.M{"$ne": "en"}}}},
			{"outline_type": 2},
		}})
	case empyrean_lens.ActionNameKeyInfoCH:
		// entry_type = 11, summary_language != en
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 11},
			{"$or": []bson.M{{"summary_language": bson.M{"$exists": false}}, {"summary_language": bson.M{"$ne": "en"}}}},
		}})
	case empyrean_lens.ActionNameSummaryEN:
		// entry_type = 5, summary_language = en
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 5},
			{"summary_language": "en"},
		}})
	case empyrean_lens.ActionNameSimpleOutlineEN:
		// entry_type = 6, summary_language = en, outline_type != 2
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 6},
			{"summary_language": "en"},
			{"$or": []bson.M{{"outline_type": bson.M{"$exists": false}}, {"outline_type": bson.M{"$ne": 2}}}},
		}})
	case empyrean_lens.ActionNameDetailOutlineEN:
		// entry_type = 6, summary_language = en, outline_type = 2
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 6},
			{"summary_language": "en"},
			{"outline_type": 2},
		}})
	case empyrean_lens.ActionNameKeyInfoEN:
		// entry_type = 11, summary_language = en
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 11},
			{"summary_language": "en"},
		}})
	case empyrean_lens.ActionNameMultiOutline:
		// entry_type = 126
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 126},
		}})
	case empyrean_lens.ActionNameSubscribeWeb:
		// entry_type = 71
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 71},
		}})
	case empyrean_lens.ActionNameSubscribePdf:
		// entry_type = 101
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 101},
		}})
	case empyrean_lens.ActionNameSubscribeMulti:
		// entry_type = 121
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 121},
		}})
	case empyrean_lens.ActionNameSubscribeCopyWeb:
		// entry_type = 7, parent_entry_type = 71
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 7},
			{"parent_entry_type": 71},
		}})
	case empyrean_lens.ActionNameSubscribeCopyPdf:
		// entry_type = 10, parent_entry_type = 101
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 10},
			{"parent_entry_type": 101},
		}})
	case empyrean_lens.ActionNameSubscribeCopyMulti:
		// entry_type = 12, parent_entry_type = 121
		return bi.NewEntryInfoDao().UpdateActionName(ctx, req.ActionName, bson.M{"$and": []bson.M{
			{"entry_type": 12},
			{"parent_entry_type": 121},
		}})
	}
	return nil
}

func UpdateEntryUrl(ctx context.Context, entryID string, entryType int) *consts.BizCode {
	// 从mongo获取记录
	entryInfo, err := bi.NewEntryInfoDao().FindByEntryIDAndEntryType(ctx, entryID, entryType)
	if err != nil || entryInfo == nil {
		hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
		return &consts.QueryRecordError
	}
	if entryInfo.UserID == "" {
		entryInfo.UserID = "resource_server"
	}
	// 更新text索引
	if entryInfo.ContentIndex != "" {
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
	}
	// 更新url
	// 多文档
	if len(entryInfo.MultiArticles) > 1 {
		// 获取文章信息
		for idx, article := range entryInfo.MultiArticles {
			articleInfo, err := GetEntryInfo(ctx, empyrean_lens.EntryTypeEnum(article.EntryType), article.EntryId)
			if err != nil {
				hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
				return &consts.QueryRecordError
			}
			entryInfo.MultiArticles[idx].URL = articleInfo.EntryURL
		}
		// 更新记录
		bi.NewEntryInfoDao().SaveEntryInfo(ctx, entryInfo)
		return nil
	}
	// 单文档
	if entryInfo.SourceEntryInfo.EntryID != "" {
		articleInfo, err := GetEntryInfo(ctx, empyrean_lens.EntryTypeEnum(entryInfo.SourceEntryInfo.EntryType), entryInfo.SourceEntryInfo.EntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
			return &consts.QueryRecordError
		}
		entryInfo.SourceEntryInfo.URL = articleInfo.EntryURL
	} else {
		// 获取文章信息
		articleInfo, err := GetEntryInfo(ctx, empyrean_lens.EntryTypeEnum(entryInfo.EntryType), entryInfo.EntryID)
		if err != nil {
			hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
			return &consts.QueryRecordError
		}
		entryInfo.EntryURL = articleInfo.EntryURL
	}
	// 更新记录
	bi.NewEntryInfoDao().SaveEntryInfo(ctx, entryInfo)
	return nil
}

func UpdateSubscriptionUserID(ctx context.Context) error {
	return bi.NewEntryInfoDao().UpdateSubscriptionUserID(ctx)
}
