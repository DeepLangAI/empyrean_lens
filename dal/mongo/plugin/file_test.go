package plugin

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestFindFileByUserIdAndCreateTime(t *testing.T) {
	conf.TestInit()
	ctx := context.Background()
	Init(ctx)
	d := NewFileDao()
	t1, _ := time.Parse("2006-01-02", "2024-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-01")
	uid := "52dde590491d4a1f898f9d1761a2c11e"
	fmt.Println(d.FindFileByUserIdAndCreateTime(ctx, uid, t1, t2))
}
