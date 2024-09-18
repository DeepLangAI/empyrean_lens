package empyrean_lens

import (
	"context"
	"empyrean_lens/consts"
	aliyun2 "empyrean_lens/dal/aliyun"
	el "empyrean_lens/dal/mongo/empyrean_lens"
	aliyun3 "empyrean_lens/service/aliyun"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"sort"
	"time"
)

func apiFailureRate(ctx context.Context, beginTime, endTime time.Time) (int, int) {
	failureModels, err := el.NewApifailureModelDao().FindTimespanFailure(ctx, beginTime, endTime)
	if err != nil {
		return 0, 0
	}
	totalReq := 0
	errReq := 0
	for _, model := range failureModels {
		if model.ApiName != "当日总览" {
			continue
		}
		totalReq += int(model.TotalCnt)
		errReq += int(model.FailCnt)
	}
	return errReq, totalReq
}

func SystemRealtimeReport(ctx context.Context) (*aliyun3.RealtimeReport, error) {
	report := &aliyun3.RealtimeReport{
		Availability: aliyun3.Metric{},
		TotalRequest: aliyun3.Metric{},
		ErrorRequest: aliyun3.Metric{},
		ProbeFailCnt: aliyun3.Metric{},
	}

	now := time.Now()
	beginTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	failureModels, err := el.NewApifailureModelDao().FindTimespanFailure(ctx, beginTime, now)
	if err != nil {
		return nil, err
	}
	probeFailRates, _, err := aliyun3.ProbeTimespanFailRate(ctx, consts.TIMESPAN_TODAY)
	if err != nil {
		return nil, err
	}
	slowQueryRates, err := aliyun3.RealtimeSlowqueryLoganlz(ctx)
	if err != nil {
		return nil, err
	}

	for _, model := range failureModels {
		if model.ApiName != "当日总览" {
			continue
		}
		report.TotalRequest.Value += int(model.TotalCnt)
		report.ErrorRequest.Value += int(model.FailCnt)
	}
	fail_0, total_0 := 0, 0
	todayNginxLogs, err := aliyun2.NginxLogsToday(ctx)
	if err != nil {
		return nil, err
	}
	for _, log := range todayNginxLogs {
		if log.Status != "200" {
			fail_0 += 1
		}
	}
	total_0 = len(todayNginxLogs)

	fail_1, total_1 := apiFailureRate(ctx, beginTime.AddDate(0, 0, -1), beginTime)
	fail_7, total_7 := apiFailureRate(ctx, beginTime.AddDate(0, 0, -7), beginTime.AddDate(0, 0, -6))
	report.TotalRequest.Value = total_0
	report.TotalRequest.DayOverDay = utils.DeltaPercent(float64(total_1), float64(total_0))
	report.TotalRequest.WeekOverWeek = utils.DeltaPercent(float64(total_7), float64(total_0))

	report.ErrorRequest.Value = fail_0
	report.ErrorRequest.DayOverDay = utils.DeltaPercent(float64(fail_1), float64(fail_0))
	report.ErrorRequest.WeekOverWeek = utils.DeltaPercent(float64(fail_7), float64(fail_0))

	systemScoreFactor := utils.SystemStablityFactor{
		ApiFailRate:   float64(report.ErrorRequest.Value) / float64(report.TotalRequest.Value),
		SlowQueryRate: slowQueryRates.Value / 100.0,
		ProbeFailRate: probeFailRates[time.Now().Format("2006-01-02")] / 100.0,
	}
	score_0 := utils.ComputeStablityScore(systemScoreFactor)
	scoreModel_1, err := el.NewSystemScoreDao().FindScoreByTime(ctx, beginTime.AddDate(0, 0, -1))
	score_1 := 0.0
	if err == nil {
		score_1 = scoreModel_1.Score
	}

	scoreModel_7, err := el.NewSystemScoreDao().FindScoreByTime(ctx, beginTime.AddDate(0, 0, -7))
	score_7 := 0.0
	if err == nil {
		score_7 = scoreModel_7.Score
	}
	report.Availability.Value = score_0
	report.Availability.DayOverDay = utils.DeltaPercent(score_1, float64(score_0))
	report.Availability.WeekOverWeek = utils.DeltaPercent(score_7, float64(score_0))

	probeLogAnlz := aliyun3.RealtimeProbeLoganlz(ctx)
	if err != nil {
		return nil, err
	}
	report.ProbeFailCnt.Value = probeLogAnlz.Value
	report.ProbeFailCnt.DayOverDay = probeLogAnlz.DayOverDay
	report.ProbeFailCnt.WeekOverWeek = probeLogAnlz.WeekOverWeek

	report.SlowRequest.Value = slowQueryRates.Value
	report.SlowRequest.DayOverDay = slowQueryRates.DayOverDay
	report.SlowRequest.WeekOverWeek = slowQueryRates.WeekOverWeek

	return report, nil
}
func SystemRealtimeReportv1_1(ctx context.Context) (*aliyun3.RealtimeReport, error) {
	report := &aliyun3.RealtimeReport{
		Availability: aliyun3.Metric{},
		TotalRequest: aliyun3.Metric{},
		ErrorRequest: aliyun3.Metric{},
		ProbeFailCnt: aliyun3.Metric{},
	}

	// 查最近7天的分数信息
	models, err := el.NewSystemScoreDao().FindTimespanScore(
		ctx,
		time.Now().AddDate(0, 0, -8),
		time.Now(),
	)
	if err != nil {
		hlog.CtxErrorf(ctx, "find timespan score failed, err: %v", err)
		return nil, err
	}
	sort.Slice(models, func(i, j int) bool {
		return !models[i].Date.Before(models[j].Date)
	})
	availabilityScores := map[string]int{}
	totalRequests := map[string]int{}
	errorRequests := map[string]int{}
	SlowRequests := map[string]int{}
	ProbeFailRequests := map[string]int{}

	dateToModel := map[string]el.SystemScoreModel{}
	for _, model := range models {
		date := model.Date.Format("2006-01-02")

		availabilityScores[date] = int(model.Score + 0.5)
		totalRequests[date] = int(model.TotalReq)
		errorRequests[date] = int(model.FailReq)
		SlowRequests[date] = int(model.SlowRate * 100)
		ProbeFailRequests[date] = int(model.ProbeFailReq)

		dateToModel[date] = model
	}
	scoreReventResults := utils.ComputeRevent(availabilityScores)
	totalRequestsReventResults := utils.ComputeRevent(totalRequests)
	errorRequestsReventResults := utils.ComputeRevent(errorRequests)
	slowRequestsReventResults := utils.ComputeRevent(SlowRequests)
	probeFailRequestsReventResults := utils.ComputeRevent(ProbeFailRequests)
	// 按日期降序
	sort.Slice(scoreReventResults, func(i, j int) bool {
		return scoreReventResults[i].Date > scoreReventResults[j].Date
	})
	sort.Slice(totalRequestsReventResults, func(i, j int) bool {
		return totalRequestsReventResults[i].Date > totalRequestsReventResults[j].Date
	})
	sort.Slice(errorRequestsReventResults, func(i, j int) bool {
		return errorRequestsReventResults[i].Date > errorRequestsReventResults[j].Date
	})
	sort.Slice(slowRequestsReventResults, func(i, j int) bool {
		return slowRequestsReventResults[i].Date > slowRequestsReventResults[j].Date
	})
	sort.Slice(probeFailRequestsReventResults, func(i, j int) bool {
		return probeFailRequestsReventResults[i].Date > probeFailRequestsReventResults[j].Date
	})

	report.Availability.Value = int(scoreReventResults[0].Score)
	report.Availability.DayOverDay = scoreReventResults[0].DayOverDay
	report.Availability.WeekOverWeek = scoreReventResults[0].WeekOverWeek

	report.TotalRequest.Value = int(totalRequestsReventResults[0].Score)
	report.TotalRequest.DayOverDay = totalRequestsReventResults[0].DayOverDay
	report.TotalRequest.WeekOverWeek = totalRequestsReventResults[0].WeekOverWeek

	report.ErrorRequest.Value = int(errorRequestsReventResults[0].Score)
	report.ErrorRequest.DayOverDay = errorRequestsReventResults[0].DayOverDay
	report.ErrorRequest.WeekOverWeek = errorRequestsReventResults[0].WeekOverWeek

	report.SlowRequest.Value = slowRequestsReventResults[0].Score / 100
	report.SlowRequest.DayOverDay = slowRequestsReventResults[0].DayOverDay
	report.SlowRequest.WeekOverWeek = slowRequestsReventResults[0].WeekOverWeek

	probeLogAnlz := aliyun3.RealtimeProbeLoganlz(ctx)
	report.ProbeFailCnt.Value = probeLogAnlz.Value
	report.ProbeFailCnt.DayOverDay = probeLogAnlz.DayOverDay
	report.ProbeFailCnt.WeekOverWeek = probeLogAnlz.WeekOverWeek

	//report.ProbeFailCnt.Value = int(probeFailRequestsReventResults[0].Score)
	//report.ProbeFailCnt.DayOverDay = probeFailRequestsReventResults[0].DayOverDay
	//report.ProbeFailCnt.WeekOverWeek = probeFailRequestsReventResults[0].WeekOverWeek
	//
	return report, nil
}
