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
	rate, _ := ProbeTimespanFailRate(ctx, consts.TIMESPAN_WEEK)
	fmt.Println(rate)
}
