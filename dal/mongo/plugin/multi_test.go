package plugin

import (
	"context"
	"empyrean_lens/conf"
	"testing"
	"time"
)

func TestMultiDao_FindMultiByMultiId(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	Init(ctx)

	t1, _ := time.Parse("2006-01-02", "2024-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-01")
	uid := "52dde590491d4a1f898f9d1761a2c11e"

	t.Run("find multi", func(t *testing.T) {
		d := NewMultiDao()
		model, err := d.FindMultiByUserIdAndCreateTime(ctx, uid, t1, t2)
		if err != nil {
			t.Error(err)
		} else {
			t.Log(model)
		}
	})
}
