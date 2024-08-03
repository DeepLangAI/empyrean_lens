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
