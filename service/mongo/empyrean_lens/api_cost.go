package empyrean_lens

import (
	"context"
	el "empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/service/aliyun"
	"fmt"
	"sort"
	"time"
)

func ApiCostResult(ctx context.Context, timeBegin, timeEnd time.Time) ([]aliyun.CoreLogTimeSpanReportModel, error) {
	models, err := el.NewApicostModelDao().FindTimespanCost(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	reports := []aliyun.CoreLogTimeSpanReportModel{}
	for _, model := range models {
		date := model.Date.Format("2006-01-02")
		reports = append(reports, aliyun.CoreLogTimeSpanReportModel{
			Date: date,
			//Core:                    model.ApiName,
			Node: model.ApiName,
			//TotalCost:               model.to,
			NumReq: model.ReqCnt,
			//Costs:                   nil,
			AvgCost:                 model.AvgCost,
			CostDistribution0_1:     model.CostDistribution0_1,
			CostDistribution1_3:     model.CostDistribution1_3,
			CostDistribution3_5:     model.CostDistribution3_5,
			CostDistribution5_10:    model.CostDistribution5_10,
			CostDistribution10_20:   model.CostDistribution10_20,
			CostDistribution20_30:   model.CostDistribution20_30,
			CostDistribution30_50:   model.CostDistribution30_50,
			CostDistribution50_100:  model.CostDistribution50_100,
			CostDistribution100_inf: model.CostDistribution100_inf,
		})
	}
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Date == reports[j].Date {
			return reports[i].Node < reports[j].Node
		}
		return reports[i].Date > reports[j].Date
	})

	return reports, nil
}

func DailyPerformanceResult(ctx context.Context, timeBegin, timeEnd time.Time) error {
	apiCostModels, err := el.NewApicostModelDao().FindTimespanCost(ctx, timeBegin, timeEnd)
	if err != nil {
		return err
	}
	apiFailModels, err := el.NewApifailureModelDao().FindTimespanFailure(ctx, timeBegin, timeEnd)
	if err != nil {
		return err
	}
	for _, model := range apiFailModels {
		if model.ApiName == "【后端】问题推荐" {
			fmt.Println(model)
		}
	}
	fmt.Println("====================")
	for _, model := range apiCostModels {
		fmt.Println(model)
	}
	return nil
}
