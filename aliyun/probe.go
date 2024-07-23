package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo"
	"empyrean_lens/utils"
	"errors"
	"fmt"
	"sync"
	"time"
)

type ProbeAnlz struct {
	TotalNodes   Metric
	SuccessNodes Metric
	FailNodes    Metric
}

func ProbeTimespanFailRate(ctx context.Context, timespan int) (map[string]float64, error) {
	now := time.Now()
	if timespan == consts.TIMESPAN_LONGTIME {
		from := time.Date(2024, 7, 0, 0, 0, 0, 0, now.Location())
		return ProbeErrorRate(ctx, from, now)
	} else if timespan == consts.TIMESPAN_WEEK {
		fromDay := now.AddDate(0, 0, -7)
		from := time.Date(fromDay.Year(), fromDay.Month(), fromDay.Day(), 0, 0, 0, 0, fromDay.Location())
		return ProbeErrorRate(ctx, from, now)
	} else if timespan == consts.TIMESPAN_TODAY {
		fromDay := now.AddDate(0, 0, 0)
		from := time.Date(fromDay.Year(), fromDay.Month(), fromDay.Day(), 0, 0, 0, 0, fromDay.Location())
		return ProbeErrorRate(ctx, from, now)
	}
	return nil, errors.New("timespan not support")
}

func ProbeErrorRate(ctx context.Context, timeBegin, timeEnd time.Time) (map[string]float64, error) {
	probeAnlz := map[string]ProbeAnlz{}
	logs, err := mongo.NewProbeLogModelDao().FindTimespanProbeLog(
		ctx,
		timeBegin,
		timeEnd,
	)
	if err != nil {
		return nil, err
	}
	for _, log := range logs {
		date := log.CreateTime.Format("2006-01-02")
		if anlz, ok := probeAnlz[date]; ok {
			anlz.TotalNodes.Value += log.TotalNodes
			anlz.SuccessNodes.Value += log.SuccessNodes
			anlz.FailNodes.Value += log.TotalNodes - log.SuccessNodes
		} else {
			probeAnlz[date] = ProbeAnlz{TotalNodes: Metric{Value: log.TotalNodes}, SuccessNodes: Metric{Value: log.SuccessNodes}, FailNodes: Metric{Value: log.TotalNodes - log.SuccessNodes}}
		}
	}
	probeErrorRate := map[string]float64{}
	for date, anlz := range probeAnlz {
		probeErrorRate[date] = float64(anlz.FailNodes.Value) / float64(anlz.TotalNodes.Value) * 100
	}
	return probeErrorRate, nil
}

func RealtimeProbeLoganlz(ctx context.Context) (ProbeAnlz, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	probeLogs := map[int][]mongo.ProbeLogModel{}
	probeAnlz := map[int]ProbeAnlz{}

	days := []int{0, 1, 7}
	for _, day := range days {
		wg.Add(1)

		go func(daysLookback int) {
			defer wg.Done()
			anchorDay := time.Now().AddDate(0, 0, -daysLookback)
			if daysLookback == 1 {
				fmt.Println(daysLookback)
			}

			logs, err := mongo.NewProbeLogModelDao().FindTimespanProbeLog(
				ctx,
				time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, anchorDay.Location()),
				time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 23, 59, 59, 999999999, anchorDay.Location()),
			)
			if err != nil {
				return
			}
			mu.Lock()
			probeLogs[daysLookback] = logs
			totalNodes := 0
			successNodes := 0
			for _, log := range logs {
				totalNodes += log.TotalNodes
				successNodes += log.SuccessNodes
			}
			failNodes := totalNodes - successNodes
			probeAnlz[daysLookback] = ProbeAnlz{TotalNodes: Metric{Value: totalNodes}, SuccessNodes: Metric{Value: successNodes}, FailNodes: Metric{Value: failNodes}}
			mu.Unlock()
		}(day)
	}
	wg.Wait()
	if anlz, ok := probeAnlz[0]; ok {
		anlz.FailNodes.DayOverDay = utils.DeltaPercent(float64(probeAnlz[1].FailNodes.Value), float64(probeAnlz[0].FailNodes.Value))
		anlz.FailNodes.WeekOverWeek = utils.DeltaPercent(float64(probeAnlz[7].FailNodes.Value), float64(probeAnlz[0].FailNodes.Value))

		anlz.TotalNodes.DayOverDay = utils.DeltaPercent(float64(probeAnlz[1].TotalNodes.Value), float64(probeAnlz[0].TotalNodes.Value))
		anlz.TotalNodes.WeekOverWeek = utils.DeltaPercent(float64(probeAnlz[7].TotalNodes.Value), float64(probeAnlz[0].TotalNodes.Value))

		anlz.SuccessNodes.DayOverDay = utils.DeltaPercent(float64(probeAnlz[1].SuccessNodes.Value), float64(probeAnlz[0].SuccessNodes.Value))
		anlz.SuccessNodes.WeekOverWeek = utils.DeltaPercent(float64(probeAnlz[7].SuccessNodes.Value), float64(probeAnlz[0].SuccessNodes.Value))
	}
	return probeAnlz[0], nil
}
