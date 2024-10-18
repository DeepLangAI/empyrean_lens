package plugin

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"testing"
	"time"
)

func TestMultiDaoFind(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	Init(ctx)

	t1, _ := time.Parse(consts.DateHourMinuteTemplate, "2023-07-01 00:00:00")
	t2, _ := time.Parse(consts.DateHourMinuteTemplate, "2024-10-30 00:00:00")
	uid := "52dde590491d4a1f898f9d1761a2c11e"

	t.Run("find multi", func(t *testing.T) {
		d := NewMultiDao()
		model, err := d.FindMultiByUserIdAndCreateTime(ctx, uid, t1, t2)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(len(model))
		}
	})

	t.Run("title", func(t *testing.T) {
		d := NewMultiDao()
		models, err := d.FindMultiByTitleAndCreateTime(ctx, "总结", t1, t2)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(models[0].Title)
		}
	})
}
