package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"fmt"
	"strings"
	"testing"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
)

func TestQueryLog(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//QueryLog(ctx, 0)
}

func TestQueryLog2(t *testing.T) {
	//client := sls.CreateNormalInterface(consts.ENDPOINT, accessKeyID, accessKeySecret, "")
	client := sls.CreateNormalInterface(consts.ENDPOINT, consts.ACCESS_KEY_ID, consts.ACCESS_KEY_SECRET, "")

	//logstore, err := client.GetLogStore(consts.proj, logStoreName)
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)
	if err != nil {
		panic(err)
	}
	fmt.Println("Get logstore successfully:", logstore.Name)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	from := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location()).Unix()
	to := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location()).Unix()

	// 查询日志
	resp, err := logstore.GetLogs("", from, to, "(__tag__:_container_name_ : lingowhale-python-prod and message : \"summary core core_name:大纲\"  )|select message,asctime ", 100000, 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 打印查询结果
	for _, log := range resp.Logs {
		fmt.Println(log)
	}
}

func TestNginxIngressLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := NginxIngressLogQuery(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			if log.CleanUrl != "/edu_parse" {
				continue
			}
			fmt.Println(log)
		}
	}
}
func TestNginxLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs := []NginxLog{}
	nlogs, _ := NginxIngressLogQuery(ctx, 0)
	mlogs, _ := ModelNginxIngressLogQuery(ctx, 0)
	logs = append(logs, nlogs...)
	logs = append(logs, mlogs...)
	fmt.Println(len(logs))
}

func TestOutlineLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//logs, e := SummaryCoreLogQuery(ctx, 0, consts.CORE_NAME_OUTLINE)
	logs, e := SummaryCoreLogQuery(ctx, 1, consts.CORE_NAME_ABSTRACT)
	if e != nil {
		t.Error(e)
	} else {
		//cnt := 0
		for _, log := range logs {
			fmt.Println(log)
			//if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST || log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST {
			//	cnt += 1
			//	fmt.Println(cnt, log)
			//}
		}
	}
}
func TestOutlineLogQueryViewpoint(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//logs, e := SummaryCoreLogQuery(ctx, 0, consts.CORE_NAME_OUTLINE)
	logs, e := SummaryCoreLogQuery(ctx, 1, consts.CORE_NAME_VIEWPOINT)
	if e != nil {
		t.Error(e)
	} else {
		cnt := 0
		for _, log := range logs {
			fmt.Println(cnt, log)
			if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST {
				cnt += 1
				fmt.Println(cnt, log)
			}
		}
	}
}

func TestNginxReportLongTime(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	NginxReportLongTime(ctx)
}

func TestCoreLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	coreLogs, err := SummaryCoreLogQuery(ctx, 0, consts.CORE_NAME_OUTLINE)
	if err != nil {
		t.Error(err)
	}
	nodes := []string{
		consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST,
		consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST,
	}
	for _, log := range coreLogs {
		if utils.Contains(nodes, log.Node) {
			fmt.Println(log)
		}
	}

}

func TestCoreReportLongTime(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, _ := CoreReportLongTime(ctx, consts.CORE_NAME_OUTLINE)
	for _, l := range logs {
		fmt.Println(l)
	}
}

func TestStatusCodeUpdate(t *testing.T) {
	s := "300,200"
	codes := strings.Split(s, ",")
	if !utils.Contains(codes, "400") {
		codes = append(codes, "400")
		s = strings.Join(codes, ",")
	}
	fmt.Println(s)
}

func TestNginxIngressBasicQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := NginxIngressBasicQuery(ctx, 0, consts.HOST_QA_BACKEND)
	if err != nil {
		t.Error(err)
	} else {
		cnt := 0
		for _, log := range logs {
			fmt.Println(cnt, log)
			cnt += 1
			//if log.CleanUrl == "/api/plugin/articles/summary" {
			//	cnt += 1
			//	fmt.Println(cnt, log)
			//}
		}
	}
}

func TestModelNginxIngressBasicQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	if logs, err := ModelNginxIngressBasicQuery(ctx, 0, consts.HOST_MULTI_MODEL); err != nil {
		t.Error(err)
	} else {
		for i, log := range logs {
			fmt.Println(i, log)
		}
	}
}

func TestMetrLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)

	if err != nil {
		t.Error(err)
	}

	daysLookback := 1
	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: lingo-chat-go-prod and message: "问答模型,") |  
select 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 1) as node, 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 2) as cnt, 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 3) as cost,trace_id,user_id, time 
from log order by time desc
`
	logs, err := logstore.GetLogs("", from, to, query, 100, 0, false)
	if err != nil {
		t.Error(err)
	}
	for _, log := range logs.Logs {
		fmt.Println(log)
	}
}

func TestQaCoreLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := QaCoreLogQuery(ctx, 0, consts.HOST_REPEATER)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}

}

func TestModelNginxIngressLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := ModelNginxIngressLogQuery(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}

}

func TestCommonCoreLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := CommonCoreLogQuery(ctx, 1, consts.CORE_NAME_VIEWPOINT)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}
}

func TestMultiCoreLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := MultiCoreLogQuery(ctx, 0, "multi")
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}
}

func Test_coreReportOfDays(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//logs, err := coreReportOfDays(ctx, "多文档端到端", []int{0, 1, 2})
	//logs, err := coreReportOfDays(ctx, "问答模型", []int{1})
	logs, err := coreReportOfDays(ctx, consts.CORE_NAME_ABSTRACT, []int{1})
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}

}

func TestNginxLogsToday(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := NginxLogsToday(ctx)
	if err != nil {
		t.Error(err)
	} else {
		for i, log := range logs {
			fmt.Println(i+1, log)
		}
	}

}

func TestSummreqCntQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	ov := SummaryGeneralOverview(ctx, []int{1})
	for _, o := range ov {
		fmt.Println(o.AbstractOverview.TotalReq, o.AbstractOverview.FailReq)
		fmt.Println(o.OutlineOverview.TotalReq, o.OutlineOverview.FailReq)
		fmt.Println(o.ViewpointOverview.TotalReq, o.ViewpointOverview.FailReq)
	}
}

func TestSummaryGeneralOfDay(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	ov, _ := SceneGeneralOfDay(ctx, 0)
	fmt.Println(ov.AbstractOverview.TotalReq, ov.AbstractOverview.FailReq)
	fmt.Println(ov.MultiOverview.TotalReq, ov.MultiOverview.FailReq)
	fmt.Println(ov.QaOverview.TotalReq, ov.QaOverview.FailReq)
	fmt.Println(ov.QaRecommendOverview.TotalReq, ov.QaRecommendOverview.FailReq)
	fmt.Println(ov.MultiAnalysisOverview.TotalReq, ov.MultiAnalysisOverview.FailReq)
	fmt.Println(ov.MultiMergeOverview.TotalReq, ov.MultiMergeOverview.FailReq)
}

func TestMultidocGeneralOverview(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
}

func TestMultiTotalRequestQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	ov_ete, ov_analysis, ov_merge, err := MultiGeneralOfDay(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(ov_ete.TotalReq, ov_ete.FailReq, ov_ete.SlowReq, ov_ete.FailRate, ov_ete.SlowRate)
		fmt.Println(ov_analysis.TotalReq, ov_analysis.FailReq, ov_analysis.SlowReq, ov_analysis.FailRate, ov_analysis.SlowRate)
		fmt.Println(ov_merge.TotalReq, ov_merge.FailReq, ov_merge.SlowReq, ov_merge.FailRate, ov_merge.SlowRate)
	}
}

func TestQaMiddlewareLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	apis := []string{
		"/api/chat/qa",
		"/api/chat/recommend",
	}
	logs, err := QaMiddlewareLogQuery(ctx, 1, apis)
	counts := map[string]int{}
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
			counts[log.Node] += 1
		}
	}
	fmt.Println(counts)
}

func TestQaErrorCntQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	cnt := QaErrorCntQuery(ctx, 0)
	fmt.Println(cnt)
}

func TestMultiNodeLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := MultiNodeLogQuery(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}
}
