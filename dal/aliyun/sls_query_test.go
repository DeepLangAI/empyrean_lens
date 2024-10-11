package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"fmt"
	"testing"
	"time"
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
	query, err := ModelNginxErrlogsQuery(ctx, "ai-infra-service.shenyandayi.com", "/multi-doc/single-doc-analysis", "2024-08-13")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(len(query))
		for _, log := range query {
			fmt.Println(log.CleanUrl)
		}
	}

	//query, err = ModelNginxErrlogsQuery(ctx, "pdfparser.shenyandayi.com", "/", "2024-08-09")
	//if err != nil {
	//	fmt.Println(err)
	//} else {
	//	fmt.Println(len(query))
	//	fmt.Println(query)
	//}

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
	//logs, err := BusinessLogQueryByTraceId(ctx, "IEqNtyf2XFmvKvo3_MsiO", "2024-08-10")
	logs, err := BusinessLogQueryByTraceId(ctx, "AC120BDD000F681A95153DCC5C72CF48", "2024-09-13")
	if err != nil {
		fmt.Println(err)
	} else {
		//fmt.Println(logs)
		for i, log := range logs {
			fmt.Println(i+1, log.Time, log.Message)
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

func TestParseTime(t *testing.T) {
	asctime := "2024-08-06 17:37:31.391"
	parse, err := time.Parse("2006-01-02 15:04:05.999", asctime)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(parse)
	}

	_time := "2024-08-06 17:37:31,391"
	p, err := time.Parse("2006-01-02 15:04:05.999", _time)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(p)
		fmt.Println(p.Format("2006-01-02 15:04:05.999"))
	}
}

func TestMultiNodeErrorQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	query, err := MultiNodeErrorQuery(ctx, 0)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, log := range query {
			if log.NodeName == "多文档整合" {
				fmt.Println(log)
			}
		}
	}
}

func TestExecuteTemplate(t *testing.T) {
	query := `
chat core api response error and (__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName }}-chat-go-pre) | select * from (
    select 
    regexp_extract(message, 'chat core api response error, core_name:(.*?) code:(.*?), msg:(.*?)$', 1) core_name,
    regexp_extract(message, 'chat core api response error, core_name:(.*?) code:(.*?), msg:(.*?)$', 2) code,
    regexp_extract(message, 'chat core api response error, core_name:(.*?) code:(.*?), msg:(.*?)$', 3) msg,
	asctime time, trace_id, user_id, "__tag__:_container_name_" env
    from log 
	order by time desc
	limit %v
)
`
	query = FormatWithTemplate(query, map[string]string{
		"BaseContainerName": consts.BaseContainerName,
	})
	fmt.Println(query)
}

func TestNginxLogQueryByUserId(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	logs, err := NginxLogQueryByUserId(ctx, "63e0713930c33a167f79d5d8", time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(logs)
	}
}

func TestModelNginxLogQueryByUserId(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	logs, err := ModelNginxLogQueryByUserId(ctx, "63e0713930c33a167f79d5d8", time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(logs)
	}

}

func TestBusinessLogQueryByUserId(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	logs, err := BusinessLogQueryByUserId(ctx, "63e0713930c33a167f79d5d8", time.Now().Add(-1*time.Hour), time.Now())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(logs)
	}
}

func TestNginxBizErrlogsQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	logs, err := NginxBizErrlogsQuery(ctx, "api.lingowhale.com", "/api/plugin/articles/summary", "2024-10-11")
	if err != nil {
		fmt.Println(err)
	} else {
		for i, log := range logs {
			fmt.Println(i+1, log)
		}
	}
}
