package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestSystemTracebackQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	req := empyrean_lens2.TracebackReq{
		DateBegin: "2024-08-10",
		DateEnd:   "",
	}
	query, err := SystemTracebackQuery(ctx, req)
	if err != nil {
		t.Errorf("SystemTracebackQuery failed: %v", err)
	}
	fmt.Println(len(query), query)
}
