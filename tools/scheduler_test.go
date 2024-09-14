package tools

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/empyrean_lens"
	aliyun2 "empyrean_lens/service/aliyun"
	"fmt"
	"testing"
)

func TestTodyScore(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	aliyun.Init(ctx)
	empyrean_lens.Init(ctx)

	dailyOverview, err := aliyun2.SystemTimespanAvailability(ctx, consts.TIMESPAN_TODAY)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%+v\n", dailyOverview)
}

func TestProbeRunner_Run(t *testing.T) {
	ctx := context.Background()
	runner := ProbeRunner{}
	runner.Run(ctx)
}
