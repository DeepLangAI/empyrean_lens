package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
)

func TestSaveUploadLogByDate(t *testing.T) {
	conf.InitConfig()
	dal.Init()
	SaveOnceUploadLogByDate()
}

func TestSaveUploadLogByDate1(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	SaveUploadLogByDate(ctx, "2024-12-02")

}
