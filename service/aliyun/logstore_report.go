package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/utils"
	"sort"
	"sync"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type CoreLogTimeSpanReportModel struct {
	Date                    string
	Core                    string
	Node                    string
	TotalCost               float64
	NumReq                  int64
	Costs                   []float64
	AvgCost                 float64
	CostDistribution0_1     float64
	CostDistribution1_3     float64
	CostDistribution3_5     float64
	CostDistribution5_10    float64
	CostDistribution10_20   float64
	CostDistribution20_30   float64
	CostDistribution30_50   float64
	CostDistribution50_100  float64
	CostDistribution100_inf float64
}

func LogStoreTimeSpanReport(ctx context.Context, timespan int) ([]CoreLogTimeSpanReportModel, error) {

	cores := []string{
		consts.CORE_NAME_OUTLINE,
		consts.CORE_NAME_ABSTRACT,
		consts.CORE_NAME_VIEWPOINT,
		"问答模型",
		"multi",
	}
	var mutex sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(cores))
	timespanReports := map[string]map[string]CoreLogTimeSpanReportModel{}
	nodes := []string{
		consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST,
		consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST,
		consts.ALIYUN_LOG_NODE_ABSTRACT_AI_COST,
		consts.ALIYUN_LOG_NODE_ABSTRACT_ETE_COST,
		consts.ALIYUN_LOG_NODE_QA_DONE_COST,
		consts.ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST,

		consts.ALIYUN_LOG_NODE_MULTI_ETE_COST,
		consts.ALIYUN_LOG_NODE_ANALYSIS_REPEATER,
		consts.ALIYUN_LOG_NODE_MERGE_REPEATER,
		consts.ALIYUN_LOG_NODE_SUMMARY_REPEATER,

		//consts.ALIYUN_LOG_NODE_ANALYSIS,
		//consts.ALIYUN_LOG_NODE_ANALYSIS_ALL,
		//consts.ALIYUN_LOG_NODE_MERGE,
		//consts.ALIYUN_LOG_NODE_MULTI_ALL_SUCCESS,
		//consts.ALIYUN_LOG_NODE_THEME_ALL_SUMMARY,
		//consts.ALIYUN_LOG_NODE_THEME_SUMMARY,
	}

	for i := 0; i < len(cores); i++ {
		go func(coreName string) {
			defer wg.Done()
			logs := []aliyun.CoreLog{}
			//if coreName == consts.CORE_NAME_ABSTRACT {
			//	fmt.Println(coreName)
			//}
			if timespan == consts.TIMESPAN_LONGTIME {
				_logs, err := aliyun.CoreReportLongTime(ctx, coreName)
				if err != nil {
					hlog.CtxErrorf(ctx, "CoreReportLongTime failed, core: %v err: %v", cores[i], err)
					return
				}
				logs = _logs
			} else if timespan == consts.TIMESPAN_WEEK {
				_logs, err := aliyun.CoreReportOneWeek(ctx, coreName)
				if err != nil {
					hlog.CtxErrorf(ctx, "CoreReportLongTime failed, core: %v err: %v", cores[i], err)
					return
				}
				logs = _logs
			} else if timespan == consts.TIMESPAN_TODAY {
				_logs, err := aliyun.CoreReportToday(ctx, coreName)
				if err != nil {
					hlog.CtxErrorf(ctx, "CoreReportLongTime failed, core: %v err: %v", cores[i], err)
					return
				}
				logs = _logs
			}
			if len(logs) == 0 {
				//hlog.CtxErrorf(ctx, "CoreReportLongTime failed, core: %v err: %v", cores[i], "no logs")
				return
			}
			mutex.Lock()
			var aiStart float64
			for _, log := range logs {
				if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_AI_START {
					aiStart = log.Cost
				}
				if !utils.Contains(nodes, log.Node) {
					continue
				}
				if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST {
					log.Cost = log.Cost - aiStart
				}
				day := log.Time.Format("2006-01-02")
				dayReports, ok := timespanReports[day]
				if !ok {
					dayReports = map[string]CoreLogTimeSpanReportModel{}
					timespanReports[day] = dayReports
				}
				//if log.Node == consts.ALIYUN_LOG_NODE_MULTI_ETE_COST {
				//	fmt.Println("======", log)
				//}

				report, ok := dayReports[log.Node]
				if !ok {
					report = CoreLogTimeSpanReportModel{
						Date:  log.Time.Format("2006-01-02"),
						Core:  log.CoreName,
						Node:  log.Node,
						Costs: []float64{},
					}
				}
				report.Costs = append(report.Costs, log.Cost)
				//if log.Node == consts.ALIYUN_LOG_NODE_ABSTRACT_ETE_COST && log.Time.Format("2006-01-02") == "2024-07-24" {
				//	fmt.Println("概述端到端======", log.Time.Day(), log.Node, len(report.Costs), log)
				//}
				report.NumReq += 1
				report.TotalCost += log.Cost
				if log.Cost < 1 {
					report.CostDistribution0_1 += 1
				} else if log.Cost < 3 {
					report.CostDistribution1_3 += 1
				} else if log.Cost < 5 {
					report.CostDistribution3_5 += 1
				} else if log.Cost < 10 {
					report.CostDistribution5_10 += 1
				} else if log.Cost < 20 {
					report.CostDistribution10_20 += 1
				} else if log.Cost < 30 {
					report.CostDistribution20_30 += 1
				} else if log.Cost < 50 {
					report.CostDistribution30_50 += 1
				} else if log.Cost < 100 {
					report.CostDistribution50_100 += 1
				} else {
					report.CostDistribution100_inf += 1
				}
				dayReports[log.Node] = report
				timespanReports[day] = dayReports

			}
			mutex.Unlock()
		}(cores[i])
	}
	wg.Wait()
	finalReports := []CoreLogTimeSpanReportModel{}
	for _, dayReports := range timespanReports {
		for _, report := range dayReports {
			//if report.Node == consts.ALIYUN_LOG_NODE_ABSTRACT_ETE_COST {
			//	fmt.Println("abstract:======", report)
			//}
			if alias, ok := consts.NODE_MAP[report.Node]; ok {
				report.Node = alias
			}
			//report.AvgCost = report.TotalCost / float64(len(report.Costs))
			report.AvgCost = utils.Avg(report.Costs)
			report.CostDistribution0_1 = report.CostDistribution0_1 / float64(len(report.Costs)) * 100
			report.CostDistribution1_3 = report.CostDistribution1_3 / float64(len(report.Costs)) * 100
			report.CostDistribution3_5 = report.CostDistribution3_5 / float64(len(report.Costs)) * 100
			report.CostDistribution5_10 = report.CostDistribution5_10 / float64(len(report.Costs)) * 100
			report.CostDistribution10_20 = report.CostDistribution10_20 / float64(len(report.Costs)) * 100
			report.CostDistribution20_30 = report.CostDistribution20_30 / float64(len(report.Costs)) * 100
			report.CostDistribution30_50 = report.CostDistribution30_50 / float64(len(report.Costs)) * 100
			report.CostDistribution50_100 = report.CostDistribution50_100 / float64(len(report.Costs)) * 100
			report.CostDistribution100_inf = report.CostDistribution100_inf / float64(len(report.Costs)) * 100
			finalReports = append(finalReports, report)
		}
	}

	// finalReports按照Date字段降续排序
	sort.Slice(finalReports, func(i, j int) bool {
		if finalReports[i].Date == finalReports[j].Date {
			return finalReports[i].Node < finalReports[j].Node
		}
		return finalReports[i].Date > finalReports[j].Date
	})
	return finalReports, nil
}
