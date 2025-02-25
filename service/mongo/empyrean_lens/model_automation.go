package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

type DailyModelAutomationData struct {
	Date           string `json:"date"`
	SingleFailNum  int    `json:"single_fail_num"`
	SingleTotalNum int    `json:"single_total_num"`
	WebFailNum     int    `json:"web_fail_num"`
	WebTotalNum    int    `json:"web_total_num"`
	MultiFailNum   int    `json:"multi_fail_num"`
	MultiTotalNum  int    `json:"multi_total_num"`
}

func GetDailyModelAutomationByTime(ctx context.Context, beginTime, endTime time.Time) ([]DailyModelAutomationData, error) {
	_, err := empyrean_lens.NewModelAutomationDao().GetModelAutomationInfoByTime(ctx, beginTime, endTime)
	if err != nil {
		hlog.CtxErrorf(ctx, "get model automation info error in GetDailyModelAutomationByTime :%v", err)
		return nil, err
	}
	modelCaseResults, err := empyrean_lens.NewModelCaseResultDao().GetModelCaseResultsByTime(ctx, beginTime, endTime)
	if err != nil {
		hlog.CtxErrorf(ctx, "get model case result info error in GetDailyModelAutomationByTime :%v", err)
		return nil, err
	}
	// map<<date, entryType>, modelCaseResults>
	dateEntryTypeModelCaseResultsMap := make(map[string]map[int][]empyrean_lens.ModelCaseResultModel)
	for _, modelCaseResult := range modelCaseResults {
		date := modelCaseResult.UpdateTime.Local().Format("2006-01-02")
		entryType := int(modelCaseResult.FileTypeDetail["entry_type"].(int32))
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
		for entryType, modelCaseResultsTemp := range entryTypeModelCaseResultsMap {
			total, fail := 0, 0
			for _, modelCaseResult := range modelCaseResultsTemp {
				total++
				if modelCaseResult.CaseResult == false {
					fail++
				}
			}
			if entryType == consts.EntryTypeWEB {
				rowData.WebTotalNum = total
				rowData.WebFailNum = fail
			} else if entryType == consts.EntryTypePDF {
				rowData.SingleTotalNum = total
				rowData.SingleFailNum = fail
			} else if entryType == consts.EntryTypeMulti {
				rowData.MultiTotalNum = total
				rowData.MultiFailNum = fail
			}
		}
		resp = append(resp, rowData)
	}
	return resp, nil
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
		fail, tot := 0, 0
		for _, caseResult := range caseResultsTemp {
			tot++
			if caseResult.CaseResult == false {
				fail++
			}
		}
		rowData.FailNum = int64(fail)
		rowData.TotalNum = int64(tot)
		resp = append(resp, &rowData)
	}
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
		resp = append(resp, &empyrean_lens2.ModelCaseResultRespData{
			EntryID:      caseResult.FileEntryId,
			EntryType:    int64(caseResult.FileTypeDetail["entry_type"].(int32)),
			CaseResult:   caseResult.CaseResult,
			TestDuration: caseResult.TestDuration,
			ErrorLog:     caseResult.ErrorLog,
		})
	}
	return resp, nil
}
