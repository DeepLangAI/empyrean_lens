package plugin

import (
	"context"
	"testing"
	"time"

	"empyrean_lens/conf"
	"empyrean_lens/consts"
)

func TestMultiDaoFind(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	Init(ctx)

	t1, _ := time.Parse(consts.DateHourMinuteTemplate, "2023-07-01 00:00:00")
	t2, _ := time.Parse(consts.DateHourMinuteTemplate, "2024-10-30 00:00:00")

	t.Run("find multi", func(t *testing.T) {
		d := NewMultiDao()
		model, err := d.FindMultiByTimeRange(ctx, t1, t2, 0, 10)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(len(model))
		}
	})

	t.Run("title", func(t *testing.T) {
		d := NewMultiDao()
		models, err := d.FindMultiByQueryAndTimeRange(ctx, "总结", t1, t2, 0, 10)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(models[0].Title)
		}
	})
}
