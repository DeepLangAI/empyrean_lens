package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"testing"
	"time"
)

func TestTracebackLogModelDao_CreateOrUpdate(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewTracebackLogModelDao()
	date := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	model := TracebackLogModel{
		ExcInfo:    "123",
		Msg:        "",
		TraceId:    "",
		UserId:     "",
		Time:       date,
		OriginLog:  nil,
		Status:     0,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}
	err := dao.CreateOrUpdate(ctx, model)
	if err != nil {
		t.Error(err)
	}
}
