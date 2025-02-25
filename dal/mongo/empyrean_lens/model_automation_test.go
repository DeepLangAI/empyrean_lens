package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"testing"
	"time"
)

func TestModelAutomationDao_GetModelAutomationInfoByTime(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewModelAutomationDao()
	beginTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()
	result, err := dao.GetModelAutomationInfoByTime(ctx, beginTime, endTime)
	if err != nil {
		t.Error(err)
	}
	for _, item := range result {
		t.Logf("%+v\n", item)
	}
}
