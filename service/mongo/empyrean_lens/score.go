package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"math"
	"sort"
	"time"
)

func ScoreRelatedDetailQuery(ctx context.Context, req empyrean_lens2.DailyScoreReq) ([]*empyrean_lens2.DailyScoreRespData, error) {
	t1, e := time.Parse("2006-01-02", req.StartTime)
	if e != nil {
		hlog.CtxErrorf(ctx, "time parse err:%v", e)
		return nil, e
	}
	t2, e := time.Parse("2006-01-02", req.EndTime)
	if e != nil {
		hlog.CtxErrorf(ctx, "time parse err:%v", e)
		return nil, e
	}
	scoreDetails, e := FindScoreDetails(ctx, req.StartTime, req.EndTime)
	if e != nil {
		hlog.CtxErrorf(ctx, "FindScoreDetails err:%v", e)
		return nil, e
	}
	tracebackDetail, e := empyrean_lens.NewTracebackLogModelDao().FindTimespanTracebackLog(ctx, t1, t2)
	if e != nil {
		hlog.CtxErrorf(ctx, "FindTimespanTracebackLog err:%v", e)
		return nil, e
	}

	numTb := map[string]int{}
	for _, tb := range tracebackDetail {
		date := tb.Time.Format("2006-01-02")
		numTb[date] += 1
	}

	data := []*empyrean_lens2.DailyScoreRespData{}
	for _, d := range scoreDetails {
		date := d.Date.Format("2006-01-02")
		data = append(data, &empyrean_lens2.DailyScoreRespData{
			Date:          date,
			Score:         int32(d.Score),
			FailRate:      d.FailRate,
			SlowRate:      d.SlowRate,
			ProbeFailRate: d.ProbeFailRate,
			NumTraceback:  int32(numTb[date]),
			DayOverDay:    d.ScoreDayOverDay,
			WeekOverWeek:  d.ScoreWeekOverWeek,
			TotalReq:      d.TotalReq,
		})
	}
	hlog.CtxInfof(ctx, "系统分数：%+v", data)
	return data, nil
}

func FindScoreDetails(ctx context.Context, timeBegin, timeEnd string) ([]empyrean_lens.SystemScoreModel, error) {
	t1, e := time.Parse("2006-01-02", timeBegin)
	if e != nil {
		return nil, e
	}
	t2, e := time.Parse("2006-01-02", timeEnd)
	if e != nil {
		return nil, e
	}
	dao := empyrean_lens.NewSystemScoreDao()
	score, e := dao.FindTimespanScore(ctx, t1, t2)
	return score, nil
}

type ScoreListItem struct {
	utils.ReventResult
	TotalReq int32
}

func SystemScoreResult(ctx context.Context, timeBegin, timeEnd time.Time) ([]ScoreListItem, error) {
	models, err := empyrean_lens.NewSystemScoreDao().FindTimespanScore(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	dateToModel := map[string]empyrean_lens.SystemScoreModel{}
	availabilityScores := map[string]int{}
	for _, model := range models {
		date := model.Date.Format("2006-01-02")
		availabilityScores[date] = int(model.Score + 0.5)
		dateToModel[date] = model
	}
	results := []ScoreListItem{}
	reventResults := utils.ComputeRevent(availabilityScores)
	for _, reventResult := range reventResults {
		results = append(results, ScoreListItem{
			ReventResult: reventResult,
			TotalReq:     dateToModel[reventResult.Date].TotalReq,
		})
	}
	return results, nil
}

func UpdateLatestScoreInfo(ctx context.Context) error {
	// 更新最新一天的分数同比、环比等
	// 查最近7天的分数信息
	models, err := empyrean_lens.NewSystemScoreDao().FindTimespanScore(
		ctx,
		time.Now().AddDate(0, 0, -8),
		time.Now().Add(8*time.Hour),
	)
	if err != nil {
		hlog.CtxErrorf(ctx, "UpdateLatestScoreInfo err:%v", err)
		return err
	}
	sort.Slice(models, func(i, j int) bool {
		return !models[i].Date.Before(models[j].Date)
	})
	availabilityScores := map[string]int{}
	dateToModel := map[string]empyrean_lens.SystemScoreModel{}
	for _, model := range models {
		date := model.Date.Format("2006-01-02")
		availabilityScores[date] = int(model.Score + 0.5)
		dateToModel[date] = model
	}
	reventResults := utils.ComputeRevent(availabilityScores)
	// 按日期降序
	sort.Slice(reventResults, func(i, j int) bool {
		return reventResults[i].Date > reventResults[j].Date
	})
	reventResult := reventResults[0]
	models[0].ScoreDayOverDay = reventResult.DayOverDay
	models[0].ScoreWeekOverWeek = reventResult.WeekOverWeek
	// 查当日探针数据
	now := time.Now()
	probeLogs, err := empyrean_lens.NewApiProbeLogModelDao().FindTimespanApiProbeLog(
		ctx,
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		time.Now().Add(8*time.Hour),
	)
	if err != nil {
		hlog.CtxErrorf(ctx, "Find Probe logs err:%v", err)
		return err
	}
	models[0].ProbeTotalReq = 0
	models[0].ProbeFailReq = 0
	for _, log := range probeLogs {
		models[0].ProbeTotalReq += 1
		if !log.Correct {
			models[0].ProbeFailReq += 1
		}
	}
	models[0].ProbeFailRate = float64(models[0].ProbeFailReq) / float64(models[0].ProbeTotalReq) * 100
	if math.IsNaN(models[0].ProbeFailRate) {
		models[0].ProbeFailRate = 0
	}

	err = empyrean_lens.NewSystemScoreDao().CreateOrUpdate(ctx, models[0].Date, models[0])
	if err != nil {
		hlog.CtxErrorf(ctx, "UpdateLatestScoreInfo err:%v", err)
		return err
	}

	latestModel := models[0]
	err = empyrean_lens.NewScoreBackupDao().CreateOrUpdate(ctx, empyrean_lens.ScoreBackupModel{
		Time:              time.Now(),
		Score:             latestModel.Score,
		ScoreDayOverDay:   latestModel.ScoreDayOverDay,
		ScoreWeekOverWeek: latestModel.ScoreWeekOverWeek,
		TotalReq:          latestModel.TotalReq,
		FailReq:           latestModel.FailReq,
		SceneTotalReq:     latestModel.SceneTotalReq,
		SceneSlowReq:      latestModel.SceneSlowReq,
		FailRate:          latestModel.FailRate,
		SlowRate:          latestModel.SlowRate,
		ProbeFailReq:      latestModel.ProbeFailReq,
		ProbeTotalReq:     latestModel.ProbeTotalReq,
		ProbeFailRate:     latestModel.ProbeFailRate,
		AvgRespCost:       latestModel.AvgRespCost,
	})
	if err != nil {
		hlog.CtxErrorf(ctx, "UpdateLatestScoreBackupInfo err:%v", err)
		return err
	}
	return nil
}
