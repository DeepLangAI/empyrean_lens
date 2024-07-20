package dal

import (
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo"
)

func Init() {
	ctx := context.Background()
	aliyun.Init(ctx)
	mongo.Init(ctx)
}
