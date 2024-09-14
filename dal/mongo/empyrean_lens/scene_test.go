package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"testing"
)

func TestSceneDao_RmRecentDays(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	err := NewSceneModelDao().RmRecentDays(ctx, 7)
	if err != nil {
		t.Error(err)
	}
}
