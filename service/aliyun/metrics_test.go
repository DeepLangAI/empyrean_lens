package aliyun

import (
	context "context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"testing"
	"time"
)

func TestQueryMetrics2(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	//days := []int{0, 1, 2, 3, 4, 5, 6, 7}
	days := []int{0, 1}
	report := AigcCostMetricOfDays(ctx, days)
	for _, detail := range report {
		fmt.Println(detail.Date, detail.GenerateType, detail.ReqCount, detail.FailCount, detail.SlowQueryCount)
	}
}
func TestQueryMetrics1(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	start := time.Date(2024, 7, 26, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 7, 27, 0, 0, 0, 0, time.UTC)
	report, err := aigcCostMetric(ctx, start, end)
	if err != nil {
		t.Errorf("aigcCostMetric failed: %v", err)
	} else {
		for date, dateDetail := range report {
			fmt.Println(date)
			for genType, detail := range dateDetail {
				fmt.Println(genType, detail)
			}
		}
	}
}

func TestQueryMetrics(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	start := time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 7, 26, 0, 0, 0, 0, time.UTC)
	metrics, err := QueryMetrics(ctx, "lingo_backend_aigc_cost{}", start, end)
	//metrics, err := QueryMetrics(ctx, "lingo_backend_aigc_count{}", start, end)
	if err != nil {
		t.Errorf("QueryMetrics failed: %v", err)
	} else {
		for _, res := range metrics.Data.Result {
			metric := res.Metric
			values := res.Values
			fmt.Println(metric, len(values))
			//for _, v := range values {
			//	timestamp := int64(v[0].(float64))
			//	tick := time.Unix(timestamp, 0)
			//	cost, err := strconv.ParseFloat(v[1].(string), 64)
			//	if err != nil {
			//		t.Errorf("ParseFloat failed: %v", err)
			//	}
			//	fmt.Println(tick, cost)
			//}
		}
	}
}
