package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"fmt"
	"testing"
	"time"
)

func TestAppCrashModelDao_GetAppCrashInfoByTime(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewAppCrashModelDao()
	logs, err := dao.GetAppCrashInfoByTime(ctx, time.Now().AddDate(0, 0, -5), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Error(err)
	}
	for _, log := range logs {
		fmt.Printf("%v+\n", log)
	}
}

func TestAppCrashModelDao_Save(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewAppCrashModelDao()
	err := dao.Save(ctx, AppCrashModel{
		Time:              time.Now(),
		PlatformType:      "IOS",
		ErrorCount:        0,
		LaunchCount:       321,
		AffectedUserCount: 0,
		ActiveUserCount:   84,
		Status:            0,
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestAppCrashModelDao_SaveBatch(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewAppCrashModelDao()
	err := dao.SaveBatch(ctx, []AppCrashModel{
		{
			Time:              time.Now(),
			PlatformType:      consts.Platform_IOS,
			ErrorCount:        0,
			LaunchCount:       321,
			AffectedUserCount: 0,
			ActiveUserCount:   84,
			Status:            0,
			CreateTime:        time.Now(),
			UpdateTime:        time.Now(),
		},
		{
			Time:              time.Now(),
			PlatformType:      consts.Platform_Android,
			ErrorCount:        0,
			LaunchCount:       207,
			AffectedUserCount: 0,
			ActiveUserCount:   94,
			Status:            0,
			CreateTime:        time.Now(),
			UpdateTime:        time.Now(),
		},
		{
			Time:              time.Date(2025, 2, 9, 0, 0, 0, 0, time.Local),
			PlatformType:      consts.Platform_IOS,
			ErrorCount:        0,
			LaunchCount:       707,
			AffectedUserCount: 0,
			ActiveUserCount:   174,
			Status:            0,
			CreateTime:        time.Now(),
			UpdateTime:        time.Now(),
		},
		{
			Time:              time.Date(2025, 2, 9, 0, 0, 0, 0, time.Local),
			PlatformType:      consts.Platform_Android,
			ErrorCount:        4,
			LaunchCount:       445,
			AffectedUserCount: 4,
			ActiveUserCount:   187,
			Status:            0,
			CreateTime:        time.Now(),
			UpdateTime:        time.Now(),
		},
	})
	if err != nil {
		t.Error(err)
	}
}
