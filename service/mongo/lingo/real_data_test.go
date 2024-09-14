package lingo

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestRealDataOfDate(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	date, err := RealDataOfDate(ctx, "2024-08-03")
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(date)
	}
}
