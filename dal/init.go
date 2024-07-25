package dal

import (
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo"
	"time"
)

func Init() {
	ctx := context.Background()
	timeout, cancelFunc := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFunc()
	aliyun.Init(timeout)
	mongo.Init(timeout)
}
