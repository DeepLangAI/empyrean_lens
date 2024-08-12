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

	//query, err := NginxErrlogsQuery(ctx, "api.lingoreader.cn", "/api/plugin/articles/summary", "2024-08-07")
	query, err := NginxErrlogsQuery(ctx, "", "/api/plugin/articles/summary", "2024-08-07")
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

	//query, err := ModelNginxErrlogsQuery(ctx, "outline-verbose.shenyandayi.com", "/generate", "2024-08-07")
	query, err := ModelNginxErrlogsQuery(ctx, "", "/generate", "2024-08-07")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(len(query))
		fmt.Println(query)
	}

	query, err = ModelNginxErrlogsQuery(ctx, "pdfparser.shenyandayi.com", "/", "2024-08-09")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(len(query))
		fmt.Println(query)
	}

}

func TestNginxLogQueryByTraceId(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	logs, err := NginxLogQueryByTraceId(ctx, "6BGsMju9j7cfC_vjTTNjl", "2024-08-09")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(logs)
	}
}

func TestBusinessLogQueryByTraceId(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	logs, err := BusinessLogQueryByTraceId(ctx, "IEqNtyf2XFmvKvo3_MsiO", "2024-08-10")
	if err != nil {
		fmt.Println(err)
	} else {
		//fmt.Println(logs)
		for _, log := range logs {
			fmt.Println(log.Time, log.Message)
		}
	}

}

func TestModelNginxLogQueryByTraceId(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	logs, err := ModelNginxLogQueryByTraceId(ctx, "66b5a873605f6c5d72519ff6", "2024-08-09")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(logs)
	}
}

func TestTracebackQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	logs, err := TracebackQuery(ctx, 0)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(logs)
	}
}
