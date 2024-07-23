package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal/mongo"
	"fmt"
	"testing"
	"time"
)

func TestProbeLogAnlz(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	mongo.Init(ctx)
	ans, _ := RealtimeProbeLoganlz(ctx)
	fmt.Println(ans)
}

func TestProbeErrorRate(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	mongo.Init(ctx)
	now := time.Now()
	begin := time.Date(2024, 7, 1, 0, 0, 0, 0, now.Location())
	errRates, err := ProbeErrorRate(ctx, begin, now)
	if err != nil {
		t.Errorf("ProbeErrorRate error: %v", err)
	} else {
		for key, val := range errRates {
			fmt.Println(key, val)
		}
	}
}
