package service

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"empyrean_lens/dal/mongo"
	"fmt"
	"testing"
)

func TestNginxMonthReport(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	mongo.Init(ctx)
	report, err := NginxMonthReport(ctx)
	if err != nil {

		t.Error(err)
	}
	for _, r := range report {
		fmt.Println(r)
	}
}
