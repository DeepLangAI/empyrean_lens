package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
	"empyrean_lens/dal/redis"
	"empyrean_lens/utils"
	"fmt"
	"testing"
	"time"
)

func TestSystemAvailability(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	s, err := SystemTimespanAvailability(ctx, consts.TIMESPAN_TODAY)
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
	CreateOrUpdateDatabase(ctx, consts.TIMESPAN_TODAY, false)
	//CreateOrUpdateDatabase(ctx, consts.TIMESPAN_TODAY)
}

func TestSystemTimespanAvailability(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	availability, err := SystemTimespanAvailability(ctx, consts.TIMESPAN_TODAY)
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
	rates, cnts, err := SlowQueryRate(ctx, consts.TIMESPAN_TODAY)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(rates)
		fmt.Println(cnts)
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

func TestZSet_Contains(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	zset := utils.NewZSet(redis.GetRdb(), "test", time.Second*5)
	//zset.Add(ctx, "wenhao1")

	contains := zset.Contains(ctx, "wenhao")
	fmt.Println(contains)

	contains = zset.Contains(ctx, "wenhao1")
	fmt.Println(contains)
}
