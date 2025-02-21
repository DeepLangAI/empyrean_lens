package http

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestUmengDal_GetCrashInfoByTime(t *testing.T) {
	ctx := context.Background()
	beginTime := time.Date(2025, 2, 9, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2025, 2, 10, 0, 0, 0, 0, time.Local)
	info, err := UmengDal.GetCrashInfoByTime(ctx, beginTime, endTime)
	if err != nil {
		t.Error(err)
	}
	for _, item := range info {
		fmt.Printf("%v\n", item)
	}
}
