package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"testing"
	"time"
)

func TestUpdateLatestAppCrashInfo(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)
	for i := 0; i < 7; i++ {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		err := UpdateLatestAppCrashInfo(ctx, dateStr)
		if err != nil {
			t.Error(err)
		}
	}

}
