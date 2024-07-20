package service

import (
	"context"
	"empyrean_lens/utils"
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
