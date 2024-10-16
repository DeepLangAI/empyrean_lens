package plugin

import (
	"context"
	"empyrean_lens/conf"
	"testing"
	"time"
)

func TestWebReaderDao_FindFileById(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	Init(ctx)

	t1, _ := time.Parse("2006-01-02", "2024-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-01")
	uid := "52dde590491d4a1f898f9d1761a2c11e"

	t.Run("web reader find", func(t *testing.T) {
		d := &WebReaderDao{}
		res, err := d.FindWebReaderByUserIdAndCreateTime(ctx, uid, t1, t2)
		if err != nil {
			t.Errorf("error: %v", err)
		}
		t.Log(res)
	})
}
