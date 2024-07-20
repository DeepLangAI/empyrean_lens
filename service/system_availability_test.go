package service

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
)

func TestSystemAvailability(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	s, err := SystemAvailability(ctx)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(s)
	}
}

func TestRealtimeAvailability(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	if report, err := RealtimeAvailability(ctx); err != nil {
		t.Error(err)
	} else {
		t.Log(report)
	}

}
