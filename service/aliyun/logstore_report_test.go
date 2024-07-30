package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestLogStoreTimeSpanReport(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	report, err := LogStoreTimeSpanReport(ctx, consts.TIMESPAN_TODAY)
	if err != nil {
		t.Error(err)
	}
	for _, r := range report {
		fmt.Println(r)
	}
}
