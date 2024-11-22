package link_trace

import (
	"context"
	"fmt"
	"sort"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func GetUserAction(ctx context.Context, req empyrean_lens.UserActionReq) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 确定时间范围
	var err error
	begin, err := time.ParseInLocation(consts.DateTimeTemplate, req.StartTime, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse start time error, err:%v", err)
		return nil, &consts.RetParamError
	}
	end, err := time.ParseInLocation(consts.DateTimeTemplate, req.EndTime, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse end time error, err:%v", err)
		return nil, &consts.RetParamError
	}
	if begin.IsZero() && end.IsZero() {
		end = time.Now()
		begin = time.Now().Add(-6 * time.Hour)
	}
	if !begin.Before(end) {
		hlog.CtxErrorf(ctx, "strat time must before end time, start:%v, end:%v", begin, end)
		return nil, &consts.RetParamError
	}
	// 直接从bi获取记录
	rows, bizCode := getUserActionFromBi(ctx, req, begin, end)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get user action from bi error, err:%v", err)
		return nil, bizCode
	}
	// 返回
	return &empyrean_lens.UserActionRespData{
		HasNext: len(rows) == int(req.Limit),
		Rows:    rows,
	}, nil
}

// 从宽表中获取
func getUserActionFromBi(ctx context.Context, req empyrean_lens.UserActionReq, begin, end time.Time) ([]*empyrean_lens.UserActionRespRow, *consts.BizCode) {
	status := []int32{}
	for _, v := range req.Status {
		status = append(status, int32(v))
	}
	if len(status) == 5 {
		status = append(status, int32(empyrean_lens.ActionStatusEnum_WORTHLESS))
		status = append(status, int32(empyrean_lens.ActionStatusEnum_LENGTH_ERROR))
	}
	var entryInfos []*bi.EntryInfo
	// 查询数据库
	if req.Query == "" {
		// 没有query，直接查询
		res, err := bi.NewEntryInfoDao().FindByTimeRange(ctx, status, req.OnlyExternal, begin, end, req.Skip, req.Limit)
		if err != nil {
			return nil, &consts.QueryRecordError
		}
		entryInfos = res
	} else {
		// 有query，先查宽表，再查最近上传未入宽表的
		res, err := bi.NewEntryInfoDao().FindByQueryAndTimeRange(ctx, req.Query, status, req.OnlyExternal, begin, end, req.Skip, req.Limit)
		if err != nil {
			return nil, &consts.QueryRecordError
		}
		entryInfos = res
	}
	// 转换
	resources := []*empyrean_lens.ResourceInfo{}
	rows := []*empyrean_lens.UserActionRespRow{}
	for _, entryInfo := range entryInfos {
		actionRow := entryInfo.TranslateUserActionRow()
		rows = append(rows, actionRow)
		resources = append(resources, actionRow.Resources...)
	}
	// 补充resources信息
	resourceMapping, err := getResourceInfo(ctx, resources)
	if err != nil {
		return nil, &consts.QueryRecordError
	}
	for _, resource := range resources {
		key := fmt.Sprintf("%d_%s", resource.EntryType, resource.EntryID)
		if _, ok := resourceMapping[key]; ok {
			resource.Title = resourceMapping[key].Title
			resource.URL = resourceMapping[key].URL
		}
	}
	return rows, nil
}

func getResourceInfo(ctx context.Context, resources []*empyrean_lens.ResourceInfo) (map[string]*empyrean_lens.ResourceInfo, *consts.BizCode) {
	entryIDs := []string{}
	for _, resource := range resources {
		entryIDs = append(entryIDs, resource.EntryID)
	}
	// 查找记录
	webReaderEntryInfos, err := bi.NewEntryInfoDao().FindByEntryIDs(ctx, entryIDs)
	if err != nil {
		return nil, &consts.QueryRecordError
	}
	resourceMapping := map[string]*empyrean_lens.ResourceInfo{}
	for _, entryInfo := range webReaderEntryInfos {
		key := fmt.Sprintf("%d_%s", entryInfo.EntryType, entryInfo.EntryID)
		resourceMapping[key] = &empyrean_lens.ResourceInfo{
			EntryType: empyrean_lens.EntryTypeEnum(entryInfo.EntryType),
			EntryID:   entryInfo.EntryID,
			Title:     entryInfo.Title,
			URL:       entryInfo.EntryURL,
		}
	}
	return resourceMapping, nil
}

func findFile(ctx context.Context, req empyrean_lens.UserActionReq, begin, end time.Time, onlyOuter bool) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 状态
	status := []int32{}
	for _, v := range req.Status {
		switch v {
		case empyrean_lens.ActionStatusEnum_SUCCESS:
			status = append(status, []int32{8}...)
		case empyrean_lens.ActionStatusEnum_FAIL:
			status = append(status, []int32{3, 6, 9, 10, 20}...)
		}
	}
	if len(status) == 0 {
		return &empyrean_lens.UserActionRespData{
			HasNext: false,
			Rows:    []*empyrean_lens.UserActionRespRow{},
		}, nil
	}
	// 查询数据库
	totalUidMapping := map[string]*bi.UserInfo{}
	files, err := []*plugin.File(nil), error(nil)
	offset, limit := int64(0), req.Limit+req.Skip
	for len(files) < int(req.Limit+req.Skip) {
		entryIDMapping := map[string]struct{}{}
		fileSplit := []*plugin.File(nil)
		if req.Query == "" {
			fileSplit, err = plugin.NewFileDao().FindFileByTimeRange(ctx, status, begin, end, offset, limit)
			if err != nil {
				hlog.CtxErrorf(ctx, "[FindFileByTimeRange] error: %+v", err)
				return nil, &consts.QueryRecordError
			}
		} else {
			fileSplit, err = plugin.NewFileDao().FindFileByQueryAndTimeRange(ctx, req.Query, status, begin, end, offset, limit)
			if err != nil {
				hlog.CtxErrorf(ctx, "[FindFileByQueryAndTimeRange] error: %+v", err)
				return nil, &consts.QueryRecordError
			}
		}
		if len(fileSplit) == 0 {
			break
		}
		if onlyOuter {
			// 获取用户类型
			uids, uidMapping := []string{}, map[string]*bi.UserInfo{}
			for _, file := range fileSplit {
				if _, ok := totalUidMapping[file.UserID]; !ok {
					uidMapping[file.UserID] = &bi.UserInfo{}
				}
			}
			for uid := range uidMapping {
				uids = append(uids, uid)
			}
			if len(uids) > 0 {
				userInfos, err := bi.NewUserInfoDao().FindFileByUids(ctx, uids)
				if err != nil {
					hlog.CtxErrorf(ctx, "[FindFileByQueryAndTimeRange] error: %+v", err)
					return nil, &consts.QueryRecordError
				}
				for _, userInfo := range userInfos {
					totalUidMapping[userInfo.UserID] = userInfo
				}
			}
			for _, file := range fileSplit {
				if _, ok := entryIDMapping[file.ID.Hex()]; !ok {
					if userInfo, ok := totalUidMapping[file.UserID]; ok {
						if userInfo.UserType == bi.ExternalUser {
							files = append(files, file)
						}
					}
				}
				entryIDMapping[file.ID.Hex()] = struct{}{}
			}
		} else {
			files = append(files, fileSplit...)
			break
		}
		offset += limit
	}
	// 转换
	fileDatas := []*empyrean_lens.UserActionRespRow{}
	for _, file := range files {
		actionName := "单文档pdf"
		if file.MultiId != "" {
			actionName = "多文档-单文档pdf"
		}
		fileDatas = append(fileDatas, fileToActionData(actionName, file))
	}
	return &empyrean_lens.UserActionRespData{
		HasNext: len(fileDatas) >= int(req.Skip+req.Limit+1),
		Rows:    fileDatas,
	}, nil
}

func findWebReader(ctx context.Context, req empyrean_lens.UserActionReq, begin, end time.Time, onlyOuter bool) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 状态
	status := []int32{}
	for _, v := range req.Status {
		switch v {
		case empyrean_lens.ActionStatusEnum_SUCCESS:
			status = append(status, []int32{0}...)
		case empyrean_lens.ActionStatusEnum_FAIL:
			status = append(status, []int32{2, 3}...)
		}
	}
	if len(status) == 0 {
		return &empyrean_lens.UserActionRespData{
			HasNext: false,
			Rows:    []*empyrean_lens.UserActionRespRow{},
		}, nil
	}
	// 查询数据库
	totalUidMapping := map[string]*bi.UserInfo{}
	webReaders, err := []*plugin.WebReader(nil), error(nil)
	offset, limit := int64(0), req.Limit+req.Skip
	for len(webReaders) < int(req.Limit+req.Skip) {
		entryIDMapping := map[string]struct{}{}
		webReaderSplit := []*plugin.WebReader(nil)
		if req.Query == "" {
			webReaderSplit, err = plugin.NewWebReaderDao().FindWebReaderByTimeRange(ctx, status, begin, end, offset, limit)
			if err != nil {
				hlog.CtxErrorf(ctx, "[FindWebReaderByTimeRange] error: %+v", err)
				return nil, &consts.QueryRecordError
			}
		} else {
			webReaderSplit, err = plugin.NewWebReaderDao().FindWebReaderByQueryAndTimeRange(ctx, req.Query, status, begin, end, offset, limit)
			if err != nil {
				hlog.CtxErrorf(ctx, "[FindWebReaderByQueryAndTimeRange] error: %+v", err)
				return nil, &consts.QueryRecordError
			}
		}
		if len(webReaderSplit) == 0 {
			break
		}
		if onlyOuter {
			// 获取用户类型
			uids, uidMapping := []string{}, map[string]*bi.UserInfo{}
			for _, webReader := range webReaderSplit {
				if _, ok := totalUidMapping[webReader.UserID]; !ok {
					uidMapping[webReader.UserID] = &bi.UserInfo{}
				}
			}
			for uid := range uidMapping {
				uids = append(uids, uid)
			}
			if len(uids) > 0 {
				userInfos, err := bi.NewUserInfoDao().FindFileByUids(ctx, uids)
				if err != nil {
					hlog.CtxErrorf(ctx, "[FindWebReaderByQueryAndTimeRange] error: %+v", err)
					return nil, &consts.QueryRecordError
				}
				for _, userInfo := range userInfos {
					totalUidMapping[userInfo.UserID] = userInfo
				}
			}
			for _, webReader := range webReaderSplit {
				if _, ok := entryIDMapping[webReader.ID.Hex()]; !ok {
					if userInfo, ok := totalUidMapping[webReader.UserID]; ok {
						if userInfo.UserType == bi.ExternalUser {
							webReaders = append(webReaders, webReader)
						}
					}
					entryIDMapping[webReader.ID.Hex()] = struct{}{}
				}
			}
		} else {
			webReaders = append(webReaders, webReaderSplit...)
			break
		}
		offset += limit
	}
	// 转换
	webReaderDatas := []*empyrean_lens.UserActionRespRow{}
	for _, webReader := range webReaders {
		actionName := "单文档web"
		if webReader.MultiId != "" {
			actionName = "多文档-单文档web"
		}
		webReaderDatas = append(webReaderDatas, webReaderToActionData(actionName, webReader))
	}
	return &empyrean_lens.UserActionRespData{
		HasNext: len(webReaderDatas) >= int(req.Skip+req.Limit+1),
		Rows:    webReaderDatas,
	}, nil
}

func findMulti(ctx context.Context, req empyrean_lens.UserActionReq, begin, end time.Time, onlyOuter bool) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 查询数据库
	totalUidMapping := map[string]*bi.UserInfo{}
	multis, err := []*plugin.MultiModel(nil), error(nil)
	offset, limit := int64(0), req.Limit+req.Skip
	for len(multis) < int(req.Limit+req.Skip) {
		entryIDMapping := map[string]struct{}{}
		multisSplit := []*plugin.MultiModel(nil)
		for _, v := range req.Status {
			multisItem := []*plugin.MultiModel{}
			if v == empyrean_lens.ActionStatusEnum_SUCCESS {
				multisItem, err = plugin.NewMultiDao().FindSuccessMultiByQueryAndStatusAndTimeRange(ctx, req.Query, begin, end, offset, limit)
			} else if v == empyrean_lens.ActionStatusEnum_FAIL {
				multisItem, err = plugin.NewMultiDao().FindFailMultiByQueryAndStatusAndTimeRange(ctx, req.Query, begin, end, offset, limit)
			}
			if err != nil {
				hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] error: %+v", err)
				return nil, &consts.QueryRecordError
			}
			for _, multi := range multisItem {
				if _, ok := entryIDMapping[multi.ID.Hex()]; !ok {
					multisSplit = append(multisSplit, multi)
					entryIDMapping[multi.ID.Hex()] = struct{}{}
				}
			}
		}
		if len(multisSplit) == 0 {
			break
		}
		if onlyOuter {
			// 获取用户类型
			uids, uidMapping := []string{}, map[string]*bi.UserInfo{}
			for _, multi := range multisSplit {
				if _, ok := totalUidMapping[multi.UserID]; !ok {
					uidMapping[multi.UserID] = &bi.UserInfo{}
				}
			}
			for uid := range uidMapping {
				uids = append(uids, uid)
			}
			if len(uids) > 0 {
				userInfos, err := bi.NewUserInfoDao().FindFileByUids(ctx, uids)
				if err != nil {
					hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] error: %+v", err)
					return nil, &consts.QueryRecordError
				}
				for _, userInfo := range userInfos {
					totalUidMapping[userInfo.UserID] = userInfo
				}
			}
			for _, multi := range multisSplit {
				if userInfo, ok := totalUidMapping[multi.UserID]; ok {
					if userInfo.UserType == bi.ExternalUser {
						multis = append(multis, multi)
					}
				}
			}
		} else {
			multis = append(multis, multisSplit...)
			break
		}
		offset += limit
	}
	sort.Slice(multis, func(i, j int) bool {
		return multis[i].CreateTime.After(multis[j].CreateTime)
	})
	// 获取文章信息
	fileIDs, webReaderIDs := []string{}, []string{}
	for _, multi := range multis {
		for _, article := range multi.ArticleList {
			if article.EntryType == consts.EntryType(empyrean_lens.EntryTypeEnum_FILE) {
				fileIDs = append(fileIDs, string(article.EntryId))
			} else if article.EntryType == consts.EntryType(empyrean_lens.EntryTypeEnum_WEB) {
				webReaderIDs = append(webReaderIDs, string(article.EntryId))
			}
		}
	}
	fileMapping, _ := plugin.NewFileDao().FindFileByIds(ctx, fileIDs)
	webReaderMapping, _ := plugin.NewWebReaderDao().FindWebReaderByIds(ctx, webReaderIDs)
	// 转换
	multiDatas := []*empyrean_lens.UserActionRespRow{}
	for _, multi := range multis {
		multiDatas = append(multiDatas, multiActionData("多文档", multi, fileMapping, webReaderMapping))
	}
	return &empyrean_lens.UserActionRespData{
		HasNext: len(multiDatas) >= int(req.Skip+req.Limit+1),
		Rows:    multiDatas,
	}, nil
}

func findFromMongo(ctx context.Context, rows []*empyrean_lens.UserActionRespRow) ([]*empyrean_lens.UserActionRespRow, *consts.BizCode) {
	// 查数据库
	entryIDs := []string{}
	for _, row := range rows {
		entryIDs = append(entryIDs, row.EntryID)
	}
	entryInfos, err := bi.NewEntryInfoDao().FindByEntryIDs(ctx, entryIDs)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryIDs] error: %+v", err)
		return nil, &consts.QueryRecordError
	}
	// 替换数据
	entryMapping := map[string]*bi.EntryInfo{}
	for _, entryInfo := range entryInfos {
		key := fmt.Sprintf("%d_%s", entryInfo.EntryType, entryInfo.EntryID)
		entryMapping[key] = entryInfo
	}
	newRows := []*empyrean_lens.UserActionRespRow{}
	for _, row := range rows {
		key := fmt.Sprintf("%d_%s", row.EntryType, row.EntryID)
		resources := row.Resources
		if entryInfo, ok := entryMapping[key]; ok {
			if entryInfo.ParentEntryID == "" {
				newRow := entryInfo.TranslateUserActionRow()
				newRow.Resources = resources
				newRows = append(newRows, newRow)
			}
		} else {
			newRows = append(newRows, row)
		}
	}
	return newRows, nil
}

func fileToActionData(actionName string, file *plugin.File) *empyrean_lens.UserActionRespRow {
	return &empyrean_lens.UserActionRespRow{
		UserID:     file.UserID,
		Channel:    utils.ChannelIntToString(file.ChannelType),
		Title:      file.Name,
		ActionName: actionName,
		Resources: []*empyrean_lens.ResourceInfo{
			{
				EntryID:   file.ID.Hex(),
				EntryType: empyrean_lens.EntryTypeEnum_FILE,
				Title:     file.Name,
				URL:       file.FileURL,
			},
		},
		CreateTime: file.CreateTime.Format(consts.DateTimeTemplate),
		Cost:       0, // todo
		Status:     actionStatus(consts.PDF, file.Status, 0, 0, 0),
		EntryType:  empyrean_lens.EntryTypeEnum_FILE,
		EntryID:    string(file.ID.Hex()),
	}
}

func webReaderToActionData(actionName string, webReader *plugin.WebReader) *empyrean_lens.UserActionRespRow {
	return &empyrean_lens.UserActionRespRow{
		UserID:     webReader.UserID,
		Channel:    utils.ChannelIntToString(webReader.ChannelType),
		Title:      webReader.Title,
		ActionName: actionName,
		Resources: []*empyrean_lens.ResourceInfo{
			{
				EntryID:   webReader.ID.Hex(),
				EntryType: empyrean_lens.EntryTypeEnum_WEB,
				Title:     webReader.Title,
				URL:       webReader.URL,
			},
		},
		CreateTime: webReader.CreateTime.Format(consts.DateTimeTemplate),
		Cost:       0, // todo
		Status:     actionStatus(consts.URL, webReader.Status, 0, 0, 0),
		EntryType:  empyrean_lens.EntryTypeEnum_WEB,
		EntryID:    string(webReader.ID.Hex()),
	}
}

func multiActionData(actionName string, multiModel *plugin.MultiModel, fileMapping map[string]*plugin.File, webReaderMapping map[string]*plugin.WebReader) *empyrean_lens.UserActionRespRow {
	resources := make([]*empyrean_lens.ResourceInfo, 0)
	for _, article := range multiModel.ArticleList {
		if article.EntryType == consts.EntryType(empyrean_lens.EntryTypeEnum_FILE) {
			if file, ok := fileMapping[string(article.EntryId)]; ok {
				resources = append(resources, &empyrean_lens.ResourceInfo{
					EntryID:   article.EntryId,
					EntryType: empyrean_lens.EntryTypeEnum_FILE,
					Title:     file.Name,
					URL:       file.FileURL,
				})
			}
		} else if article.EntryType == consts.EntryType(empyrean_lens.EntryTypeEnum_WEB) {
			if webReader, ok := webReaderMapping[string(article.EntryId)]; ok {
				resources = append(resources, &empyrean_lens.ResourceInfo{
					EntryID:   article.EntryId,
					EntryType: empyrean_lens.EntryTypeEnum_WEB,
					Title:     webReader.Title,
					URL:       webReader.URL,
				})
			}
		}
	}
	return &empyrean_lens.UserActionRespRow{
		UserID:     multiModel.UserID,
		Channel:    utils.ChannelIntToString(multiModel.ChannelType),
		Title:      multiModel.Title,
		ActionName: actionName,
		Resources:  resources,
		CreateTime: multiModel.CreateTime.Format(consts.DateTimeTemplate),
		Cost:       0, // todo
		Status:     actionStatus(consts.MULTI, 0, multiModel.AnalysisStatus, multiModel.MergeStatus, multiModel.SummaryStatus),
		EntryType:  empyrean_lens.EntryTypeEnum_MULTI,
		EntryID:    string(multiModel.ID.Hex()),
	}
}

func actionStatus(actionType string, status, analysisStatus, mergeStatus, summaryStatus int) empyrean_lens.ActionStatusEnum {
	switch actionType {
	case consts.PDF:
		if status == consts.PDFSuccessStatus {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		} else {
			return empyrean_lens.ActionStatusEnum_FAIL
		}
	case consts.URL:
		if status == consts.URLSuccessStatus {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		} else {
			return empyrean_lens.ActionStatusEnum_FAIL
		}
	case consts.MULTI:
		if analysisStatus == consts.MultiSuccessAnalysisStatus && mergeStatus == consts.MultiSuccessMergeStatus && summaryStatus == consts.MultiSuccessSummaryStatus {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		} else {
			return empyrean_lens.ActionStatusEnum_FAIL
		}
	default:
		return -1
	}
}
