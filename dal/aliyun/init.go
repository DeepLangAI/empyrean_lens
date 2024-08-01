package aliyun

import (
	"context"
	"empyrean_lens/consts"
	sls "github.com/aliyun/aliyun-log-go-sdk"
)

var client sls.ClientInterface

func Init(ctx context.Context) {
	client = sls.CreateNormalInterfaceV2(
		consts.ENDPOINT,
		sls.NewStaticCredentialsProvider(
			consts.ACCESS_KEY_ID,
			consts.ACCESS_KEY_SECRET,
			consts.SECURE_TOKEN,
		),
	)
}

func GetSlsClient() sls.ClientInterface {
	return client
}
