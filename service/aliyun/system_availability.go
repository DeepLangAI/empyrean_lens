package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo"
	"empyrean_lens/utils"
	"time"
)

func SystemTimespanAvailability(ctx context.Context, timespan int) ([]utils.ReventResult, error) {
	systemFactors := map[string]utils.SystemStablityFactor{}
	availabilityScores := map[string]int{}
	nginxLogs := []NginxTimeSpanReportModel{}

	nginxLogs, err := NginxTimespanReport(ctx, timespan)
	if err != nil {
		return nil, err
	}

	probeFailRates, err := ProbeTimespanFailRate(ctx, timespan)
	if err != nil {
		return nil, err
	}

	slowqueryRates, err := SlowQueryRate(ctx, timespan)
	if err != nil {
		return nil, err
	}

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
		factor.ProbeFailRate = probeFailRates[log.Date] / 100.0
		factor.SlowQueryRate = slowqueryRates[log.Date] / 100.0
		systemFactors[log.Date] = factor
	}
	for date, factor := range systemFactors {
		availabilityScores[date] = utils.ComputeStablityScore(factor)
	}
	reventResults := utils.ComputeRevent(availabilityScores)
	return reventResults, nil
}

func RealtimeSlowqueryLoganlz(ctx context.Context) (*MetricFloat, error) {
	slowRates, err := SlowQueryRate(ctx, consts.TIMESPAN_WEEK)
	if err != nil {
		return nil, err
	}
	slowRates_0 := slowRates[time.Now().AddDate(0, 0, 0).Format("2006-01-02")]
	slowRates_1 := slowRates[time.Now().AddDate(0, 0, -1).Format("2006-01-02")]
	slowRates_7 := slowRates[time.Now().AddDate(0, 0, -7).Format("2006-01-02")]
	metric := &MetricFloat{
		Value:        0,
		DayOverDay:   0,
		WeekOverWeek: 0,
	}
	metric.Value = slowRates_0
	metric.DayOverDay = utils.DeltaPercent(slowRates_1, slowRates_0)
	metric.WeekOverWeek = utils.DeltaPercent(slowRates_7, slowRates_0)
	return metric, nil
}

func SlowQueryRate(ctx context.Context, timespan int) (map[string]float64, error) {
	timeBegin := time.Now()
	timeEnd := time.Date(timeBegin.Year(), timeBegin.Month(), timeBegin.Day(), 23, 59, 59, 0, timeBegin.Location())

	if timespan == consts.TIMESPAN_TODAY {
		anchorDay := time.Now()
		timeBegin = time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, anchorDay.Location())
	} else if timespan == consts.TIMESPAN_WEEK {
		anchorDay := time.Now().AddDate(0, 0, -7)
		timeBegin = time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, anchorDay.Location())
	} else if timespan == consts.TIMESPAN_LONGTIME {
		timeBegin = time.Date(2024, 7, 1, 0, 0, 0, 0, timeBegin.Location())
	}
	dao := mongo.NewSceneModelDao()
	sceneLogs, err := dao.FindTimespanScene(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	totalCnts := map[string]int32{}
	slowCnts := map[string]int32{}
	result := map[string]float64{}
	for _, log := range sceneLogs {
		date := log.Date.Format("2006-01-02")
		totalCnts[date] += log.TotalCnt
		slowCnts[date] += log.SlowCnt
	}
	for date, slowCnt := range slowCnts {
		totalCnt := totalCnts[date]
		if totalCnt == 0 {
			result[date] = 0
		} else {
			result[date] = float64(slowCnt) / float64(totalCnt) * 100
		}
	}
	return result, nil
}

type Metric struct {
	Value        int
	DayOverDay   float64
	WeekOverWeek float64
}

type MetricFloat struct {
	Value        float64
	DayOverDay   float64
	WeekOverWeek float64
}

type RealtimeReport struct {
	Availability Metric
	TotalRequest Metric
	ErrorRequest Metric
	SlowRequest  MetricFloat
	ProbeFailCnt Metric
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
	overviews := aliyun.SceneGeneralOverview(ctx, days)

	return overviews, nil
}

func saveNginxReportWithTransx(ctx context.Context, reports []NginxTimeSpanReportModel) {
	if len(reports) == 0 {
		return
	}
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
			FailReason: report.AbstractOverview.FailReason,
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
			FailReason: report.OutlineOverview.FailReason,
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
			FailReason: report.ViewpointOverview.FailReason,
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
			FailReason: report.MultiOverview.FailReason,
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
			FailReason: report.MultiAnalysisOverview.FailReason,
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
			FailReason: report.MultiMergeOverview.FailReason,
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
			FailReason: report.QaOverview.FailReason,
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
			FailReason: report.QaRecommendOverview.FailReason,
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
