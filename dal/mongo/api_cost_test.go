package mongo

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestApiCostDao_FindTimespanCost(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	logs, err := NewApicostModelDao().FindTimespanCost(
		ctx,
		time.Date(2024, 7, 23, 0, 0, 0, 0, time.UTC),
		time.Now(),
	)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}
}
