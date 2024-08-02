package dal

import (
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/dal/mongo/lingo"
	"time"
)

func Init() {
	ctx := context.Background()
	timeout, cancelFunc := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFunc()
	aliyun.Init(timeout)
	empyrean_lens.Init(timeout)
	lingo.Init(timeout)
}
