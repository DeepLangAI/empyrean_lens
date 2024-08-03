package dal

import (
	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/dal/mongo/lingo"
	"os"
	"time"
)

func Init() {
	ctx := context.Background()
	timeout, cancelFunc := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFunc()
	aliyun.Init(timeout)
	empyrean_lens.Init(timeout)
	env := os.Getenv(constslib.ModeEnvName)
	if env != "prod" {
		lingo.Init(timeout)
	}
}
