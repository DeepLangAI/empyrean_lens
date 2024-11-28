package tools

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/service/aliyun"
	"empyrean_lens/service/mongo/empyrean_lens"
	"time"

	"github.com/go-co-op/gocron"
)

type ProbeRunner struct {
}

func GetCurrentDate() string {
	now := time.Now()             // 获取当前时间
	return now.Format("20060102") // 格式化为 "YYYYMMDD"
}

func (self *ProbeRunner) Run(ctx context.Context) {
	s := gocron.NewScheduler(time.UTC)
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
	s.Every(10).Minutes().Do(func() {
		empyrean_lens.SaveUploadLogByDate(ctx, GetCurrentDate())
		empyrean_lens.SaveGenerateErrlogByDate(ctx, GetCurrentDate())
	})

	s.StartAsync()
}
