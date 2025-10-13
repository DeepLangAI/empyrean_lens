package aliyun

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal/redis"
	"empyrean_lens/utils"
	"fmt"
	"sort"
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
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
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
		for i, log := range logs {
			//if log.CleanUrl != "/edu_parse" {
			//	continue
			//}
			fmt.Println(i, log)
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
	logs, e := SummaryCoreLogQuery(ctx, 0, consts.CORE_NAME_OUTLINE)
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
	logs, err := NginxIngressBasicQuery(ctx, 1, consts.HOST_WCD)
	if err != nil {
		t.Error(err)
	} else {
		cnt := 0
		for _, log := range logs {
			fmt.Println(cnt, log)
			cnt += 1
			//if log.BizCode != 0 {
			//	cnt += 1
			//	fmt.Println(cnt, log)
			//}
		}
	}
}

func TestModelNginxIngressBasicQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//if logs, err := ModelNginxIngressBasicQuery(ctx, 0, consts.HOST_ABSTRACT); err != nil {
	if logs, err := ModelNginxIngressBasicQuery(ctx, 0, consts.HOST_ABSTRACT); err != nil {
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

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

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
	ov := SceneGeneralOverview(ctx, []int{1})
	for _, o := range ov {
		//fmt.Println(o.AbstractOverview.TotalReq, o.AbstractOverview.FailReq)
		//fmt.Println(o.OutlineOverview.TotalReq, o.OutlineOverview.FailReq)
		//fmt.Println(o.ViewpointOverview.TotalReq, o.ViewpointOverview.FailReq)
		for _, x := range o.Overviews {
			if len(x.SlowDetails) > 0 {
				fmt.Println("==============================", len(x.SlowDetails), x.SlowDetails)
			}
		}
		//fmt.Println(o.Overviews)
	}
}

func TestSummaryGeneralOfDay(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	ov, _ := SceneGeneralOfDay(ctx, 0)
	for _, o := range ov.Overviews {
		//if len(o.SlowDetails) > 0 {
		//	fmt.Println(o.Name, o.TotalReq, o.FailReq, o.SceneSlowReq, o.SlowDetails)
		//}
		fmt.Println(o.Name, o.TotalReq, o.FailReq, o.SlowReq, o.FailReason)
	}
	//fmt.Println(ov.AbstractOverview.TotalReq, ov.AbstractOverview.FailReq)
	//fmt.Println(ov.OutlineOverview.TotalReq, ov.OutlineOverview.FailReq)
	//fmt.Println(ov.ViewpointOverview.TotalReq, ov.ViewpointOverview.FailReq)
	//fmt.Println(ov.MultiOverview.TotalReq, ov.MultiOverview.FailReq)
	//fmt.Println(ov.QaOverview.TotalReq, ov.QaOverview.FailReq)
	//fmt.Println(ov.QaRecommendOverview.TotalReq, ov.QaRecommendOverview.FailReq)
	//fmt.Println(ov.MultiAnalysisOverview.TotalReq, ov.MultiAnalysisOverview.FailReq)
	//fmt.Println(ov.MultiMergeOverview.TotalReq, ov.MultiMergeOverview.FailReq)
}

func TestMultidocGeneralOverview(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
}

func TestMultiTotalRequestQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	ov, err := MultiGeneralOfDay(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(ov.MultiMergeOverview.FailDetails)
		//fmt.Println(ov.MultiEteOverview.SlowDetails)
		//fmt.Println(ov_ete.TotalReq, ov_ete.FailReq, ov_ete.SceneSlowReq, ov_ete.FailRate, ov_ete.SlowRate)
		//fmt.Println(ov_analysis.TotalReq, ov_analysis.FailReq, ov_analysis.SceneSlowReq, ov_analysis.FailRate, ov_analysis.SlowRate)
		//fmt.Println(ov_merge.TotalReq, ov_merge.FailReq, ov_merge.SceneSlowReq, ov_merge.FailRate, ov_merge.SlowRate)
	}
}

func TestCollectGeneralOfDay(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	ov, err := CollectGeneralOfDay(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println(ov[0])
	}
}

func TestQaMiddlewareLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	apis := []string{
		"/api/chat/qa",
		"/api/chat/recommend",
	}
	logs, err := QaMiddlewareReqLogQuery(ctx, 1, apis)
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

func TestQaMiddlewareRespLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	urls := []string{
		"/api/chat/qa",
		"/api/chat/recommend",
	}
	logs, err := QaMiddlewareRespLogQuery(ctx, 0, urls)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}

}

func TestLingoCoreErrorLogs(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := LingoCoreErrorLogs(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}
}

func TestSummreqCntQuery1(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	query, err := SummreqCntQuery(ctx, 0)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(query)
}

func TestNginxLogsOfAPI_Business(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//api, err := NginxErrorLogsOfAPI(ctx, "api.lingoreader.cn", "/api/plugin/articles/summary", "2024-08-09")
	api, err := NginxErrorLogsOfAPI(ctx, "", "/api/plugin/articles/summary", "2024-08-09")
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("日志数：", len(api))
		for _, log := range api {
			fmt.Println(log)
		}
	}
}

func TestNginxLogsOfAPI_Model(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//api, err := NginxErrorLogsOfAPI(ctx, "pdfparser.shenyandayi.com", "/", "2024-08-09")
	api, err := NginxErrorLogsOfAPI(ctx, "api.lingowhale.com", "/api/plugin/articles/summary/list_v2", "2025-01-02")
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("日志数：", len(api))
		for _, log := range api {
			fmt.Println(log)
		}
	}
}

func TestEndToEndLogsQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	//logs, err := EndToEndLogsQuery(ctx, "6BGsMju9j7cfC_vjTTNjl", "2024-08-09")
	//logs, err := EndToEndLogsQuery(ctx, "IEqNtyf2XFmvKvo3_MsiO", "2024-08-10")
	logs, err := EndToEndLogsQuery(ctx, "66cdc5dace30fa9ead513fe0", "2024-08-27")

	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("日志数：", len(logs))
		for _, log := range logs {
			fmt.Println(log.OriginLog)
		}
	}
}

func TestTracebackQueryOfTimespan(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	timespan := TracebackQueryOfTimespan(ctx, consts.TIMESPAN_WEEK)
	fmt.Println(timespan)
}

func TestTracebackQueryOfDays(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs := TracebackQueryOfDays(ctx, []int{1})
	fmt.Println(logs)
}

func TestNginxLogsDaysAgo(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	Init(ctx)
	redis.Init()

	logs, _ := NginxLogsDaysAgo(ctx, 1)
	cnts := map[string]int{}
	for _, log := range logs {
		tick := log.Time.Format(consts.DateHourMinuteTemplate)
		cnts[tick] += 1
	}
	keys := utils.KeysOfMap(cnts)
	for _, key := range keys {
		fmt.Println(key, cnts[key])
	}
}

type SceneLogs struct {
	Scene string
	Logs  []EndToEndLog
}

func TestEndToEndUserLogsQuery(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	Init(ctx)

	results := []SceneLogs{}
	groupedPaths := map[string][]string{
		"数据处理": {
			"/crawl",
			"/wcd-raw",
			"/edu_parse",
			"/api/plugin/file/add",
			"/api/readers/url/upload",
			"/api/readers/url/content/upload",
		},
		"摘录": {
			"/api/plugin/extract/detail",
		},
		"单文档": {
			"/api/repeater/abstract",
			"/api/repeater/outline",
			"/api/repeater/viewpoint",
			"/api/plugin/articles/summary",
		},
		"问答": {
			"/qa/main",
			"/qa/query_recommend",
		},
		"多文档": {
			"/doc/single/analyze",
			"/doc/multi/analyze",
			"/doc/multi/outline",
			"/multi-doc/single-doc-analysis",
			"/multi-doc/doc-merge",
			"/multi-doc/doc-summary",
		},
		"订阅": {
			"/api/feed/v1/subscription/upsert",
		},
	}
	groupedLogs := map[string][]EndToEndLog{}
	logs, err := EndToEndUserLogsQuery(ctx, "63e0713930c33a167f79d5d8", time.Now().Add(-1*time.Hour).Format(consts.DateHourMinuteTemplate), time.Now().Format(consts.DateHourMinuteTemplate))
	if err != nil {
		t.Error(err)
	} else {
		fmt.Println("日志数共有：", len(logs))
		for _, log := range logs {
			key := ""
			for _key, paths := range groupedPaths {
				if utils.Contains(paths, log.ApiPath) {
					key = _key
					break
				}
			}
			if key == "" {
				if strings.Contains(log.ApiPath, "safety") {
					key = "安全"
				}
			}
			if key == "" {
				key = "待分类"
			}
			if groupedLogs[key] == nil {
				groupedLogs[key] = []EndToEndLog{}
			}
			groupedLogs[key] = append(groupedLogs[key], log)
		}
	}
	keys := utils.KeysOfMap(groupedLogs)
	orderedKeys := []string{
		"单文档",
		"摘录",
		"问答",
		"多文档",
		"数据处理",
		"订阅",
		"安全",
		"其他",
	}
	sort.Slice(keys, func(i, j int) bool {
		idxI := utils.Index(orderedKeys, keys[i])
		idxJ := utils.Index(orderedKeys, keys[j])
		if idxI != -1 && idxJ != -1 {
			return idxI < idxJ
		}
		if idxI == -1 && idxJ == -1 {
			return keys[i] < keys[j]
		}
		return idxI == -1
	})
	for _, key := range keys {
		sort.Slice(groupedLogs[key], func(i, j int) bool {
			return groupedLogs[key][i].Time >= groupedLogs[key][j].Time
		})
		results = append(results, SceneLogs{
			Scene: key,
			Logs:  groupedLogs[key],
		})
		fmt.Println("场景：", key, "日志数：", len(groupedLogs[key]))
	}
}
