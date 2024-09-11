package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
)

func TestUpdateLatestScoreInfo(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	UpdateLatestScoreInfo(ctx)
}
