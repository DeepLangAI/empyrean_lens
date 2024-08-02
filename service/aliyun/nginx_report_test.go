package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"fmt"
	"testing"
)

func TestNginxTimeSpanReport(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	empyrean_lens.Init(ctx)
	report, err := NginxTimespanReport(ctx, consts.TIMESPAN_WEEK)
	if err != nil {

		t.Error(err)
	}
	for _, r := range report {
		fmt.Println(r)
	}
}
