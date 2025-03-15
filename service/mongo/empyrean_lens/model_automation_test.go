package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"fmt"
	"testing"
	"time"
)

func TestGetDailyModelAutomationByTime(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)
	beginTime := time.Now().AddDate(0, 0, -1)
	endTime := time.Now()
	resp, err := GetDailyModelAutomationByTime(ctx, beginTime, endTime, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range resp {
		fmt.Printf("%+v\n", v)
	}
}

func TestGetModelAutomationInfoByTime(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)
	beginTime := time.Now().AddDate(0, 0, -1)
	endTime := time.Now()
	resp, err := GetModelAutomationInfoByTime(ctx, beginTime, endTime, consts.EntryTypeWEB, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range resp {
		fmt.Printf("%+v\n", v)
	}
}

func TestGetCaseResultsByTime(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)
	beginTime := time.Now().AddDate(0, 0, -1)
	endTime := time.Now()
	resp, err := GetCaseResultsByTime(ctx, beginTime, endTime, consts.EntryTypeWEB, "edu_tree_lost", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range resp {
		fmt.Printf("%+v\n", v)
	}
}
