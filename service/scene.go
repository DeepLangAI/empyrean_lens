package service

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
		result := map[string]string{
			"Date":     model.Date.Format("2006-01-02"),
			"Scene":    model.Scene,
			"TotalCnt": fmt.Sprintf("%v", model.TotalCnt),
			"FailCnt":  fmt.Sprintf("%v", model.FailCnt),
			"SlowCnt":  fmt.Sprintf("%v", model.SlowCnt),
			"FailRate": fmt.Sprintf("%.2f", float64(model.FailCnt)/float64(model.TotalCnt)*100),
			"SlowRate": fmt.Sprintf("%.2f", float64(model.SlowCnt)/float64(model.TotalCnt)*100),
		}
		results = append(results, result)
	}
	return results, nil
}
