package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"errors"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sort"
	"time"
)

type DailyModelAutomationData struct {
	Date           string    `json:"date"`
	SingleFailNum  int       `json:"single_fail_num"`
	SingleTotalNum int       `json:"single_total_num"`
	SingleDocNum   int       `json:"single_doc_num"`
	WebFailNum     int       `json:"web_fail_num"`
	WebTotalNum    int       `json:"web_total_num"`
	WebDocNum      int       `json:"web_doc_num"`
	MultiFailNum   int       `json:"multi_fail_num"`
	MultiTotalNum  int       `json:"multi_total_num"`
	MultiDocNum    int       `json:"multi_doc_num"`
	CreateTime     time.Time `bson:"create_time"`
	UpdateTime     time.Time `bson:"update_time"`
}

func GetDailyModelAutomationByTime(ctx context.Context, beginTime, endTime time.Time, skip, limit int64) ([]DailyModelAutomationData, error) {
	// _, err := empyrean_lens.NewModelAutomationDao().GetModelAutomationInfoByTime(ctx, beginTime, endTime)
	// if err != nil {
	// 	hlog.CtxErrorf(ctx, "get model automation info error in GetDailyModelAutomationByTime :%v", err)
	// 	return nil, err
	// }

	stats, err := empyrean_lens.NewDailyAutomationStatsDao().GetStatsByDateRange(ctx, beginTime, endTime)
	if err == nil && len(stats) > 0 {
		resp := make([]DailyModelAutomationData, len(stats))
		for i, stat := range stats {
			resp[i] = DailyModelAutomationData{
				Date:           stat.Date,
				SingleFailNum:  stat.SingleFailNum,
				SingleTotalNum: stat.SingleTotalNum,
				SingleDocNum:   stat.SingleDocNum,
				WebFailNum:     stat.WebFailNum,
				WebTotalNum:    stat.WebTotalNum,
				WebDocNum:      stat.WebDocNum,
				MultiFailNum:   stat.MultiFailNum,
				MultiTotalNum:  stat.MultiTotalNum,
				MultiDocNum:    stat.MultiDocNum,
				CreateTime:     stat.CreateTime,
				UpdateTime:     stat.UpdateTime,
			}
		}

		// 先对数据进行排序
		sort.Slice(resp, func(i, j int) bool {
			return resp[i].Date > resp[j].Date
		})

		// 返回所有数据，让前端处理分页
		return resp, nil
	}

	modelCaseResults, err := empyrean_lens.NewModelCaseResultDao().GetModelCaseResultsByTime(ctx, beginTime, endTime)
	if err != nil {
		hlog.CtxErrorf(ctx, "get model case result info error in GetDailyModelAutomationByTime :%v", err)
		return nil, err
	}
	hlog.CtxDebugf(ctx, "get data done: %v", len(modelCaseResults))
	// map<<date, entryType>, modelCaseResults>
	dateEntryTypeModelCaseResultsMap := make(map[string]map[int][]empyrean_lens.ModelCaseResultModel)
	for _, modelCaseResult := range modelCaseResults {
		date := modelCaseResult.CreateTime.Format("2006-01-02")
		var entryType int
		if val, ok := modelCaseResult.FileTypeDetail["entry_type"]; ok && val != nil {
			switch v := val.(type) {
			case int:
				entryType = v
			case int32:
				entryType = int(v)
			case int64:
				entryType = int(v)
			case float64:
				entryType = int(v)
			case float32:
				entryType = int(v)
			default:
				return nil, errors.New("entry_type is not valid")
			}
		} else {
			return nil, errors.New("entry_type is not valid")
		}
		//entryType := int(modelCaseResult.FileTypeDetail["entry_type"].(int32))
		if _, ok := dateEntryTypeModelCaseResultsMap[date]; !ok {
			dateEntryTypeModelCaseResultsMap[date] = make(map[int][]empyrean_lens.ModelCaseResultModel)
			dateEntryTypeModelCaseResultsMap[date][entryType] = make([]empyrean_lens.ModelCaseResultModel, 0)
		}
		dateEntryTypeModelCaseResultsMap[date][entryType] = append(dateEntryTypeModelCaseResultsMap[date][entryType], modelCaseResult)
	}
	resp := make([]DailyModelAutomationData, 0)
	for date, entryTypeModelCaseResultsMap := range dateEntryTypeModelCaseResultsMap {
		var rowData DailyModelAutomationData
		rowData.Date = date

		// 添加一个变量来跟踪当天的最后更新时间
		var lastUpdateTime time.Time

		for entryType, modelCaseResultsTemp := range entryTypeModelCaseResultsMap {
			total, fail := 0, 0
			entryIdSet := make(map[string]bool)
			for _, modelCaseResult := range modelCaseResultsTemp {
				if modelCaseResult.UpdateTime.After(lastUpdateTime) {
					lastUpdateTime = modelCaseResult.UpdateTime
				}

				if _, ok := entryIdSet[modelCaseResult.FileEntryId]; !ok {
					entryIdSet[modelCaseResult.FileEntryId] = true
				}
				total++
				if modelCaseResult.CaseResult == false {
					fail++
				}
			}
			if entryType == consts.EntryTypeWEB {
				rowData.WebTotalNum = total
				rowData.WebFailNum = fail
				rowData.WebDocNum = len(entryIdSet)
			} else if entryType == consts.EntryTypePDF {
				rowData.SingleTotalNum = total
				rowData.SingleFailNum = fail
				rowData.SingleDocNum = len(entryIdSet)
			} else if entryType == consts.EntryTypeMulti {
				rowData.MultiTotalNum = total
				rowData.MultiFailNum = fail
				rowData.MultiDocNum = len(entryIdSet)
			}
		}
		rowData.CreateTime = lastUpdateTime
		rowData.UpdateTime = lastUpdateTime
		resp = append(resp, rowData)
	}
	hlog.CtxDebugf(ctx, "get data done: len(resp)=%v", len(resp))
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Date > resp[j].Date
	})

	// 在返回结果前添加分页处理
	if skip >= int64(len(resp)) {
		return []DailyModelAutomationData{}, nil
	}

	end := skip + limit
	if end > int64(len(resp)) {
		end = int64(len(resp))
	}

	return resp[skip:end], nil
}

func GetModelAutomationInfoByTime(ctx context.Context, beginTime, endTime time.Time, entryType int64, failOrTotal bool) ([]*empyrean_lens2.ModelAutomationRespData, error) {
	caseResults, err := empyrean_lens.NewModelCaseResultDao().GetModelCaseResultsByTimeAndEntryType(ctx, beginTime, endTime, entryType, failOrTotal)
	if err != nil {
		hlog.CtxErrorf(ctx, "get model case result info error in GetModelAutomationInfoByTime :%v", err)
		return nil, err
	}
	// map<case_type, caseResult>
	caseTypeCaseResultMap := make(map[string][]empyrean_lens.ModelCaseResultModel)
	for _, caseResult := range caseResults {
		caseType := caseResult.CaseType
		caseTypeCaseResultMap[caseType] = append(caseTypeCaseResultMap[caseType], caseResult)
	}
	var resp []*empyrean_lens2.ModelAutomationRespData
	for caseType, caseResultsTemp := range caseTypeCaseResultMap {
		var rowData empyrean_lens2.ModelAutomationRespData
		rowData.CaseType = caseType
		fail := 0
		// 去重处理， 要计算的是测试失败文档数，而不是测试失败用例数
		entryIdMap := make(map[string]bool)
		for _, caseResult := range caseResultsTemp {
			if _, ok := entryIdMap[caseResult.FileEntryId]; !ok {
				entryIdMap[caseResult.FileEntryId] = true
			}
			if caseResult.CaseResult == false {
				fail++
			}
		}
		rowData.FailNum = int64(fail)
		rowData.TotalNum = int64(len(entryIdMap))
		resp = append(resp, &rowData)
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].CaseType < resp[j].CaseType
	})
	return resp, nil
}

func GetCaseResultsByTime(ctx context.Context, beginTime, endTime time.Time, entryType int64, caseType string, failOrTotal bool) ([]*empyrean_lens2.ModelCaseResultRespData, error) {
	caseResults, err := empyrean_lens.NewModelCaseResultDao().GetModelCaseResultsByTimeAndEntryTypeAndCaseType(ctx, beginTime, endTime, entryType, caseType, failOrTotal)
	if err != nil {
		hlog.CtxErrorf(ctx, "get model case result info error in GetCaseResultsByTime :%v", err)
		return nil, err
	}
	var resp []*empyrean_lens2.ModelCaseResultRespData
	for _, caseResult := range caseResults {
		shareLinkMap, flag := caseResult.FailureDetails["share_link"].(map[string]interface{})
		shareUrl := ""
		if flag {
			if _, ok := shareLinkMap["web"]; ok {
				shareUrl += shareLinkMap["web"].(string)
			}
			if shareUrl != "" {
				shareUrl += "\n"
			}
			if _, ok := shareLinkMap["h5"]; ok {
				shareUrl += shareLinkMap["h5"].(string)
			}
		}
		hlog.CtxDebugf(ctx, "shareUrl: %v", shareUrl)
		var entryType int64
		if val, ok := caseResult.FileTypeDetail["entry_type"]; ok && val != nil {
			switch v := val.(type) {
			case int:
				entryType = int64(v)
			case int32:
				entryType = int64(v)
			case int64:
				entryType = v
			case float32:
				entryType = int64(v)
			case float64:
				entryType = int64(v)
			default:
				return nil, errors.New("entry_type is not valid")
			}
		} else {
			return nil, errors.New("entry_type is not valid")
		}
		resp = append(resp, &empyrean_lens2.ModelCaseResultRespData{
			EntryID:      caseResult.FileEntryId,
			EntryType:    entryType,
			CaseResult:   caseResult.CaseResult,
			TestDuration: caseResult.TestDuration,
			ErrorLog:     caseResult.ErrorLog,
			ShareLink:    shareUrl,
		})
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].EntryID > resp[j].EntryID
	})
	return resp, nil
}

func SaveModelAutomationInfo(ctx context.Context, req map[string]string) (primitive.ObjectID, error) {
	return empyrean_lens.NewModelAutomationDao().SaveModelAutomationInfo(ctx, req)
}

func SaveModelCaseResultInfo(ctx context.Context, req map[string]string) (primitive.ObjectID, error) {
	return empyrean_lens.NewModelCaseResultDao().SaveModelCaseResultInfo(ctx, req)
}
