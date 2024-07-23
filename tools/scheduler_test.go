package tools

import (
	"context"
	aliyun1 "empyrean_lens/aliyun"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo"
	"testing"
	"time"
)

func TestUpdateDatabase(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	aliyun.Init(ctx)
	mongo.Init(ctx)
	nginxReports, err := aliyun1.NginxTimespanReport(ctx, consts.TIMESPAN_WEEK)
	if err != nil {
		t.Error(err)
		return
	}

	businessTimeSpanReports, err := aliyun1.LogStoreTimeSpanReport(ctx, consts.TIMESPAN_WEEK)
	if err != nil {
		t.Error(err)
		return
	}

	sysRevent, err := aliyun1.SystemTimespanAvailability(ctx, consts.TIMESPAN_WEEK)
	if err != nil {
		t.Error(err)
		return
	}

	for _, report := range nginxReports {
		t.Log(report)
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			t.Errorf("date format error: %s", report.Date)
			return
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
		if err := mongo.NewApifailureModelDao().CreateOrUpdate(ctx, model.Date, model.ApiName, model); err != nil {
			t.Error(err)
			return
		}
	}

	for _, report := range businessTimeSpanReports {
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			t.Errorf("date format error: %s", report.Date)
			return
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
			t.Errorf("save api cost model error: %s", err)
		}
	}

	for _, report := range sysRevent {
		dao := mongo.NewSystemScoreDao()
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			t.Errorf("date format error: %s", report.Date)
			return
		}
		model := mongo.SystemScoreModel{
			Date:       date,
			Score:      report.Score,
			Status:     mongo.StatusValid,
			CreateTime: time.Time{},
			UpdateTime: time.Time{},
		}
		if err := dao.CreateOrUpdate(ctx, date, model); err != nil {
			t.Error(err)
			return
		}
	}
}

func TestInitDatabase(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	aliyun.Init(ctx)
	mongo.Init(ctx)

	nginxReport, err := aliyun1.NginxTimespanReport(ctx, consts.TIMESPAN_LONGTIME)
	if err != nil {
		t.Error(err)
		return
	}
	businessTimeSpanReport, err := aliyun1.LogStoreTimeSpanReport(ctx, consts.TIMESPAN_LONGTIME)
	if err != nil {
		t.Error(err)
		return
	}
	dailyOverview, err := aliyun1.SystemTimespanAvailability(ctx, consts.TIMESPAN_LONGTIME)
	if err != nil {
		t.Error(err)
		return
	}

	//mongo.NewApifailureModelDao().DropTable(ctx)
	//mongo.NewApicostModelDao().DropTable(ctx)

	dao := mongo.NewApifailureModelDao()
	for _, report := range nginxReport {
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			t.Errorf("date format error: %s", report.Date)
			return
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
			t.Error(err)
			return
		}
	}

	for _, report := range businessTimeSpanReport {
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			t.Errorf("date format error: %s", report.Date)
			return
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
			t.Errorf("save api cost model error: %s", err)
		}
	}

	for _, report := range dailyOverview {
		dao := mongo.NewSystemScoreDao()
		date, err := time.Parse("2006-01-02", report.Date)
		if err != nil {
			t.Errorf("date format error: %s", report.Date)
			return
		}
		model := mongo.SystemScoreModel{
			Date:       date,
			Score:      report.Score,
			Status:     mongo.StatusValid,
			CreateTime: time.Time{},
			UpdateTime: time.Time{},
		}
		if err := dao.CreateOrUpdate(ctx, date, model); err != nil {
			t.Error(err)
			return
		}

	}
}

func TestProbeRunner_Run(t *testing.T) {
	ctx := context.Background()
	runner := ProbeRunner{}
	runner.Run(ctx)
}
