package mongo

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/dal/mongo"
	"empyrean_lens/utils"
	"fmt"
	"sort"
	"time"
)

func SaveBatch(ctx context.Context, data empyrean_lens.WriteProbeReq) error {
	dao := mongo.NewApiProbeLogModelDao()
	for _, d := range data.Data {
		model := mongo.ApiProbeLogModel{
			Scene:      d.Scene,
			Api:        d.API,
			Host:       d.Host,
			IsCore:     d.IsCore,
			Success:    d.Success,
			Correct:    d.Correct,
			Cost:       d.Cost,
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.Save(ctx, model); err != nil {
			return err
		}
	}
	return nil
}

type ProbeReportModel struct {
	Date      string
	Scene     string
	API       string
	Host      string
	Costs     []float64
	Corrects  []float64
	Successes []float64
}

func ProbeReport(ctx context.Context) ([]map[string]string, error) {
	dao := mongo.NewApiProbeLogModelDao()
	timeBegin := time.Date(2024, 7, 1, 0, 0, 0, 0, time.Local)
	timeEnd := time.Now()
	logs, err := dao.FindTimespanApiProbeLog(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	cache := map[string]ProbeReportModel{}
	for _, log := range logs {
		key := fmt.Sprintf("%v %v%v,%v", log.CreateTime.Format("2006-01-02"), log.Host, log.Api, log.Scene)
		cacheVal, ok := cache[key]
		fmt.Println("key: ", key, "ok:", ok)
		if !ok {
			cacheVal = ProbeReportModel{
				Date:  log.CreateTime.Format("2006-01-02"),
				Scene: log.Scene,
				API:   log.Api,
				Host:  log.Host,
			}
		}
		cacheVal.Costs = append(cacheVal.Costs, log.Cost)
		if log.Correct {
			cacheVal.Corrects = append(cacheVal.Corrects, 1.0)
		} else {
			cacheVal.Corrects = append(cacheVal.Corrects, 0.0)
		}
		if log.Success {
			cacheVal.Successes = append(cacheVal.Successes, 1.0)
		} else {
			cacheVal.Successes = append(cacheVal.Successes, 0.0)
		}
		cache[key] = cacheVal
	}
	results := []map[string]string{}
	for _, val := range cache {
		results = append(results, map[string]string{
			"Date":        val.Date,
			"Scene":       val.Scene,
			"Api":         val.API,
			"Host":        val.Host,
			"AvgCost":     fmt.Sprintf("%.2f", utils.AvgSimple(val.Costs)),
			"CorrectRate": fmt.Sprintf("%.2f", utils.AvgSimple(val.Corrects)*100),
			"SuccessRate": fmt.Sprintf("%.2f", utils.AvgSimple(val.Successes)*100),
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i]["Date"] == results[j]["Date"] {
			return results[i]["Scene"] < results[j]["Scene"]
		}
		return results[i]["Date"] > results[j]["Date"]
	})
	return results, err
}
