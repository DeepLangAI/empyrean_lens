package service

import (
	"context"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestLogStoreMonthReport(t *testing.T) {
	ctx := context.Background()
	dal.Init()
	report, err := LogStoreMonthReport(ctx)
	if err != nil {
		t.Error(err)
	}
	for _, r := range report {
		fmt.Println(r)
	}
}
