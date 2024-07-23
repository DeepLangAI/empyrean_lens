package service

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestSystemRealtimeReport(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	report, err := SystemRealtimeReport(ctx)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(report)
	}
}
