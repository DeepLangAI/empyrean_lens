package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestSystemScoreDao_FindScoreByTime(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewSystemScoreDao()
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	result, err := dao.FindTimespanScore(ctx, date, now)
	if err != nil {
		t.Error(err)
	} else {
		for _, model := range result {
			fmt.Println(model)
		}
		//fmt.Println(result)
	}

	//if result == nil {
	//	t.Error("result is nil")
	//}
	//
	model, err := dao.FindScoreByTime(ctx, date)
	if err != nil {
		t.Error(err)
	}
	if model == nil {
		t.Error("model is nil")
	} else {
		fmt.Println(model)
	}
}
