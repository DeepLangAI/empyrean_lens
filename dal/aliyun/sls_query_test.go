package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
)

func TestQaRecommendFailcntQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	failCnt := QaRecommendFailcntQuery(ctx, 0)
	fmt.Println(failCnt)
}

func TestQaRecommendSuccessQuerry(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	logs, err := QaRecommendAllQuerry(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("共调用推荐模型次数：", len(logs))
		for _, log := range logs {
			fmt.Println(log)
		}
	}
}

func TestNginxErrlogsQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	query, err := NginxErrlogsQuery(ctx, "api.lingoreader.cn", "/api/plugin/articles/summary", "2024-08-07")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(len(query))
		fmt.Println(query)
	}
}

func TestModelNginxErrlogsQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	query, err := ModelNginxErrlogsQuery(ctx, "outline-verbose.shenyandayi.com", "/generate", "2024-08-07")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(len(query))
		fmt.Println(query)
	}

}
