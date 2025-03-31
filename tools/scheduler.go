package tools

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/service/aliyun"
	"empyrean_lens/service/mongo/empyrean_lens"
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

	// 每1分钟刷新一下当天的最新数据
	s.Every(1).Minutes().StartImmediately().Do(func() {
		//empyrean_lens.UpdateLatestScoreInfo(ctx)
		aliyun.CreateOrUpdateDatabase(ctx, consts.TIMESPAN_TODAY, false)
		empyrean_lens.UpdateLatestScoreInfo(ctx) // 更新当天的分数同比、环比信息
	})
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
