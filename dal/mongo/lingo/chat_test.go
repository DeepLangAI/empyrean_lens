package lingo

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestChatDao_FindModels(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	dao := NewChatModelDao()
	now := time.Now()
	timeBegin := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	timeEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local)
	models, err := dao.FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(models)
	}
}
