package service

import (
	"context"
	"empyrean_lens/dal/mongo"
	"empyrean_lens/utils"
	"time"
)

func SystemScoreResult(ctx context.Context, timeBegin, timeEnd time.Time) ([]utils.ReventResult, error) {
	models, err := mongo.NewSystemScoreDao().FindTimespanScore(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	availabilityScores := map[string]int{}
	for _, model := range models {
		date := model.Date.Format("2006-01-02")
		availabilityScores[date] = int(model.Score + 0.5)
	}
	reventResults := utils.ComputeRevent(availabilityScores)
	return reventResults, nil
}
