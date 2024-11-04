package link_trace

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/tools"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetWcdDetail(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	aliyun.Init(ctx)

	traceId := "6725d8163dfb8b681e88274a"
	timeBegin := time.Now().AddDate(0, 0, -1)
	timeEnd := timeBegin.AddDate(0, 0, 1)
	ossList, err := aliyun.WcdOsskeyQuery(ctx, traceId, timeBegin, timeEnd)
	assert.Nil(t, err)
	ossOp := tools.GetOssOperator(ctx)
	for _, oss := range ossList {
		file, err := ossOp.DownloadWcdOssFile(oss.Bucket, oss.Key)
		assert.Nil(t, err)
		fmt.Println(file)
	}
}

func TestGetWcdOssLogDetail(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	aliyun.Init(ctx)
	plugin.Init(ctx)
	entryId := "6724e0d6a00135540d6a5036"
	traceId := "6724e0d6a00135540d6a5035"
	req := empyrean_lens.WcdOssDetalReq{
		EntryID: entryId,
		TraceID: traceId,
	}
	detail, code := GetWcdOssLogDetail(ctx, req)
	assert.Nil(t, code)
	fmt.Println(detail)
}
