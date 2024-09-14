package empyrean_lens

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"testing"
	"time"
)

func TestProbeTimespanFailRate(t *testing.T) {
	d := map[string]int{}
	d["1"] += 1
	d["1"] += 1
	d["2"] += 1
	fmt.Println(d)
	fmt.Println(d["1"], d["3"])
}

func TestProbeReport(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	report, err := ProbeReport(ctx)
	if err != nil {
		t.Errorf("ProbeReport err: %v", err)
	}
	for _, r := range report {
		fmt.Println(r)
	}
}

func TestProbeListInfo(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	req := empyrean_lens.ApiProbeReq{
		DateBegin: "2024-08-12",
	}
	info, err := ProbeListInfo(ctx, req)
	if err != nil {
		t.Errorf("ProbeListInfo err: %v", err)
	}
	for _, r := range info {
		fmt.Println(r)
	}
}

func TestProbeDetail(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	req := empyrean_lens.ProbeLogDetailReq{
		DateBegin:  "2024-08-12",
		DateEnd:    "",
		Scene:      "",
		NotCorrect: false,
		NotSuccess: false,
	}
	detail, err := ProbeDetail(ctx, req)
	if err != nil {
		t.Errorf("ProbeDetail err: %v", err)
	}
	for _, r := range detail {
		fmt.Println(r)
	}
}

func TestDateTime(t *testing.T) {
	dateStr := "2024-08-12"
	date, _ := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	fmt.Println(date)
	fmt.Println(time.Now())
	fmt.Println(time.Now().Sub(date).Hours())
}
