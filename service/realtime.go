package service

import (
	"context"
	"empyrean_lens/aliyun"
	aliyun2 "empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo"
	"empyrean_lens/utils"
	"time"
)

func apiFailureRate(ctx context.Context, beginTime, endTime time.Time) (int, int) {
	failureModels, err := mongo.NewApifailureModelDao().FindTimespanFailure(ctx, beginTime, endTime)
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

func SystemRealtimeReport(ctx context.Context) (*aliyun.RealtimeReport, error) {
	report := &aliyun.RealtimeReport{
		Availability: aliyun.Metric{},
		TotalRequest: aliyun.Metric{},
		ErrorRequest: aliyun.Metric{},
		ProbeFailCnt: aliyun.Metric{},
	}

	now := time.Now()
	beginTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	failureModels, err := mongo.NewApifailureModelDao().FindTimespanFailure(ctx, beginTime, now)
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
		SlowQueryRate: 1,
		ProbeFailRate: 1,
	}
	score_0 := utils.ComputeStablityScore(systemScoreFactor)
	scoreModel_1, err := mongo.NewSystemScoreDao().FindScoreByTime(ctx, beginTime.AddDate(0, 0, -1))
	score_1 := 0.0
	if err == nil {
		score_1 = scoreModel_1.Score
	}

	scoreModel_7, err := mongo.NewSystemScoreDao().FindScoreByTime(ctx, beginTime.AddDate(0, 0, -7))
	score_7 := 0.0
	if err == nil {
		score_7 = scoreModel_7.Score
	}
	report.Availability.Value = score_0
	report.Availability.DayOverDay = utils.DeltaPercent(score_1, float64(score_0))
	report.Availability.WeekOverWeek = utils.DeltaPercent(score_7, float64(score_0))

	probeLogAnlz, err := aliyun.RealtimeProbeLoganlz(ctx)
	if err != nil {
		return nil, err
	}
	report.ProbeFailCnt.Value = probeLogAnlz.FailNodes.Value
	report.ProbeFailCnt.DayOverDay = probeLogAnlz.FailNodes.DayOverDay
	report.ProbeFailCnt.WeekOverWeek = probeLogAnlz.FailNodes.WeekOverWeek

	return report, nil
}
