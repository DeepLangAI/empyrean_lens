package dal

import (
	"context"
	"empyrean_lens/dal/aliyun"
)

func Init() {
	ctx := context.Background()
	aliyun.Init(ctx)
}
