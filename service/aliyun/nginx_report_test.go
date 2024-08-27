package aliyun

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
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
