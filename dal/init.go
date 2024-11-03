package dal

import (
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/empyrean_lens"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/dal/redis"
	"time"
)

func Init() {
	ctx := context.Background()
	timeout, cancelFunc := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFunc()
	aliyun.Init(timeout)
	empyrean_lens.Init(timeout)
	plugin.Init(timeout)
	bi.Init(timeout)
	redis.Init()
	//env := os.Getenv(constslib.ModeEnvName)
	//if env != "prod" {
	//	lingo.Init(timeout)
	//}
}
