package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"testing"
	"time"
)

func TestSaveDailyAutomationStatsByDateRange_ExistingDate(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)

	// 测试用例：使用数据库中已存在的日期 2025-03-14
	existingDate, err := time.Parse("2006-01-02", "2025-03-14")
	if err != nil {
		t.Fatalf("解析日期失败: %v", err)
	}

	// 调用被测试的函数
	err = SaveDailyAutomationStatsByDateRange(ctx, existingDate)
	if err != nil {
		t.Fatalf("SaveDailyAutomationStatsByDateRange失败: %v", err)
	}

	// 验证是否成功跳过（通过检查日志输出）
	// 注意：由于函数本身会打印日志，我们可以在运行测试时观察输出
	// 预期会看到类似 "Data for date 2025-03-14 already exists, skipping" 的日志
}

func TestSaveDailyAutomationStatsByDateRange_NewDate(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)

	// 测试用例：使用一个不存在的日期
	newDate := time.Now().AddDate(0, 0, 1) // 使用明天的日期

	// 调用被测试的函数
	err := SaveDailyAutomationStatsByDateRange(ctx, newDate)
	if err != nil {
		t.Fatalf("SaveDailyAutomationStatsByDateRange失败: %v", err)
	}
}
