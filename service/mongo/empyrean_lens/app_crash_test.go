package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"fmt"
	"testing"
	"time"
)

func TestUpdateLatestAppCrashInfo(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)
	for i := 0; i < 15; i++ {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		err := UpdateLatestAppCrashInfo(ctx, dateStr)
		if err != nil {
			t.Error(err)
		}
	}

}

func TestGetDailyAppCrashByTime(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)
	beginTime, endTime := time.Date(2025, 02, 04, 0, 0, 0, 0, time.Local), time.Now().Add(8*time.Hour)
	resp, err := GetDailyAppCrashByTime(ctx, beginTime, endTime)
	if err != nil {
		t.Error(err)
	}
	for _, r := range resp {
		fmt.Printf("%+v\n", r)
	}
}
