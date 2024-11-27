package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
	"time"
)

func TestSaveGenerateErrlogByDate(t *testing.T) {
	conf.InitConfig()
	dal.Init()
	// 起始日期和结束日期
	startDate := "2024-11-01"
	endDate := "2024-11-27"

	// 将字符串转换为 time.Time 类型
	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		t.Fatalf("Invalid start date format: %v", err)
	}
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		t.Fatalf("Invalid end date format: %v", err)
	}

	// 循环从 startDate 到 endDate 每天调用一次 SaveGenerateErrlogByDate
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		dateStr := startTime.Format("2006-01-02")
		SaveGenerateErrlogByDate(context.Background(), dateStr)
		startTime = startTime.AddDate(0, 0, 1) // 增加一天
	}
}
