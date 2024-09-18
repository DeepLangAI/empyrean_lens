package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestProbeTimespanFailRate(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	rate, _, _ := ProbeTimespanFailRate(ctx, consts.TIMESPAN_WEEK)
	fmt.Println(rate)
}

func TestRealtimeProbeLoganlz(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	loganlz := RealtimeProbeLoganlz(ctx)
	fmt.Println(loganlz)
}
