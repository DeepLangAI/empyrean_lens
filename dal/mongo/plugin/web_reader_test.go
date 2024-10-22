package plugin

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/utils"
	"testing"
	"time"
)

func TestWebReaderDaoFind(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	Init(ctx)

	t1, _ := time.Parse("2006-01-02", "2024-01-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-30")
	// content := "52dde590491d4a1f898f9d1761a2c11e"

	t.Run("web reader find", func(t *testing.T) {
		d := &WebReaderDao{}
		res, err := d.FindWebReaderByTimeRange(ctx, []int32{0}, t1, t2, 0, 10)
		if err != nil {
			t.Errorf("error: %v", err)
		}
		t.Log(utils.JSONMarshal(res))
		// t.Log(res[0].CreateTime.Format(consts.DateTimeTemplate))
		// t.Log(res[0].CreateTime)
		// t.Log(res[0].UpdateTime)
		// t.Log(res[0].UpdateTime.Sub(res[0].CreateTime).Seconds())
	})
}
