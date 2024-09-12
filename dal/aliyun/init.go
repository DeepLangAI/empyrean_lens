package aliyun

import (
	"context"
	"empyrean_lens/consts"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

var client sls.ClientInterface
var logstore *sls.LogStore

func Init(ctx context.Context) {
	client = sls.CreateNormalInterfaceV2(
		consts.ENDPOINT,
		sls.NewStaticCredentialsProvider(
			consts.ACCESS_KEY_ID,
			consts.ACCESS_KEY_SECRET,
			consts.SECURE_TOKEN,
		),
	)
	logStore, err := client.GetLogStore(consts.PROJECT_NAME, consts.MODEL_NGINX_LOG_STORE_NAME)
	if err != nil {
		hlog.CtxErrorf(ctx, "get log store error: %v", err)
	}
	logstore = logStore
}
