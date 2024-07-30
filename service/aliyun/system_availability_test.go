package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestSystemAvailability(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	s, err := SystemTimespanAvailability(ctx, consts.TIMESPAN_WEEK)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(s)
	}
}

//func TestRealtimeAvailability(t *testing.T) {
//	ctx := context.Background()
//	conf.InitConfig()
//	dal.Init()
//	if report, err := RealtimeAvailability(ctx); err != nil {
//		t.Error(err)
//	} else {
//		t.Log(report)
//	}
//
//}

func TestCreateOrUpdateDatabase(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	//CreateOrUpdateDatabase(ctx, consts.TIMESPAN_LONGTIME)
	CreateOrUpdateDatabase(ctx, consts.TIMESPAN_WEEK)
	//CreateOrUpdateDatabase(ctx, consts.TIMESPAN_TODAY)
}

func TestSystemTimespanAvailability(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	availability, err := SystemTimespanAvailability(ctx, consts.TIMESPAN_WEEK)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(availability)
	}
}

func TestSlowQueryRate(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	rates, err := SlowQueryRate(ctx, consts.TIMESPAN_WEEK)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(rates)
	}
}

func TestRealtimeSlowqueryLoganlz(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	loganlz, err := RealtimeSlowqueryLoganlz(ctx)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(loganlz)
	}
}
