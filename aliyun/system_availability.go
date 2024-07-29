package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo"
	"empyrean_lens/utils"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func SystemTimespanAvailability(ctx context.Context, timespan int) ([]utils.ReventResult, error) {
	systemFactors := map[string]utils.SystemStablityFactor{}
	availabilityScores := map[string]int{}
	nginxLogs := []NginxTimeSpanReportModel{}
	//probeFailRates := map[string]float64{}

	nginxLogs, err := NginxTimespanReport(ctx, timespan)
	if err != nil {
		return nil, err
	}

	//_probeFailRates, err := ProbeTimespanFailRate(ctx, timespan)
	//if err != nil {
	//	return nil, err
	//}
	//probeFailRates = _probeFailRates

	for _, log := range nginxLogs {
		if log.CoreApiName != "当日总览" {
			continue
		}
		factor, exists := systemFactors[log.Date]
		if !exists {
			systemFactors[log.Date] = utils.SystemStablityFactor{}
		}
		factor.ApiFailRate = log.FailRate / 100.0
		//factor.ProbeFailRate = probeFailRates[log.Date]
		factor.ProbeFailRate = 1
		factor.SlowQueryRate = 1
		systemFactors[log.Date] = factor
	}
	for date, factor := range systemFactors {
		availabilityScores[date] = utils.ComputeStablityScore(factor)
	}
	reventResults := utils.ComputeRevent(availabilityScores)
	return reventResults, nil
}

func realtimeProbeLogs(ctx context.Context) []mongo.ProbeLogModel {
	now := time.Now()
	logs, err := mongo.NewProbeLogModelDao().FindTimespanProbeLog(
		ctx,
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()),
		time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location()),
	)
	if err != nil {
		hlog.CtxErrorf(ctx, "failed to get probe logs: %v", err)
		return nil
	}
	return logs
}

func NginxLogsOfDay(ctx context.Context, daysLookback int) ([]aliyun.NginxLog, error) {
	anchorDay := time.Now().AddDate(0, 0, -daysLookback).Format("2006-01-02")

	logs := []aliyun.NginxLog{}
	businessLogs, err := aliyun.NginxIngressLogQuery(ctx, daysLookback)
	if err != nil {
		return nil, err
	}
	modelLogs, err := aliyun.ModelNginxIngressLogQuery(ctx, daysLookback)
	if err != nil {
		return nil, nil
	}
	// 上报日志的时间，与阿里云将日志入库的时间有可能不同，会导致当天最后一段时间的日志可能落在了第二天内
	// 这里需要用真实的日志时间来调整
	for _, log := range businessLogs {
		if log.Time.Format("2006-01-02") == anchorDay {
			logs = append(logs, log)
		}
	}
	for _, log := range modelLogs {
		if log.Time.Format("2006-01-02") == anchorDay {
			logs = append(logs, log)
		}
	}
	return logs, nil

}

func realtimeNginxLogs(ctx context.Context) map[int][]aliyun.NginxLog {
	var wg sync.WaitGroup
	var mu sync.Mutex
	nginxLogs := map[int][]aliyun.NginxLog{}
	days := []int{0, 1, 7}
	for _, day := range days {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			logs, err := NginxLogsOfDay(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "failed to get nginx logs: %v", err)
				return
			}
			mu.Lock()
			nginxLogs[daysLookback] = logs
			mu.Unlock()
		}(day)
	}
	wg.Wait()
	return nginxLogs
}

type Metric struct {
	Value        int
	DayOverDay   float64
	WeekOverWeek float64
}

type RealtimeReport struct {
	Availability Metric
	TotalRequest Metric
	ErrorRequest Metric
	ProbeFailCnt Metric
}

func RealtimeAvailability(ctx context.Context) (RealtimeReport, error) {
	report := RealtimeReport{}
	systemFactors := map[int]utils.SystemStablityFactor{}
	nginxLogs := realtimeNginxLogs(ctx)
	probeAnlz, err := RealtimeProbeLoganlz(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "failed to get probe logs: %v", err)
	}

	totalRequests := map[int]int{}
	errorRequests := map[int]int{}
	for day, logs := range nginxLogs {
		totalRequests[day] = len(logs)

		errorRequests[day] = 0

		for _, log := range logs {
			if log.Status != "200" {
				errorRequests[day]++
			}
		}
	}

	report.TotalRequest.Value = totalRequests[0]
	report.TotalRequest.DayOverDay = utils.DeltaPercent(float64(totalRequests[1]), float64(totalRequests[0]))
	report.TotalRequest.WeekOverWeek = utils.DeltaPercent(float64(totalRequests[7]), float64(totalRequests[0]))

	report.ErrorRequest.Value = errorRequests[0]
	report.ErrorRequest.DayOverDay = utils.DeltaPercent(float64(errorRequests[1]), float64(errorRequests[0]))
	report.ErrorRequest.WeekOverWeek = utils.DeltaPercent(float64(errorRequests[7]), float64(errorRequests[0]))

	systemFactors[0] = utils.SystemStablityFactor{
		ApiFailRate:   float64(errorRequests[0]) / float64(totalRequests[0]),
		ProbeFailRate: float64(probeAnlz.FailNodes.Value) / float64(probeAnlz.TotalNodes.Value),
	}
	systemFactors[1] = utils.SystemStablityFactor{ApiFailRate: float64(errorRequests[1]) / float64(totalRequests[1])}
	systemFactors[7] = utils.SystemStablityFactor{ApiFailRate: float64(errorRequests[7]) / float64(totalRequests[7])}
	report.Availability.Value = utils.ComputeStablityScore(systemFactors[0])
	report.Availability.DayOverDay = utils.DeltaPercent(float64(utils.ComputeStablityScore(systemFactors[1])), float64(utils.ComputeStablityScore(systemFactors[0])))
	report.Availability.WeekOverWeek = utils.DeltaPercent(float64(utils.ComputeStablityScore(systemFactors[7])), float64(utils.ComputeStablityScore(systemFactors[0])))

	report.ProbeFailCnt.Value = probeAnlz.FailNodes.Value
	report.ProbeFailCnt.DayOverDay = probeAnlz.FailNodes.DayOverDay
	report.ProbeFailCnt.WeekOverWeek = probeAnlz.FailNodes.WeekOverWeek

	return report, nil
}

func SceneTimespanReport(ctx context.Context, timespan int) ([]aliyun.SceneOverviews, error) {
	days := []int{}
	if timespan == consts.TIMESPAN_TODAY {
		days = append(days, 0)
	} else if timespan == consts.TIMESPAN_WEEK {
		for i := 0; i < 7; i++ {
			days = append(days, i)
		}
	} else if timespan == consts.TIMESPAN_LONGTIME {
		now := time.Now()
		start := time.Date(2024, 7, 1, 0, 0, 0, 0, now.Location())
		totalDays := int(time.Since(start).Hours() / 24)

		for i := 0; i < totalDays; i++ {

			days = append(days, i)
		}
	}
	overviews := aliyun.SummaryGeneralOverview(ctx, days)

	return overviews, nil
}

func CreateOrUpdateDatabase(ctx context.Context, timespan int) error {
	nginxReport, err := NginxTimespanReport(ctx, timespan)
	if err != nil {
		return err
	}
	businessTimeSpanReport, err := LogStoreTimeSpanReport(ctx, timespan)
	if err != nil {
		return err
	}
	dailyOverview, err := SystemTimespanAvailability(ctx, timespan)
	if err != nil {
		return err
	}
	sceneOverview, err := SceneTimespanReport(ctx, timespan)
	if err != nil {
		return err
	}

	if timespan == consts.TIMESPAN_LONGTIME {
		if err := mongo.NewApifailureModelDao().DropTable(ctx); err != nil {
			return err
		}
		if err := mongo.NewApicostModelDao().DropTable(ctx); err != nil {
			return err
		}
		if err := mongo.NewSystemScoreDao().DropTable(ctx); err != nil {
			return err
		}
		if err := mongo.NewSceneModelDao().DropTable(ctx); err != nil {
			return err
		}
	}

	dao := mongo.NewApifailureModelDao()
	for _, report := range nginxReport {
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			return err
		}
		model := mongo.ApiFailureModel{
			Date:          date,
			ApiName:       report.CoreApiName,
			HostName:      report.HostName,
			FailCnt:       int32(report.FailCount),
			TotalCnt:      int32(report.TotalCount),
			ErrCode3xxCnt: int32(report.FailStatus3xx),
			ErrCode4xxCnt: int32(report.FailStatus4xx),
			ErrCode5xxCnt: int32(report.FailStatus5xx),
			Status:        mongo.StatusValid,
			CreateTime:    time.Now(),
			UpdateTime:    time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, model.Date, model.ApiName, model); err != nil {
			return err
		}
	}

	for _, report := range businessTimeSpanReport {
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			return err
		}
		model := mongo.ApiCostModel{
			Date:                    date,
			ApiName:                 report.Node,
			ReqCnt:                  report.NumReq,
			AvgCost:                 report.AvgCost,
			CostDistribution0_1:     report.CostDistribution0_1,
			CostDistribution1_3:     report.CostDistribution1_3,
			CostDistribution3_5:     report.CostDistribution3_5,
			CostDistribution5_10:    report.CostDistribution5_10,
			CostDistribution10_20:   report.CostDistribution10_20,
			CostDistribution20_30:   report.CostDistribution20_30,
			CostDistribution30_50:   report.CostDistribution30_50,
			CostDistribution50_100:  report.CostDistribution50_100,
			CostDistribution100_inf: report.CostDistribution100_inf,
			Status:                  mongo.StatusValid,
			CreateTime:              time.Now(),
			UpdateTime:              time.Now(),
		}
		if err := mongo.NewApicostModelDao().CreateOrUpdate(ctx, model.Date, model.ApiName, model); err != nil {
			//t.Errorf("save api cost model error: %s", err)
		}
	}

	for _, report := range dailyOverview {
		dao := mongo.NewSystemScoreDao()
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			return err
		}
		model := mongo.SystemScoreModel{
			Date:       date,
			Score:      report.Score,
			Status:     mongo.StatusValid,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model); err != nil {
			return err
		}
	}

	for _, report := range sceneOverview {
		dao := mongo.NewSceneModelDao()
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			return err
		}
		model := mongo.SceneModel{
			Date:       date,
			Scene:      report.AbstractOverview.Name,
			TotalCnt:   int32(report.AbstractOverview.TotalReq),
			FailCnt:    int32(report.AbstractOverview.FailReq),
			SlowCnt:    int32(report.AbstractOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}

		model = mongo.SceneModel{
			Date:       date,
			Scene:      report.OutlineOverview.Name,
			TotalCnt:   int32(report.OutlineOverview.TotalReq),
			FailCnt:    int32(report.OutlineOverview.FailReq),
			SlowCnt:    int32(report.OutlineOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}

		model = mongo.SceneModel{
			Date:       date,
			Scene:      report.ViewpointOverview.Name,
			TotalCnt:   int32(report.ViewpointOverview.TotalReq),
			FailCnt:    int32(report.ViewpointOverview.FailReq),
			SlowCnt:    int32(report.ViewpointOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}

		model = mongo.SceneModel{
			Date:       date,
			Scene:      report.MultiOverview.Name,
			TotalCnt:   int32(report.MultiOverview.TotalReq),
			FailCnt:    int32(report.MultiOverview.FailReq),
			SlowCnt:    int32(report.MultiOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}

		model = mongo.SceneModel{
			Date:       date,
			Scene:      report.MultiAnalysisOverview.Name,
			TotalCnt:   int32(report.MultiAnalysisOverview.TotalReq),
			FailCnt:    int32(report.MultiAnalysisOverview.FailReq),
			SlowCnt:    int32(report.MultiAnalysisOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}

		model = mongo.SceneModel{
			Date:       date,
			Scene:      report.MultiMergeOverview.Name,
			TotalCnt:   int32(report.MultiMergeOverview.TotalReq),
			FailCnt:    int32(report.MultiMergeOverview.FailReq),
			SlowCnt:    int32(report.MultiMergeOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}

		model = mongo.SceneModel{
			Date:       date,
			Scene:      report.QaOverview.Name,
			TotalCnt:   int32(report.QaOverview.TotalReq),
			FailCnt:    int32(report.QaOverview.FailReq),
			SlowCnt:    int32(report.QaOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}

		model = mongo.SceneModel{
			Date:       date,
			Scene:      report.QaRecommendOverview.Name,
			TotalCnt:   int32(report.QaRecommendOverview.TotalReq),
			FailCnt:    int32(report.QaRecommendOverview.FailReq),
			SlowCnt:    int32(report.QaRecommendOverview.SlowReq),
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.CreateOrUpdate(ctx, date, model.Scene, model); err != nil {
			return err
		}
	}

	return nil

}
