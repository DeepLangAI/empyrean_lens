package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"time"
)

func ProbeTimespanFailRate(ctx context.Context, timespan int) (map[string]float64, error) {
	timeBegin := time.Now()
	timeEnd := time.Date(timeBegin.Year(), timeBegin.Month(), timeBegin.Day(), 23, 59, 59, 0, timeBegin.Location())

	if timespan == consts.TIMESPAN_TODAY {
		anchorDay := time.Now()
		timeBegin = time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, anchorDay.Location())
	} else if timespan == consts.TIMESPAN_WEEK {
		anchorDay := time.Now().AddDate(0, 0, -7)
		timeBegin = time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, anchorDay.Location())
	} else if timespan == consts.TIMESPAN_LONGTIME {
		timeBegin = time.Date(2024, 7, 1, 0, 0, 0, 0, timeBegin.Location())
	}
	models, err := empyrean_lens.NewApiProbeLogModelDao().FindTimespanApiProbeLog(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	failReq := map[string]int32{}
	totalReq := map[string]int32{}
	failRate := map[string]float64{}
	for _, model := range models {
		date := model.CreateTime.Format("2006-01-02")
		totalReq[date] += 1
		if !model.Correct {
			failReq[date] += 1
		}
	}
	for date, failCnt := range failReq {
		failRate[date] = float64(failCnt) / float64(totalReq[date]) * 100
	}
	return failRate, nil
}

func RealtimeProbeLoganlz(ctx context.Context) *Metric {

	timeBegin := time.Now()
	timeEnd := time.Date(timeBegin.Year(), timeBegin.Month(), timeBegin.Day(), 23, 59, 59, 0, timeBegin.Location())
	anchorDay := time.Now().AddDate(0, 0, -7)
	timeBegin = time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, anchorDay.Location())
	models, err := empyrean_lens.NewApiProbeLogModelDao().FindTimespanApiProbeLog(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil
	}
	probeFailCnts_0 := 0
	probeFailCnts_1 := 0
	probeFailCnts_7 := 0
	failCnts := map[string]int{}
	probeMetric := &Metric{
		Value:        0,
		DayOverDay:   0,
		WeekOverWeek: 0,
	}

	for _, model := range models {
		date := model.CreateTime.Add(8 * time.Hour).Format("2006-01-02")
		if !model.Correct {
			failCnts[date] += 1
		}
	}
	probeFailCnts_0 = failCnts[time.Now().AddDate(0, 0, 0).Format("2006-01-02")]
	probeFailCnts_1 = failCnts[time.Now().AddDate(0, 0, -1).Format("2006-01-02")]
	probeFailCnts_7 = failCnts[time.Now().AddDate(0, 0, -7).Format("2006-01-02")]
	probeMetric.Value = probeFailCnts_0
	probeMetric.DayOverDay = utils.DeltaPercent(float64(probeFailCnts_1), float64(probeFailCnts_0))
	probeMetric.WeekOverWeek = utils.DeltaPercent(float64(probeFailCnts_7), float64(probeFailCnts_0))
	return probeMetric
}
