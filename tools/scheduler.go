package tools

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/service/mongo/empyrean_lens"
	"empyrean_lens/service/shence"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"github.com/go-co-op/gocron"
)

type ProbeRunner struct {
}

func GetCurrentDate() string {
	now := time.Now()                      // 获取当前时间
	return now.Format(consts.DateTemplate) // 格式化为 "YYYYMMDD"
}

func (self *ProbeRunner) Run(ctx context.Context) {
	s := gocron.NewScheduler(time.Local)

	////// 首次同步数据
	//hlog.CtxInfof(ctx, "开始执行首次新内容形态页加载性能数据及趋势数据同步")
	//startTime := time.Date(2024, 10, 17, 0, 0, 0, 0, time.Local)
	//endTime := time.Now()
	//
	//if err := shence.SyncApiPerformanceStats(ctx, startTime, endTime); err != nil {
	//	hlog.CtxErrorf(ctx, "首次同步新内容形态页加载性能数据失败: %v", err)
	//} else {
	//	hlog.CtxInfof(ctx, "首次同步新内容形态页加载性能数据成功")
	//}
	//
	//if err := shence.SyncApiPerformanceTrend(ctx, startTime, endTime); err != nil {
	//	hlog.CtxErrorf(ctx, "首次同步新内容形态页加载性能趋势数据失败: %v", err)
	//} else {
	//	hlog.CtxInfof(ctx, "首次同步新内容形态页加载性能趋势数据成功")
	//}
	//
	//if err := shence.SyncApiPerformanceVersionTrend(ctx, startTime, endTime); err != nil {
	//	hlog.CtxErrorf(ctx, "首次同步系统所有版本新内容形态页加载性能数据失败: %v", err)
	//} else {
	//	hlog.CtxInfof(ctx, "首次同步系统所有版本新内容形态页加载性能数据成功")
	//}

	//每1分钟同步新内容形态页加载性能数据
	s.Every(1).Minutes().StartImmediately().Do(func() {
		hlog.CtxInfof(ctx, "开始同步新内容形态页加载性能数据")
		now := time.Now()
		if err := shence.SyncApiPerformanceStats(ctx, now, now); err != nil {
			hlog.CtxErrorf(ctx, "同步新内容形态页加载性能数据失败: %v", err)
		} else {
			hlog.CtxInfof(ctx, "同步新内容形态页加载性能数据成功")
		}
	})

	//每2小时同步一次新内容形态页加载性能趋势数据
	s.Every(2).Hours().StartImmediately().Do(func() {
		hlog.CtxInfof(ctx, "开始同步新内容形态页加载性能趋势数据（每2小时）")
		now := time.Now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if err := shence.SyncApiPerformanceTrend(ctx, todayStart, now); err != nil {
			hlog.CtxErrorf(ctx, "同步新内容形态页加载性能趋势数据失败: %v", err)
		} else {
			hlog.CtxInfof(ctx, "同步新内容形态页加载性能趋势数据成功")
		}
		// 新增：同步所有版本API性能趋势数据
		if err := shence.SyncApiPerformanceVersionTrend(ctx, todayStart, now); err != nil {
			hlog.CtxErrorf(ctx, "同步所有版本API性能趋势数据失败: %v", err)
		} else {
			hlog.CtxInfof(ctx, "同步所有版本API性能趋势数据成功")
		}
	})

	//// 每1分钟刷新一下当天的最新数据
	//s.Every(1).Minutes().StartImmediately().Do(func() {
	//	//empyrean_lens.UpdateLatestScoreInfo(ctx)
	//	aliyun.CreateOrUpdateDatabase(ctx, consts.TIMESPAN_TODAY, false)
	//	empyrean_lens.UpdateLatestScoreInfo(ctx) // 更新当天的分数同比、环比信息
	//})

	//// 由于采集日志不及时，需要晚上刷一下近一周的数据
	//s.Every(1).Day().At("23:50").Do(func() {
	//	aliyun.CreateOrUpdateDatabase(ctx, consts.TIMESPAN_WEEK, false)
	//})
	//s.StartBlocking()
	//每10分钟获取一次前端上报到神策对异常日志信息
	s.Every(10).Minutes().StartImmediately().Do(func() {
		empyrean_lens.SaveUploadLogByDate(ctx, GetCurrentDate())
		empyrean_lens.SaveGenerateErrlogByDate(ctx, GetCurrentDate())
		empyrean_lens.UpdateLatestAppCrashInfo(ctx, GetCurrentDate())
	})

	// 添加新的定时任务：每天早上6点搬移model_case_result中的数据到daily_automation_stats中
	s.Every(1).Day().At("06:00").Do(func() {
		if err := empyrean_lens.SaveOnceDailyAutomationStats(context.Background()); err != nil {
			hlog.Errorf("Failed to migrate historical data: %v", err)
		}
	})

	// 添加新的定时任务：每天早上12点补充api_stats_daily的历史数据
	s.Every(1).Day().At("12:00").Do(func() {
		if err := empyrean_lens.SaveOnceDailyApiStats(context.Background()); err != nil {
			hlog.Errorf("Failed to migrate API stats historical data: %v", err)
		}
	})

	//// 添加API统计数据计算任务
	//s.Every(1).Day().At("12:00").Do(func() {
	//	hlog.CtxInfof(ctx, "开始执行每日API统计数据计算任务")
	//
	//	err := empyrean_lens.CalculateAndSaveDailyStats(ctx, time.Now())
	//	if err != nil {
	//		hlog.CtxErrorf(ctx, "计算每日API统计数据失败: %v", err)
	//	} else {
	//		hlog.CtxInfof(ctx, "每日API统计数据计算完成")
	//	}
	//})

	s.StartAsync()

}
