package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"
)

type AigcMetricReport struct {
	Date         string
	GenerateType string
	Costs        []float64
	ReqCount     int64

	FailCount      int64
	FailRate       float64
	SlowQueryCount int64
	SlowQueryRate  float64
}

const GENERATE_TYPE_ABSTRACT = "0"
const GENERATE_TYPE_OUTLINE = "1"
const GENERATE_TYPE_VIEWPOINT = "3"

type MetricData struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Name         string `json:"__name__"`
				GenerateType string `json:"generate_type"`
			} `json:"metric"`
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func QueryMetrics(ctx context.Context, query string, timeBegin, timeEnd time.Time) (*MetricData, error) {
	uri := fmt.Sprintf("https://%v.%v/prometheus/%v/%v/api/v1/query_range",
		consts.PROJECT_NAME,
		consts.ENDPOINT,
		consts.PROJECT_NAME,
		consts.METRIC_SOTRE_NAME,
	)
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%v", int(timeBegin.Unix())))
	params.Set("end", fmt.Sprintf("%v", int(timeEnd.Unix())))
	params.Set("step", "1m")
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	metricData := &MetricData{}
	if err := utils.DoGetWithAuth(ctx, uri, params, headers, &metricData, consts.ALIYUN_SLS_API_KEY, consts.ALIYUN_SLS_API_SECRET); err != nil {
		return nil, err
	}
	if metricData.Status != "success" {
		return nil, fmt.Errorf("query metrics failed: %v", metricData.Status)
	}
	return metricData, nil
}

func aigcCostAnlz(ctx context.Context, report AigcMetricReport) AigcMetricReport {
	//fmt.Printf("Date: %s, GenerateType: %s, Costs: %v, ReqCount: %d\n", report.Date, report.GenerateType, report.Costs, report.ReqCount)
	report.FailCount = report.ReqCount - int64(len(report.Costs))
	if report.FailCount <= 0 {
		report.FailCount = 0
	}
	report.FailRate = float64(report.FailCount) / float64(report.ReqCount) * 100

	slowQueryThreshold := 10
	if report.GenerateType == GENERATE_TYPE_ABSTRACT {
		slowQueryThreshold = consts.SLOWQUERY_THRESHOLD_ABSTRACT
	} else if report.GenerateType == GENERATE_TYPE_OUTLINE {
		slowQueryThreshold = consts.SLOWQUERY_THRESHOLD_OUTLINE
	} else if report.GenerateType == GENERATE_TYPE_VIEWPOINT {
		slowQueryThreshold = consts.SLOWQUERY_THRESHOLD_VIEWPOINT
	}
	for _, cost := range report.Costs {
		if cost > float64(slowQueryThreshold) {
			report.SlowQueryCount++
		}
	}
	report.SlowQueryRate = float64(report.SlowQueryCount) / float64(report.ReqCount) * 100
	return report
}

func aigcCostMetric(ctx context.Context, timeBegin, timeEnd time.Time) (map[string]map[string]AigcMetricReport, error) {
	// date, generate_type, report
	finalReport := map[string]map[string]AigcMetricReport{}
	reqCount := map[string]map[string]int{}
	metrics, err := QueryMetrics(ctx, "lingo_backend_aigc_cost{}", timeBegin, timeEnd)
	if err != nil {
		hlog.CtxErrorf(ctx, "QueryMetrics failed: %v", err)
		return nil, err
	} else {
		for _, res := range metrics.Data.Result {
			metric := res.Metric
			values := res.Values
			for _, v := range values {
				tick := time.Unix(int64(v[0].(float64)), 0)
				timeStr := tick.Format("2006-01-02")
				//if timeStr != timeBegin.Format("2006-01-02") {
				//	continue
				//}
				cost, err := strconv.ParseFloat(v[1].(string), 64)
				if err != nil {
					hlog.CtxErrorf(ctx, "ParseFloat failed: %v", err)
					return nil, err
				}
				if _, ok := finalReport[timeStr]; !ok {
					finalReport[timeStr] = map[string]AigcMetricReport{}
				}
				if _, ok := finalReport[timeStr][metric.GenerateType]; !ok {
					finalReport[timeStr][metric.GenerateType] = AigcMetricReport{
						Date:         timeStr,
						GenerateType: metric.GenerateType,
						Costs:        []float64{},
						ReqCount:     0,
					}
				}
				report := finalReport[timeStr][metric.GenerateType]
				report.Costs = append(report.Costs, cost)
				finalReport[timeStr][metric.GenerateType] = report
			}
		}
	}

	metrics, err = QueryMetrics(ctx, "lingo_backend_aigc_count{}", timeBegin, timeEnd)
	if err != nil {
		hlog.CtxErrorf(ctx, "QueryMetrics failed: %v", err)
		return nil, err
	} else {
		for _, res := range metrics.Data.Result {
			metric := res.Metric
			values := res.Values
			for _, v := range values {
				tick := time.Unix(int64(v[0].(float64)), 0)
				timeStr := tick.Format("2006-01-02")
				count, err := strconv.ParseInt(v[1].(string), 10, 64)
				if err != nil {
					hlog.CtxErrorf(ctx, "ParseFloat failed: %v", err)
					return nil, err
				}
				if _, ok := reqCount[timeStr]; !ok {
					reqCount[timeStr] = map[string]int{}
				}
				if _, ok := reqCount[timeStr][metric.GenerateType]; !ok {
					reqCount[timeStr][metric.GenerateType] = 0
				} else {
					reqCount[timeStr][metric.GenerateType] += int(count)
				}
			}
		}
	}

	//reports := []AigcMetricReport{}
	result := map[string]map[string]AigcMetricReport{}
	for _, dateDetail := range finalReport {
		for _, detail := range dateDetail {
			if cnt, ok := reqCount[detail.Date][detail.GenerateType]; ok {
				detail.ReqCount = int64(cnt)
			}
			//detail = aigcCostAnlz(ctx, detail)
			if _, ok := result[detail.Date]; !ok {
				result[detail.Date] = map[string]AigcMetricReport{}
			}
			result[detail.Date][detail.GenerateType] = detail
			//finalReport[detail.Date][detail.GenerateType] = detail
			//reports = append(reports, detail)
		}
	}
	//reports = aigcCostAnlz(ctx, reports)
	return result, nil
}

func AigcCostMetricOfDays(ctx context.Context, days []int) []AigcMetricReport {
	var mutex sync.Mutex
	wg := sync.WaitGroup{}
	//date gentype
	finalReport := map[string]map[string]AigcMetricReport{}
	reports := []AigcMetricReport{}
	for _, daysLookback := range days {
		wg.Add(1)
		go func(day int) {
			defer wg.Done()
			anchorDay := time.Now().AddDate(0, 0, -day)
			start := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.UTC)
			end := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 23, 59, 59, 0, time.UTC)

			dayReport, err := aigcCostMetric(ctx, start, end)
			if err != nil {
				hlog.CtxErrorf(ctx, "aigcCostMetric failed: %v", err)
				return
			}
			mutex.Lock()

			for date, dateDetail := range dayReport {
				for genType, detail := range dateDetail {
					if _, ok := finalReport[date]; !ok {
						finalReport[date] = map[string]AigcMetricReport{}
					}
					dateReport := finalReport[date]
					if _, ok := dateReport[genType]; !ok {
						finalReport[date][genType] = AigcMetricReport{}
					}
					report := dateReport[genType]
					report.Date = detail.Date
					report.GenerateType = detail.GenerateType
					report.Costs = append(report.Costs, detail.Costs...)
					report.ReqCount += detail.ReqCount

					report = aigcCostAnlz(ctx, report)

					dateReport[genType] = report
					finalReport[date] = dateReport
				}
			}
			mutex.Unlock()
		}(daysLookback)
	}
	wg.Wait()
	//fmt.Println(finalReport)
	for _, dateDetail := range finalReport {
		for _, detail := range dateDetail {
			reports = append(reports, detail)
		}
	}

	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Date == reports[j].Date {
			return reports[i].GenerateType < reports[j].GenerateType
		}
		return reports[i].Date > reports[j].Date
	})
	return reports
}
