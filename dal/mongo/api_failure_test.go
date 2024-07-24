package mongo

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestApiFailureDao_FindTimespanFailure(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	logs, err := NewApifailureModelDao().FindTimespanFailure(
		ctx,
		time.Date(2024, 7, 24, 0, 0, 0, 0, time.UTC),
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
