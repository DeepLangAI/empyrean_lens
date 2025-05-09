package aliyun

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
	"empyrean_lens/utils"
	"fmt"
	"testing"
)

func TestNginxTimeSpanReport(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	//eldal.Init(ctx)
	report, err := NginxTimespanReport(ctx, consts.TIMESPAN_TODAY)
	if err != nil {
		t.Error(err)
	}
	for _, r := range report {
		if r.CoreApiName != "当日总览" {
			fmt.Printf("%+v\n", r)
		}
		//if r.HostName == "当日总览" {
		//	fmt.Println(r)
		//}
	}
}

func TestNginxApiFailureDetail_BusinessBizError(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	req := empyrean_lens.DailyApiFailureDetailReq{}
	req.DateBegin = "2024-09-26"
	req.Host = "api.lingowhale.com"
	req.Path = "/api/plugin/articles/summary"
	req.CodeType = consts.ErrorCodeTypeBusiness
	data, err := NginxApiFailureDetail(ctx, req)
	if err != nil {
		t.Error(err)
	}
	for _, r := range data {
		fmt.Println(r)
	}

}

func TestNginxApiFailureDetail_Business(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	req := empyrean_lens.DailyApiFailureDetailReq{}
	req.DateBegin = "2024-08-09"
	req.Host = "api.lingoreader.cn"
	req.Path = "/api/plugin/articles/summary"
	data, err := NginxApiFailureDetail(ctx, req)
	if err != nil {
		t.Error(err)
	}
	for _, r := range data {
		fmt.Println(r)
	}

}

func TestNginxApiFailureDetail_Model(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	req := empyrean_lens.DailyApiFailureDetailReq{}
	req.DateBegin = "2024-08-09"
	req.Host = "pdfparser.shenyandayi.com"
	req.Path = "/"
	data, err := NginxApiFailureDetail(ctx, req)
	if err != nil {
		t.Error(err)
	}
	for _, r := range data {
		fmt.Println(r)
	}

}

func TestEndToEndTraceLogs(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	req := empyrean_lens.EndToEndTraceReq{
		DateBegin: "2024-08-09",
		DateEnd:   "",
		TraceID:   "6BGsMju9j7cfC_vjTTNjl",
		Level:     "",
	}
	data, err := EndToEndTraceLogs(ctx, req)
	if err != nil {
		t.Error(err)
	}
	for _, r := range data {
		fmt.Println(r)
	}
}

func TestRequestTrend(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	req := empyrean_lens.RequestTrendReq{Date: "2024-08-27"}
	trend, err := RequestTrend(ctx, req)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(trend.Data0)
	fmt.Println(trend.Data1)
	fmt.Println(trend.Data7)
}

func TestEndToEndUserTraceLogs(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	logs, err := EndToEndUserTraceLogs(ctx, empyrean_lens.EndToEndUserTraceReq{
		TimeBegin: "2024-09-19 10:52:00",
		TimeEnd:   "2024-09-19 11:02:00",
		UserID:    "8c8680b199ef4c0b9eba736908c66722",
	})
	if err != nil {
		t.Error(err)
	}
	for _, r := range logs {
		fmt.Println(r.TraceID, r.NumTotalLogs, r.TimeBegin)
		for scene, sceneLogs := range r.SceneLogs {
			fmt.Println(scene, len(sceneLogs))
		}
	}
}

func TestRequestTrendV2(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	req := empyrean_lens.RequestTrendReq{Date: "2024-09-19"}
	data, err := RequestTrendV2(ctx, req)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(utils.JSONMarshal(data))
}
