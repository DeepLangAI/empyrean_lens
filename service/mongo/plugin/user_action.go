package plugin

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/utils"
	"sync"
	"time"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func GetUserAction(ctx context.Context, req empyrean_lens.GetUserActionReq) ([]*empyrean_lens.GetUserActionRespData, *consts.BizCode) {
	begin, _ := time.Parse(consts.DateHourMinuteTemplate, req.StartTime)
	end, _ := time.Parse(consts.DateHourMinuteTemplate, req.EndTime)
	if begin.IsZero() && end.IsZero() {
		end = time.Now()
		begin = time.Now().Add(-6 * time.Hour)
	} else if begin.IsZero() && !end.IsZero() {
		begin = end.Add(-6 * time.Hour)
	} else if !begin.IsZero() && end.IsZero() {
		end = time.Now()
	}

	var res []*empyrean_lens.GetUserActionRespData
	var bizCode *consts.BizCode
	if req.Content == "" {
		res, bizCode = findByTime(ctx, begin, end)
	} else {
		res, bizCode = findByContentAndTime(ctx, req.Content, begin, end)
	}
	if bizCode != nil {
		return nil, bizCode
	}
	return res, bizCode
}

func findByTime(ctx context.Context, begin, end time.Time) ([]*empyrean_lens.GetUserActionRespData, *consts.BizCode) {
	data, set := []*empyrean_lens.GetUserActionRespData{}, map[string]struct{}{}
	mu := &sync.Mutex{}

	funcList := []utillib.AsyncFunc{}
	funcList = append(funcList, func() error {
		files, err := plugin.NewFileDao().FindFileByCreateTime(ctx, begin, end)
		if err != nil {
			return err
		}
		mu.Lock()
		appendFileActionData(ctx, &data, &set, files)
		mu.Unlock()
		return nil
	})

	funcList = append(funcList, func() error {
		readers, err := plugin.NewWebReaderDao().FindWebReaderByCreateTime(ctx, begin, end)
		if err != nil {
			return err
		}
		mu.Lock()
		appendWebReaderActionData(ctx, &data, &set, readers)
		mu.Unlock()
		return nil
	})

	funcList = append(funcList, func() error {
		multiModels, err := plugin.NewMultiDao().FindMultiByCreateTime(ctx, begin, end)
		if err != nil {
			return err
		}
		mFileMap, mURLMap := findMutliEntryUrls(ctx, multiModels, begin, end)
		mu.Lock()
		appendMultiActionData(ctx, &data, &set, multiModels, mFileMap, mURLMap)
		mu.Unlock()
		return nil
	})

	errs := utillib.ParallelExec(ctx, funcList, len(funcList))
	if len(errs) > 0 {
		hlog.CtxErrorf(ctx, "[FindByTime] error: %+v", utils.JSONMarshal(errs))
		return nil, &consts.SystemErr
	}
	return data, nil
}

func findByContentAndTime(ctx context.Context, content string, begin, end time.Time) ([]*empyrean_lens.GetUserActionRespData, *consts.BizCode) {
	data, set := []*empyrean_lens.GetUserActionRespData{}, map[string]struct{}{}
	mu := &sync.Mutex{}

	funcList := []utillib.AsyncFunc{}
	funcList = append(funcList, func() error {
		files, err := plugin.NewFileDao().FindFileByFileURLAndCreateTime(ctx, content, begin, end)
		if err != nil {
			return err
		}
		mu.Lock()
		appendFileActionData(ctx, &data, &set, files)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		file, err := plugin.NewFileDao().FindFileByIdAndCreateTime(ctx, content, begin, end)
		if err != nil && err.Error() != consts.DB_NOT_FOUND_ERR {
			return err
		}
		mu.Lock()
		appendFileActionData(ctx, &data, &set, []plugin.File{file})
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		files, err := plugin.NewFileDao().FindFileByNameAndCreateTime(ctx, content, begin, end)
		if err != nil {
			return err
		}
		mu.Lock()
		appendFileActionData(ctx, &data, &set, files)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		files, err := plugin.NewFileDao().FindFileByUserIdAndCreateTime(ctx, content, begin, end)
		if err != nil {
			return err
		}
		mu.Lock()
		appendFileActionData(ctx, &data, &set, files)
		mu.Unlock()
		return nil
	})

	funcList = append(funcList, func() error {
		web, err := plugin.NewWebReaderDao().FindWebReaderByIdAndCreateTime(ctx, content, begin, end)
		if err != nil && err.Error() != consts.DB_NOT_FOUND_ERR {
			return err
		}
		mu.Lock()
		appendWebReaderActionData(ctx, &data, &set, []plugin.WebReader{web})
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		webs, err := plugin.NewWebReaderDao().FindWebReaderByUserIdAndCreateTime(ctx, content, begin, end)
		if err != nil {
			return err
		}
		mu.Lock()
		appendWebReaderActionData(ctx, &data, &set, webs)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		webs, err := plugin.NewWebReaderDao().FindWebReaderByWebReaderURLAndCreateTime(ctx, content, begin, end)
		if err != nil {
			return err
		}
		mu.Lock()
		appendWebReaderActionData(ctx, &data, &set, webs)
		mu.Unlock()
		return nil
	})

	funcList = append(funcList, func() error {
		multiModel, err := plugin.NewMultiDao().FindMultiByIdAndCreateTime(ctx, content, begin, end)
		if err != nil && err.Error() != consts.DB_NOT_FOUND_ERR {
			return err
		}
		mFileMap, mURLMap := findMutliEntryUrls(ctx, []plugin.MultiModel{multiModel}, begin, end)
		mu.Lock()
		appendMultiActionData(ctx, &data, &set, []plugin.MultiModel{multiModel}, mFileMap, mURLMap)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		multiModels, err := plugin.NewMultiDao().FindMultiByTitleAndCreateTime(ctx, content, begin, end)
		hlog.CtxInfof(ctx, "multi search title...%v", content)
		hlog.CtxInfof(ctx, "%+s", utils.JSONMarshal(multiModels))
		if err != nil {
			return err
		}
		mFileMap, mURLMap := findMutliEntryUrls(ctx, multiModels, begin, end)
		mu.Lock()
		appendMultiActionData(ctx, &data, &set, multiModels, mFileMap, mURLMap)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		multiModels, err := plugin.NewMultiDao().FindMultiByUserIdAndCreateTime(ctx, content, begin, end)
		if err != nil {
			return err
		}
		mFileMap, mURLMap := findMutliEntryUrls(ctx, multiModels, begin, end)
		mu.Lock()
		appendMultiActionData(ctx, &data, &set, multiModels, mFileMap, mURLMap)
		mu.Unlock()
		return nil
	})

	errs := utillib.ParallelExec(ctx, funcList, len(funcList))
	if len(errs) > 0 {
		hlog.CtxErrorf(ctx, "[findByContentAndTime] error: %+v", utils.JSONMarshal(errs))
		return nil, &consts.SystemErr
	}
	return data, nil
}

func findMutliEntryUrls(ctx context.Context, multiModels []plugin.MultiModel, begin, end time.Time) (*sync.Map, *sync.Map) {
	mFileMap, mURLMap, multiWg := &sync.Map{}, &sync.Map{}, &sync.WaitGroup{}
	for _, model := range multiModels {
		id := model.ID
		for _, entry := range model.ArticleList {
			multiWg.Add(1)
			if entry.EntryType == consts.EntryTypePDF {
				go func(e plugin.ArticleEntry) {
					defer multiWg.Done()
					file, err := plugin.NewFileDao().FindFileByIdAndCreateTime(ctx, e.EntryId, begin, end)
					if err != nil {
						return
					}
					urls := []string{file.FileURL}
					if v, ok := mFileMap.Load(id); ok {
						urls = append(urls, v.([]string)...)
						mFileMap.Store(id, urls)
					} else {
						mFileMap.Store(id, urls)
					}
				}(entry)
			} else if entry.EntryType == consts.EntryTypeWEB {
				go func(e plugin.ArticleEntry) {
					defer multiWg.Done()
					web, err := plugin.NewWebReaderDao().FindWebReaderByIdAndCreateTime(ctx, e.EntryId, begin, end)
					if err != nil {
						return
					}
					urls := []string{web.URL}
					if v, ok := mURLMap.Load(id); ok {
						urls = append(urls, v.([]string)...)
						mURLMap.Store(id, urls)
					} else {
						mURLMap.Store(id, urls)
					}
				}(entry)
			}
		}
	}
	multiWg.Wait()
	return mFileMap, mURLMap
}

func appendFileActionData(ctx context.Context, data *[]*empyrean_lens.GetUserActionRespData, set *map[string]struct{}, files []plugin.File) {
	for _, file := range files {
		if file.ID.IsZero() {
			continue
		}
		if _, ok := (*set)[file.ID.Hex()+"/"+consts.PDF]; ok {
			continue
		}
		*data = append(*data, &empyrean_lens.GetUserActionRespData{
			UID:         file.UserID,
			Time:        file.CreateTime.Format(consts.DateTimeTemplate),
			Action:      consts.PDF,
			Title:       file.Name,
			Files:       []string{file.FileURL},
			Cost:        file.UpdateTime.Sub(file.CreateTime).Seconds(),
			Status:      actionStatus(ctx, consts.PDF, file.Status, -1, -1, -1),
			ActionType:  empyrean_lens.ActionType_PDF,
			ID:          file.ID.Hex(),
			ChannelType: empyrean_lens.ChannelType(file.ChannelType),
		})
		(*set)[file.ID.Hex()+"/"+consts.PDF] = struct{}{}
	}
}

func appendWebReaderActionData(ctx context.Context, data *[]*empyrean_lens.GetUserActionRespData, set *map[string]struct{}, webReaders []plugin.WebReader) {
	for _, reader := range webReaders {
		if reader.ID.IsZero() {
			continue
		}
		if _, ok := (*set)[reader.ID.Hex()+"/"+consts.URL]; ok {
			continue
		}
		*data = append(*data, &empyrean_lens.GetUserActionRespData{
			UID:         reader.UserID,
			Time:        reader.CreateTime.Format(consts.DateTimeTemplate),
			Action:      consts.URL,
			Title:       reader.Title,
			Urls:        []string{reader.URL},
			Cost:        reader.UpdateTime.Sub(reader.CreateTime).Seconds(),
			Status:      actionStatus(ctx, consts.URL, reader.Status, -1, -1, -1),
			ActionType:  empyrean_lens.ActionType_PDF,
			ID:          reader.ID.Hex(),
			ChannelType: empyrean_lens.ChannelType(reader.ChannelType),
		})
		(*set)[reader.ID.Hex()+"/"+consts.URL] = struct{}{}
	}
}

func appendMultiActionData(ctx context.Context, data *[]*empyrean_lens.GetUserActionRespData, set *map[string]struct{}, multiModels []plugin.MultiModel, fileMap, webMap *sync.Map) {
	for _, multi := range multiModels {
		if multi.ID.IsZero() {
			continue
		}
		if _, ok := (*set)[multi.ID.Hex()+"/"+consts.MULTI]; ok {
			continue
		}
		var fileUrls, webUrls []string
		if files, ok := fileMap.Load(multi.ID); ok {
			fileUrls = files.([]string)
		}
		if urls, ok := webMap.Load(multi.ID); ok {
			webUrls = urls.([]string)
		}
		*data = append(*data, &empyrean_lens.GetUserActionRespData{
			UID:         multi.UserID,
			Time:        multi.CreateTime.Format(consts.DateTimeTemplate),
			Action:      consts.MULTI,
			Title:       multi.Title,
			Files:       fileUrls,
			Urls:        webUrls,
			Cost:        multi.UpdateTime.Sub(multi.CreateTime).Seconds(),
			Status:      actionStatus(ctx, consts.URL, -1, multi.AnalysisStatus, multi.MergeStatus, multi.SummaryStatus),
			ActionType:  empyrean_lens.ActionType_PDF,
			ID:          multi.ID.Hex(),
			ChannelType: empyrean_lens.ChannelType(multi.ChannelType),
		})
		(*set)[multi.ID.Hex()+"/"+consts.MULTI] = struct{}{}
	}
}

func actionStatus(ctx context.Context, actionType string, status, analysisStatus, mergeStatus, summaryStatus int) empyrean_lens.UserActionStatus {
	switch actionType {
	case consts.PDF:
		if status == consts.PDFSuccessStatus {
			return empyrean_lens.UserActionStatus_Success
		} else {
			return empyrean_lens.UserActionStatus_Fail
		}
	case consts.URL:
		if status == consts.URLSuccessStatus {
			return empyrean_lens.UserActionStatus_Success
		} else {
			return empyrean_lens.UserActionStatus_Fail
		}
	case consts.MULTI:
		if analysisStatus == consts.MultiSuccessAnalysisStatus && mergeStatus == consts.MultiSuccessMergeStatus && summaryStatus == consts.MultiSuccessSummaryStatus {
			return empyrean_lens.UserActionStatus_Success
		} else {
			return empyrean_lens.UserActionStatus_Fail
		}
	default:
		hlog.CtxErrorf(ctx, "error actionType: %s", actionType)
		return -1
	}
}
