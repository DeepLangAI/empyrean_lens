package service

import (
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/utils"
	"sync"
	"time"
)

func SystemAvailability(ctx context.Context) ([]utils.ReventResult, error) {
	systemFactors := map[string]utils.SystemStablityFactor{}
	availabilityScores := map[string]int{}
	monthLogs, err := NginxMonthReport(ctx)
	if err != nil {
		return nil, err
	}
	for _, log := range monthLogs {
		if log.CoreApiName != "当日总览" {
			continue
		}
		factor, exists := systemFactors[log.Date]
		if !exists {
			systemFactors[log.Date] = utils.SystemStablityFactor{}
		}
		factor.ApiFailRate = log.FailRate / 100.0
		systemFactors[log.Date] = factor
	}
	for date, factor := range systemFactors {
		availabilityScores[date] = utils.ComputeStablityScore(factor)
	}
	reventResults := utils.ComputeRevent(availabilityScores)
	return reventResults, nil
}
func realtimeNginxLogs(ctx context.Context) map[int][]aliyun.NginxLog {
	var wg sync.WaitGroup
	var mu sync.Mutex
	nginxLogs := map[int][]aliyun.NginxLog{}
	days := []int{0, 1, 7}
	for _, day := range days {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			anchorDay := time.Now().AddDate(0, 0, -daysLookback).Format("2006-01-02")

			logs := []aliyun.NginxLog{}
			businessLogs, err := aliyun.NginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				return
			}
			modelLogs, err := aliyun.ModelNginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				return
			}
			// 上报日志的时间，与阿里云将日志入库的时间有可能不同，会导致当天最后一段时间的日志可能落在了第二天内
			// 这里需要用真实的日志时间来调整
			for _, log := range businessLogs {
				if log.Time.Format("2006-01-02") == anchorDay {
					logs = append(logs, log)
				}
			}
			for _, log := range modelLogs {
				if log.Time.Format("2006-01-02") == anchorDay {
					logs = append(logs, log)
				}
			}
			//logs = append(logs, businessLogs...)
			//logs = append(logs, modelLogs...)
			mu.Lock()
			nginxLogs[daysLookback] = logs
			mu.Unlock()
		}(day)
	}
	wg.Wait()
	return nginxLogs
}

func deltaPercent(oldValue, newValue float64) float64 {
	if oldValue == 0 {
		return 0
	}
	return (newValue - oldValue) / oldValue * 100
}

type Metric struct {
	Value        int
	DayOverDay   float64
	WeekOverWeek float64
}

type RealtimeReport struct {
	Availability Metric
	TotalRequest Metric
	ErrorRequest Metric
}

func RealtimeAvailability(ctx context.Context) (RealtimeReport, error) {
	report := RealtimeReport{}
	systemFactors := map[int]utils.SystemStablityFactor{}
	nginxLogs := realtimeNginxLogs(ctx)
	totalRequests := map[int]int{}
	errorRequests := map[int]int{}
	for day, logs := range nginxLogs {
		totalRequests[day] = len(logs)

		errorRequests[day] = 0

		for _, log := range logs {
			if log.Status != "200" {
				errorRequests[day]++
			}
		}
	}

	report.TotalRequest.Value = totalRequests[0]
	report.TotalRequest.DayOverDay = deltaPercent(float64(totalRequests[1]), float64(totalRequests[0]))
	report.TotalRequest.WeekOverWeek = deltaPercent(float64(totalRequests[7]), float64(totalRequests[0]))

	report.ErrorRequest.Value = errorRequests[0]
	report.ErrorRequest.DayOverDay = deltaPercent(float64(errorRequests[1]), float64(errorRequests[0]))
	report.ErrorRequest.WeekOverWeek = deltaPercent(float64(errorRequests[7]), float64(errorRequests[0]))

	systemFactors[0] = utils.SystemStablityFactor{ApiFailRate: float64(errorRequests[0]) / float64(totalRequests[0])}
	systemFactors[1] = utils.SystemStablityFactor{ApiFailRate: float64(errorRequests[1]) / float64(totalRequests[1])}
	systemFactors[7] = utils.SystemStablityFactor{ApiFailRate: float64(errorRequests[7]) / float64(totalRequests[7])}
	report.Availability.Value = utils.ComputeStablityScore(systemFactors[0])
	report.Availability.DayOverDay = deltaPercent(float64(utils.ComputeStablityScore(systemFactors[1])), float64(utils.ComputeStablityScore(systemFactors[0])))
	report.Availability.WeekOverWeek = deltaPercent(float64(utils.ComputeStablityScore(systemFactors[7])), float64(utils.ComputeStablityScore(systemFactors[0])))

	return report, nil
}
