package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sort"
	"time"
)

type DailyModelAutomationData struct {
	Date           string `json:"date"`
	SingleFailNum  int    `json:"single_fail_num"`
	SingleTotalNum int    `json:"single_total_num"`
	SingleDocNum   int    `json:"single_doc_num"`
	WebFailNum     int    `json:"web_fail_num"`
	WebTotalNum    int    `json:"web_total_num"`
	WebDocNum      int    `json:"web_doc_num"`
	MultiFailNum   int    `json:"multi_fail_num"`
	MultiTotalNum  int    `json:"multi_total_num"`
	MultiDocNum    int    `json:"multi_doc_num"`
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
			entryIdSet := make(map[string]bool)
			for _, modelCaseResult := range modelCaseResultsTemp {
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
		resp = append(resp, rowData)
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Date > resp[j].Date
	})
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
		resp = append(resp, &empyrean_lens2.ModelCaseResultRespData{
			EntryID:      caseResult.FileEntryId,
			EntryType:    int64(caseResult.FileTypeDetail["entry_type"].(int32)),
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
