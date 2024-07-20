package probe

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/go-co-op/gocron"
	"path/filepath"
	"time"
)

type ProbeRunner struct {
}

func (self *ProbeRunner) Run(ctx context.Context) {
	s := gocron.NewScheduler(time.UTC)
	s.Every(10).Minutes().Do(func() {
		funcMap := RegisterFunctions()
		projRoot := utils.GetProjectPath()
		graph, err := LoadGraphFromConfig(filepath.Join(projRoot, consts.GRAPH_CONFIG_PATH), funcMap)
		if err != nil {
			hlog.CtxErrorf(ctx, "LoadGraphFromConfig failed: %v", err)
			return
		}
		graph.PrintGraph()
		graph.Trace(ctx)
	})
	s.StartBlocking()
}
