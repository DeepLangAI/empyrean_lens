package mongo

import (
	"context"
	"empyrean_lens/dal/mongo"
	"fmt"
	"sort"
	"time"
)

func SceneResult(ctx context.Context, timeBegin, timeEnd time.Time) ([]map[string]string, error) {
	models, err := mongo.NewSceneModelDao().FindTimespanScene(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}

	sort.Slice(models, func(i, j int) bool {
		datei := models[i].Date.Format("2006-01-02")
		datej := models[j].Date.Format("2006-01-02")
		if datei == datej {
			return models[i].Scene > models[j].Scene
		}
		return datei > datej
	})
	results := []map[string]string{}
	for _, model := range models {
		failRate := 0.0
		slowRate := 0.0
		if model.TotalCnt > 0 {
			failRate = float64(model.FailCnt) / float64(model.TotalCnt) * 100
			slowRate = float64(model.SlowCnt) / float64(model.TotalCnt) * 100
		}
		failReason := "-"
		if len(model.FailReason) > 0 {
			failReason = model.FailReason
		}
		result := map[string]string{
			"Date":       model.Date.Format("2006-01-02"),
			"Scene":      model.Scene,
			"TotalCnt":   fmt.Sprintf("%v", model.TotalCnt),
			"FailCnt":    fmt.Sprintf("%v", model.FailCnt),
			"FailReason": fmt.Sprintf("%v", failReason),
			"SlowCnt":    fmt.Sprintf("%v", model.SlowCnt),
			"FailRate":   fmt.Sprintf("%.2f", failRate),
			"SlowRate":   fmt.Sprintf("%.2f", slowRate),
		}
		results = append(results, result)
	}
	return results, nil
}
