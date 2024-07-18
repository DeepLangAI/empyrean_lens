package service

import (
	"context"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestNginxMonthReport(t *testing.T) {
	ctx := context.Background()
	dal.Init()
	report, err := NginxMonthReport(ctx)
	if err != nil {

		t.Error(err)
	}
	for _, r := range report {
		fmt.Println(r)
	}
}
