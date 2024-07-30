package mongo

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
	"time"
)

func TestDailyPerformanceResult(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	timeBegin := time.Date(2024, 7, 25, 0, 0, 0, 0, time.UTC)
	timeEnd := time.Date(2024, 7, 26, 0, 0, 0, 0, time.UTC)
	DailyPerformanceResult(ctx, timeBegin, timeEnd)
}
