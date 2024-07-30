package mongo

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestApiProbeLogModelDao_FindTimespanApiProbeLog(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewApiProbeLogModelDao()
	logs, _ := dao.FindTimespanApiProbeLog(ctx, time.Now().AddDate(0, 0, -2), time.Now().AddDate(0, 0, 1))
	for _, log := range logs {
		if !log.Correct {
			fmt.Println(log)
		}
	}
}
