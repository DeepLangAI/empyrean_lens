package empyrean_lens

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
	logs, _ := dao.FindTimespanApiProbeLog(ctx, time.Now().AddDate(0, 0, -5), time.Now().AddDate(0, 0, 1))
	for _, log := range logs {
		if !log.Correct {
			fmt.Println(log)
		}
	}
}

func TestApiProbeLogModelDao_Save(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewApiProbeLogModelDao()
	err := dao.Save(ctx, ApiProbeLogModel{
		Scene:      "",
		Api:        "",
		Host:       "",
		IsCore:     false,
		Success:    false,
		Correct:    false,
		Cost:       0,
		Status:     0,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	})
	if err != nil {
		t.Error(err)
	}

}
