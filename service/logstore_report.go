package service

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"sort"
	"sync"
)

type CoreLogMonthReportModel struct {
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

func LogStoreMonthReport(ctx context.Context) ([]CoreLogMonthReportModel, error) {

	cores := []string{
		consts.CORE_NAME_OUTLINE,
		consts.CORE_NAME_ABSTRACT,
		consts.CORE_NAME_VIEWPOINT,
	}
	var mutex sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(cores))
	monthReports := map[int]map[string]CoreLogMonthReportModel{}
	nodes := []string{
		consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST,
		consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST,
		consts.ALIYUN_LOG_NODE_ABSTRACT_AI_COST,
		consts.ALIYUN_LOG_NODE_ABSTRACT_ETE_COST,
		consts.ALIYUN_LOG_NODE_QA_DONE_COST,
		consts.ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST,
	}

	for i := 0; i < len(cores); i++ {
		go func(coreName string) {
			defer wg.Done()
			logs, err := aliyun.CoreReportThisMonth(ctx, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "CoreReportThisMonth failed, core: %v err: %v", cores[i], err)
				return
			}
			mutex.Lock()
			//var aiStart float64
			for _, log := range logs {
				//if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_AI_START {
				//	aiStart = log.Cost
				//}
				if !utils.Contains(nodes, log.Node) {
					continue
				}
				//if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST {
				//	log.Cost = log.Cost
				//}
				day := log.Time.Day()
				dayReports, ok := monthReports[day]
				if !ok {
					dayReports = map[string]CoreLogMonthReportModel{}
					monthReports[day] = dayReports
				}

				report, ok := dayReports[log.Node]
				if !ok {
					report = CoreLogMonthReportModel{
						Date:  log.Time.Format("2006-01-02"),
						Core:  log.CoreName,
						Node:  log.Node,
						Costs: []float64{},
					}
				}
				report.Costs = append(report.Costs, log.Cost)
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

			}
			mutex.Unlock()
		}(cores[i])
	}
	wg.Wait()
	finalReports := []CoreLogMonthReportModel{}
	for _, dayReports := range monthReports {
		for _, report := range dayReports {
			if alias, ok := consts.NODE_MAP[report.Node]; ok {
				report.Node = alias
			}
			report.AvgCost = report.TotalCost / float64(len(report.Costs))
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
