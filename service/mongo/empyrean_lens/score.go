package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
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
		})
	}
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
