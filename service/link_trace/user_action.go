package link_trace

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func GetUserAction(ctx context.Context, req empyrean_lens.UserActionReq) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 确定时间范围
	var err error
	begin, err := time.ParseInLocation(consts.DateHourMinSecTemplate, req.StartTime, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse start time error, err:%v", err)
		return nil, &consts.RetParamError
	}
	end, err := time.ParseInLocation(consts.DateHourMinSecTemplate, req.EndTime, time.Local)
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
	// 并发查询
	res := []*empyrean_lens.UserActionRespData{}
	wg, mu := sync.WaitGroup{}, sync.Mutex{}
	wg.Add(3)
	// file 数据
	go func() {
		defer wg.Done()
		data, err := findFile(ctx, req, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "find file error, err:%v", err)
			return
		}
		mu.Lock()
		res = append(res, data)
		mu.Unlock()
	}()
	// web reader 数据
	go func() {
		defer wg.Done()
		data, err := findWebReader(ctx, req, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "find web reader error, err:%v", err)
			return
		}
		mu.Lock()
		res = append(res, data)
		mu.Unlock()
	}()
	// multi 数据
	go func() {
		defer wg.Done()
		data, err := findMulti(ctx, req, begin, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "find multi error, err:%v", err)
			return
		}
		mu.Lock()
		res = append(res, data)
		mu.Unlock()
	}()
	wg.Wait()
	// 排序组合
	rows := []*empyrean_lens.UserActionRespRow{}
	for _, v := range res {
		rows = append(rows, v.Rows...)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreateTime > rows[j].CreateTime
	})
	hasNext := false
	for _, v := range res {
		hasNext = hasNext || v.HasNext
	}
	hasNext = hasNext || len(rows) >= int(req.Limit+req.Skip)
	if len(rows) > int(req.Skip) {
		rows = rows[int(req.Skip):]
	}
	if len(rows) > int(req.Limit) {
		rows = rows[:int(req.Limit)]
	}
	// 返回
	return &empyrean_lens.UserActionRespData{
		HasNext: hasNext,
		Rows:    rows,
	}, nil
}

func findFile(ctx context.Context, req empyrean_lens.UserActionReq, begin, end time.Time) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 状态
	status := []int32{}
	switch req.Status {
	case empyrean_lens.ActionStatusEnum_SUCCESS:
		status = []int32{8}
	case empyrean_lens.ActionStatusEnum_FAIL:
		status = []int32{3, 6, 9, 10, 20}
	}
	// 查询数据库
	files, err := []*plugin.File(nil), error(nil)
	if req.Query == "" {
		files, err = plugin.NewFileDao().FindFileByTimeRange(ctx, status, begin, end, 0, req.Skip+req.Limit+1)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindFileByTimeRange] error: %+v", err)
			return nil, &consts.QueryRecordError
		}
	} else {
		files, err = plugin.NewFileDao().FindFileByQueryAndTimeRange(ctx, req.Query, status, begin, end, 0, req.Skip+req.Limit+1)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindFileByQueryAndTimeRange] error: %+v", err)
			return nil, &consts.QueryRecordError
		}
	}
	// 转换
	fileDatas := []*empyrean_lens.UserActionRespRow{}
	for _, file := range files {
		fileDatas = append(fileDatas, fileToActionData("单文档", file))
	}
	return &empyrean_lens.UserActionRespData{
		HasNext: len(fileDatas) >= int(req.Skip+req.Limit+1),
		Rows:    fileDatas,
	}, nil
}

func findWebReader(ctx context.Context, req empyrean_lens.UserActionReq, begin, end time.Time) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 状态
	status := []int32{}
	switch req.Status {
	case empyrean_lens.ActionStatusEnum_SUCCESS:
		status = []int32{0}
	case empyrean_lens.ActionStatusEnum_FAIL:
		status = []int32{2, 3}
	}

	// 查询数据库
	webReaders, err := []*plugin.WebReader(nil), error(nil)
	if req.Query == "" {
		webReaders, err = plugin.NewWebReaderDao().FindWebReaderByTimeRange(ctx, status, begin, end, 0, req.Skip+req.Limit+1)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindWebReaderByTimeRange] error: %+v", err)
			return nil, &consts.QueryRecordError
		}
	} else {
		webReaders, err = plugin.NewWebReaderDao().FindWebReaderByQueryAndTimeRange(ctx, req.Query, status, begin, end, 0, req.Skip+req.Limit+1)
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindWebReaderByQueryAndTimeRange] error: %+v", err)
			return nil, &consts.QueryRecordError
		}
	}
	// 转换
	webReaderDatas := []*empyrean_lens.UserActionRespRow{}
	for _, webReader := range webReaders {
		webReaderDatas = append(webReaderDatas, webReaderToActionData("单文档", webReader))
	}
	return &empyrean_lens.UserActionRespData{
		HasNext: len(webReaderDatas) >= int(req.Skip+req.Limit+1),
		Rows:    webReaderDatas,
	}, nil
}

func findMulti(ctx context.Context, req empyrean_lens.UserActionReq, begin, end time.Time) (*empyrean_lens.UserActionRespData, *consts.BizCode) {
	// 查询数据库
	multis, err := []*plugin.MultiModel(nil), error(nil)
	if req.Query == "" {
		if req.Status == empyrean_lens.ActionStatusEnum_UNK {
			multis, err = plugin.NewMultiDao().FindMultiByTimeRange(ctx, begin, end, 0, req.Skip+req.Limit)
		} else {
			success := req.Status == empyrean_lens.ActionStatusEnum_SUCCESS
			multis, err = plugin.NewMultiDao().FindMultiByStatusAndTimeRange(ctx, success, begin, end, 0, req.Skip+req.Limit)
		}
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindMultiByTimeRange] error: %+v", err)
			return nil, &consts.QueryRecordError
		}
	} else {
		if req.Status == empyrean_lens.ActionStatusEnum_UNK {
			multis, err = plugin.NewMultiDao().FindMultiByQueryAndTimeRange(ctx, req.Query, begin, end, 0, req.Skip+req.Limit)
		} else {
			success := req.Status == empyrean_lens.ActionStatusEnum_SUCCESS
			multis, err = plugin.NewMultiDao().FindMultiByQueryAndStatusAndTimeRange(ctx, req.Query, success, begin, end, 0, req.Skip+req.Limit)
		}
		if err != nil {
			hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] error: %+v", err)
			return nil, &consts.QueryRecordError
		}
	}
	// 转换
	multiDatas := []*empyrean_lens.UserActionRespRow{}
	for _, multi := range multis {
		multiDatas = append(multiDatas, multiActionData("多文档", multi))
	}
	return &empyrean_lens.UserActionRespData{
		HasNext: len(multiDatas) >= int(req.Skip+req.Limit+1),
		Rows:    multiDatas,
	}, nil
}

func fileToActionData(actionName string, file *plugin.File) *empyrean_lens.UserActionRespRow {
	return &empyrean_lens.UserActionRespRow{
		UserID:     file.UserID,
		Channel:    utils.ChannelIntToString(file.ChannelType),
		Title:      file.Name,
		ActionName: actionName,
		FileTypes:  []string{string("pdf")},
		CreateTime: file.CreateTime.Format(consts.DateHourMinSecTemplate),
		Cost:       0, // todo
		Status:     actionStatus("multi", file.Status, 0, 0, 0),
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
		FileTypes:  []string{string("url")},
		CreateTime: webReader.CreateTime.Format(consts.DateHourMinSecTemplate),
		Cost:       0, // todo
		Status:     actionStatus("multi", webReader.Status, 0, 0, 0),
		EntryType:  empyrean_lens.EntryTypeEnum_WEB,
		EntryID:    string(webReader.ID.Hex()),
	}
}

func multiActionData(actionName string, multiModel *plugin.MultiModel) *empyrean_lens.UserActionRespRow {
	fileTypes, pdfNum, urlNum := make([]string, 0), 1, 1
	for _, article := range multiModel.ArticleList {
		if article.EntryType == consts.EntryType(empyrean_lens.EntryTypeEnum_FILE) {
			fileTypes = append(fileTypes, fmt.Sprintf("%s%d", "pdf", pdfNum))
			pdfNum++
		} else if article.EntryType == consts.EntryType(empyrean_lens.EntryTypeEnum_WEB) {
			fileTypes = append(fileTypes, fmt.Sprintf("%s%d", "url", urlNum))
			urlNum++
		}
	}
	return &empyrean_lens.UserActionRespRow{
		UserID:     multiModel.UserID,
		Channel:    utils.ChannelIntToString(multiModel.ChannelType),
		Title:      multiModel.Title,
		ActionName: actionName,
		FileTypes:  fileTypes,
		CreateTime: multiModel.CreateTime.Format(consts.DateHourMinSecTemplate),
		Cost:       0, // todo
		Status:     actionStatus("multi", 0, multiModel.AnalysisStatus, multiModel.MergeStatus, multiModel.SummaryStatus),
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
