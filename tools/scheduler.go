package tools

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/service/aliyun"
	"github.com/go-co-op/gocron"
	"time"
)

type ProbeRunner struct {
}

func (self *ProbeRunner) Run(ctx context.Context) {
	s := gocron.NewScheduler(time.UTC)
	// 每10分钟，运行一次探针
	//s.Every(10).Minutes().Do(func() {
	//	funcMap := probe.RegisterFunctions()
	//	projRoot := utils.GetProjectPath()
	//	graph, err := probe.LoadGraphFromConfig(filepath.Join(projRoot, consts.GRAPH_CONFIG_PATH), funcMap)
	//	if err != nil {
	//		hlog.CtxErrorf(ctx, "LoadGraphFromConfig failed: %v", err)
	//		return
	//	}
	//	graph.PrintGraph()
	//	graph.Trace(ctx)
	//})
	// 每1分钟刷新一下当天的最新数据
	s.Every(1).Minutes().Do(func() {
		aliyun.CreateOrUpdateDatabase(ctx, consts.TIMESPAN_TODAY, false)
	})
	// 由于采集日志不及时，需要晚上刷一下近一周的数据
	s.Every(1).Day().At("23:50").Do(func() {
		aliyun.CreateOrUpdateDatabase(ctx, consts.TIMESPAN_WEEK, false)
	})
	//s.StartBlocking()
	s.StartAsync()
}
