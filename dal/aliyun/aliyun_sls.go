package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/cloudwego/hertz/pkg/common/hlog"
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

type NginxLog struct {
	CleanUrl string    `json:"clean_url"`
	Time     time.Time `json:"time"`
	Method   string    `json:"method"`
	Status   string    `json:"status"`
	Host     string    `json:"host"`
	Cost     float64   `json:"cost"`
}

func ModelNginxIngressBasicQuery(ctx context.Context, daysLookback int, host string) ([]NginxLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.MODEL_NGINX_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.MODEL_NGINX_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	//fromdayStr := lookbackDay.Format("2006-01-02")
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
	SELECT 
	"content.path" url,
	"content.method" method, 
	"content.status" status, 
	"content.time" time, 
	"content.vhost" host,
	"content.http_user_agent" ua
	FROM log WHERE
	"content.path" in(
		%v
	) LIMIT %v
`
	apiDetails := consts.MODEL_NGINX_INGRESS_APIS[host]
	formatedApis := []string{}
	for _, api := range apiDetails {
		formatedApis = append(formatedApis, fmt.Sprintf("'%s'", api.Api))
	}
	query = fmt.Sprintf(query, strings.Join(formatedApis, ",\n"), consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	// 查询日志
	resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	if err != nil {
		return nil, err
	}

	nlogs := []NginxLog{}
	for _, log := range resp.Logs {
		if log["host"] != host {
			continue
		}
		//fmt.Println(log)
		//if log["ua"] != "hertz" {
		//	continue
		//}
		//fmt.Println(log)
		t, e := time.Parse(time.RFC3339, log["time"])
		// 时间是UTC时间，需要+8小时
		t = t.Add(time.Hour * 8)
		//if t.Format("2006-01-02") != fromdayStr {
		//	continue
		//}
		if t.Unix() <= from {
			continue
		}
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		nlog := NginxLog{
			CleanUrl: log["url"],
			Time:     t,
			Method:   log["method"],
			Status:   log["status"],
			Host:     log["host"],
			//Cost:     cost,
		}
		nlogs = append(nlogs, nlog)
	}
	hlog.CtxInfof(ctx, "日期%v，查modelIngress，host: %v, 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), host, len(nlogs))
	return nlogs, nil
}

func NginxIngressBasicQuery(ctx context.Context, daysLookback int, host string) ([]NginxLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.NGINX_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.NGINX_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
host: %v |
SELECT  * FROM  (
  SELECT 
    REGEXP_REPLACE(url, '\?.*$', '') AS clean_url, time, method, status, host, http_referer, request_time cost
  FROM log WHERE method IN ('GET', 'POST')
) t
WHERE clean_url IN (
%s
)
LIMIT %d
`
	apiDetails := consts.NGINX_INGRESS_APIS[host]
	formatedApis := []string{}
	for _, api := range apiDetails {
		formatedApis = append(formatedApis, fmt.Sprintf("'%s'", api.Api))
	}
	query = fmt.Sprintf(query, host, strings.Join(formatedApis, ",\n"), consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	// 查询日志
	resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	hlog.CtxInfof(ctx, "日期%v，查nginxIngress，host: %v, 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), host, resp.Count)
	nlogs := []NginxLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse("02/Jan/2006:15:04:05", log["time"])
		if t.Format("2006-01-02") != lookbackDay.Format("2006-01-02") {
			continue
		}
		//if t.Unix() < from {
		//	continue
		//}
		if host == "api-repeater.lingoreader.cn" && log["clean_url"] == "/doc/multi/outline" {
			if log["http_referer"] != "https://lingowhale.com/" {
				continue
			}
		}

		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		cost, e := strconv.ParseFloat(log["cost"], 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		nlog := NginxLog{
			CleanUrl: log["clean_url"],
			Time:     t,
			Method:   log["method"],
			Status:   log["status"],
			Host:     log["host"],
			Cost:     cost,
		}
		nlogs = append(nlogs, nlog)
	}
	return nlogs, nil
}

func ModelNginxIngressLogQuery(ctx context.Context, daysLookback int) ([]NginxLog, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var nlogs []NginxLog
	for host, _ := range consts.MODEL_NGINX_INGRESS_APIS {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			nginxLogs, err := ModelNginxIngressBasicQuery(ctx, daysLookback, host)
			if err != nil {
				hlog.CtxErrorf(ctx, "query nginx log error: %v", err)
				return
			}
			mu.Lock()
			nlogs = append(nlogs, nginxLogs...)
			mu.Unlock()

		}(host)

	}
	wg.Wait()
	return nlogs, nil
}

func NginxIngressLogQuery(ctx context.Context, daysLookback int) ([]NginxLog, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var nlogs []NginxLog
	for host, _ := range consts.NGINX_INGRESS_APIS {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			nginxLogs, err := NginxIngressBasicQuery(ctx, daysLookback, host)
			if err != nil {
				hlog.CtxErrorf(ctx, "query nginx log error: %v", err)
				return
			}
			mu.Lock()
			nlogs = append(nlogs, nginxLogs...)
			mu.Unlock()

		}(host)
	}
	wg.Wait()
	return nlogs, nil
}

func MultiTotalRequestQuery(ctx context.Context, daysLookback int) (int, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)

	if err != nil {
		return 0, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()
	query := `
	__tag__:_container_name_ : lingo-python-prod and files merge multi. article_list | select * from log
limit %v
	`

	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)

	logs, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	if err != nil {
		return 0, err
	}
	cnt := len(logs.Logs)

	return cnt, nil
}

type CoreLog struct {
	CoreName string
	Node     string
	Cost     float64
	TraceId  string
	Time     time.Time
	UserId   string
}

func MultiCoreLogQuery(ctx context.Context, daysLookback int, coreName string) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
((__tag__:_container_name_: lingo-python-prod) and message: "%v core node node_name") |
select  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 1) as node_name,  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 2) as multi_id,  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 3) as entry_id,  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 4) as cost,  trace_id, user_id, asctime time from log order by time desc
limit %v
`
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	logs, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	if err != nil {
		return nil, err
	}
	coreLogs := []CoreLog{}
	for _, log := range logs.Logs {
		cost, e := strconv.ParseFloat(log["cost"], 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		t, e := time.Parse("2006-01-02 15:04:05.999", log["time"])
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		coreLog := CoreLog{
			CoreName: "多文档",
			Node:     log["node_name"],
			Cost:     cost,
			TraceId:  log["trace_id"],
			Time:     t,
			UserId:   log["user_id"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil
}

func QaMiddlewareLogQuery(ctx context.Context, daysLookback int, apis []string) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: lingo-chat-go-prod) and "Request rout" | 
select * from ( 
select  regexp_extract(message, 'Request rout:(.*), Method:POST, RequestBody:.*', 1) url,
time, trace_id, user_id
from log  
) where url in (
%v
) limit %v
`
	formatedApis := []string{}
	for _, api := range apis {
		formatedApis = append(formatedApis, fmt.Sprintf("'%s'", api))
	}
	query = fmt.Sprintf(query, strings.Join(formatedApis, ",\n"), consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "qa core sql query: %v", query)
	logs, err := logstore.GetLogs("", from, to, query, 100, 0, false)
	if err != nil {
		return nil, err
	}
	coreLogs := []CoreLog{}
	for _, log := range logs.Logs {
		t, e := time.Parse("2006-01-02 15:04:05.999", log["time"])
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		coreLog := CoreLog{
			CoreName: "问答后端",
			Node:     log["url"],
			//Cost:     cost,
			TraceId: log["trace_id"],
			Time:    t,
			UserId:  log["user_id"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil
}

func QaCoreLogQuery(ctx context.Context, daysLookback int, coreName string) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: lingo-chat-go-prod and message: "%v,") |  
select 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 1) as node, 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 2) as cnt, 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 3) as cost,trace_id,user_id, time 
from log order by time desc
limit %v
`
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "qa core sql query: %v", query)
	logs, err := logstore.GetLogs("", from, to, query, 100, 0, false)
	if err != nil {
		return nil, err
	}
	coreLogs := []CoreLog{}
	for _, log := range logs.Logs {
		cost, e := strconv.ParseFloat(log["cost"], 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		t, e := time.Parse("2006-01-02 15:04:05.999", log["time"])
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		coreLog := CoreLog{
			CoreName: "问答模型",
			Node:     log["node"],
			Cost:     cost,
			TraceId:  log["trace_id"],
			Time:     t,
			UserId:   log["user_id"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil

}

func CommonCoreLogQuery(ctx context.Context, daysLookback int, coreName string) ([]CoreLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}
	hlog.CtxInfof(ctx, "get logstore: %v success", consts.LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
__tag__:_container_name_:lingo-python-prod and "core link core_name:%v" |
SELECT 
regexp_extract(message, 'core link core_name:(.*?), core_node:(.*?), resource_id:(.*?), cost:(.*?) seconds', 1) as core_name,
regexp_extract(message, 'core link core_name:(.*?), core_node:(.*?), resource_id:(.*?), cost:(.*?) seconds', 2) as core_node,
regexp_extract(message, 'core link core_name:(.*?), core_node:(.*?), resource_id:(.*?), cost:(.*?) seconds', 4) as cost, asctime, user_id, trace_id
FROM log LIMIT %v
`

	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	coreLogs := []CoreLog{}

	for _, log := range resp.Logs {

		cost, e := strconv.ParseFloat(log["cost"], 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		t, e := time.Parse("2006-01-02 15:04:05.999", log["asctime"])
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		coreLog := CoreLog{
			CoreName: log["core_name"],
			Node:     log["core_node"],
			Cost:     cost,
			TraceId:  log["trace_id"],
			Time:     t,
			UserId:   log["user_id"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil
}

func SummreqCntQuery(ctx context.Context, daysLookback int) (map[string]int, error) {
	cnts := map[string]int{}
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
__tag__:_container_name_:lingo-python-prod and summary start | select * from (
    select 
    regexp_extract(message, 'summary start, uid:(.*), generate_type:(.*).', 1) uid, 
    regexp_extract(message, 'summary start, uid:(.*), generate_type:(.*).', 2) generate_type 
    from log limit %v
)
`

	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "summary sql query: %v", query)
	resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 打印查询结果
	hlog.CtxInfof(ctx, "日期%v，查SummaryCore，共%v条日志", time.Unix(from, 0).Format("2006-01-02"), resp.Count)
	for _, log := range resp.Logs {
		cnts[log["generate_type"]] += 1
	}
	return cnts, nil
}

func QaErrorCntQuery(ctx context.Context, daysLookback int) int64 {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)
	if err != nil {
		return 0
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: lingo-chat-go-prod and "%v") |  
select count(*) cnt from log
limit %v
`
	coreName := "模型返回异常"
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	// 打印查询结果
	hlog.CtxInfof(ctx, "日期%v，查QaError: %v, 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), coreName, resp.Count)
	for _, log := range resp.Logs {
		cnt, e := strconv.ParseInt(log["cnt"], 10, 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cnt error: %v", e)
			return 0
		}
		return cnt
	}
	return 0
}

func SummaryCoreLogQuery(ctx context.Context, daysLookback int, coreName string) ([]CoreLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `__tag__:_container_name_ : lingo-python-prod and  message : "summary core core_name:%s" |
	select
	regexp_extract(message, '^summary core core_name:(.*?),\s+node:(.*?),\s+cost:(.*?)$', 1) as core_name,
	regexp_extract(message, '^summary core core_name:(.*?),\s+node:(.*?),\s+cost:(.*?)$', 2) as node,
	regexp_extract(message, '^summary core core_name:(.*?),\s+node:(.*?),\s+cost:(.*?)$', 3) as cost,
	trace_id, asctime, user_id
	from log  order by asctime, trace_id desc limit %v
`

	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 打印查询结果
	hlog.CtxInfof(ctx, "日期%v，查SummaryCore，coreName: %v, 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), coreName, resp.Count)
	coreLogs := []CoreLog{}
	for _, log := range resp.Logs {
		cost, e := strconv.ParseFloat(log["cost"], 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		t, e := time.Parse("2006-01-02 15:04:05,999", log["asctime"])
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		clog := CoreLog{
			CoreName: log["core_name"],
			Node:     log["node"],
			Cost:     cost,
			TraceId:  log["trace_id"],
			Time:     t,
			UserId:   log["user_id"],
		}
		coreLogs = append(coreLogs, clog)
	}
	return coreLogs, nil
}

func NginxLogsToday(ctx context.Context) ([]NginxLog, error) {
	//now := time.Now()
	//startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	//endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	nginxLogs := []NginxLog{}
	logs, err := NginxIngressLogQuery(ctx, 0)
	if err != nil {
		hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
		return nil, err
	}
	mlogs, err := ModelNginxIngressLogQuery(ctx, 0)
	if err != nil {
		hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
		return nil, err
	}
	nginxLogs = append(nginxLogs, logs...)
	nginxLogs = append(nginxLogs, mlogs...)

	startingDay := time.Now()
	from := time.Date(startingDay.Year(), startingDay.Month(), startingDay.Day(), 0, 0, 0, 0, startingDay.Location()).Format("2006-01-02")
	filteredLogs := []NginxLog{}
	// 由于采集日志有时延，当天的日志可能被落在第二天
	for _, log := range nginxLogs {
		//if log.Time.Unix() >= from {
		if log.Time.Format("2006-01-02") == from {
			filteredLogs = append(filteredLogs, log)
		}
	}

	return filteredLogs, nil
}

func NginxReportOneWeek(ctx context.Context) ([]NginxLog, error) {
	//startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	//startingDay := time.Now().AddDate(0, 0, -7)
	//from := time.Date(startingDay.Year(), startingDay.Month(), startingDay.Day(), 0, 0, 0, 0, startingDay.Location()).Unix()
	//totalDays := int(time.Since(startingDay).Hours() / 24)

	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	nginxLogs := []NginxLog{}
	for i := 0; i <= 7; i++ {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			logs, err := NginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mlogs, err := ModelNginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mutex.Lock()
			nginxLogs = append(nginxLogs, logs...)
			nginxLogs = append(nginxLogs, mlogs...)
			mutex.Unlock()
		}(i)
	}
	wg.Wait()
	return nginxLogs, nil
}
func NginxReportLongTime(ctx context.Context) ([]NginxLog, error) {
	now := time.Now()
	//startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfMonth := time.Date(2024, 7, 1, 0, 0, 0, 0, now.Location())
	totalDays := int(time.Since(startOfMonth).Hours() / 24)

	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	nginxLogs := []NginxLog{}
	for i := 0; i <= totalDays; i++ {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			logs, err := NginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mlogs, err := ModelNginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mutex.Lock()
			nginxLogs = append(nginxLogs, logs...)
			nginxLogs = append(nginxLogs, mlogs...)
			mutex.Unlock()
		}(i)
	}
	wg.Wait()
	return nginxLogs, nil
}

func coreReportOfDays(ctx context.Context, coreName string, days []int) ([]CoreLog, error) {
	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	coreLogs := []CoreLog{}
	for _, day := range days {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			vlogs, err := CommonCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "SummaryCoreLogQuery failed: %v")
				return
			}
			logs, err := SummaryCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "SummaryCoreLogQuery failed: %v")
				return
			}
			qaLogs, err := QaCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "QaCoreLogQuery failed: %v")
				return
			}
			multiLogs, err := MultiCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "QaCoreLogQuery failed: %v")
				return
			}
			mutex.Lock()
			coreLogs = append(coreLogs, vlogs...)
			coreLogs = append(coreLogs, logs...)
			coreLogs = append(coreLogs, qaLogs...)
			coreLogs = append(coreLogs, multiLogs...)
			mutex.Unlock()
		}(day)
	}
	wg.Wait()
	return coreLogs, nil

}

func CoreReportToday(ctx context.Context, coreName string) ([]CoreLog, error) {
	days := []int{0}
	//for i := 0; i <= 7; i++ {
	//	days = append(days, i)
	//}
	logs, err := coreReportOfDays(ctx, coreName, days)
	return logs, err
}
func CoreReportOneWeek(ctx context.Context, coreName string) ([]CoreLog, error) {
	days := []int{}
	for i := 0; i <= 7; i++ {
		days = append(days, i)
	}
	logs, err := coreReportOfDays(ctx, coreName, days)
	return logs, err
}

func CoreReportLongTime(ctx context.Context, coreName string) ([]CoreLog, error) {
	now := time.Now()
	startOfMonth := time.Date(2024, 7, 1, 0, 0, 0, 0, now.Location())
	totalDays := int(time.Since(startOfMonth).Hours() / 24)

	days := []int{}
	for i := 0; i <= totalDays; i++ {
		days = append(days, i)
	}
	return coreReportOfDays(ctx, coreName, days)
}

type SceneOverview struct {
	Name     string
	Costs    []float64
	TotalReq int64
	FailReq  int64
	SlowReq  int64

	FailRate float64
	SlowRate float64
}

type SceneOverviews struct {
	Date                string
	AbstractOverview    SceneOverview
	OutlineOverview     SceneOverview
	ViewpointOverview   SceneOverview
	MultiOverview       SceneOverview
	QaOverview          SceneOverview
	QaRecommendOverview SceneOverview
}

func aigcCostAnlz(report SceneOverview, slowQueryThreshold int) SceneOverview {
	report.FailReq = report.TotalReq - int64(len(report.Costs))
	if report.FailReq < 0 {
		report.FailReq = 0
	}
	if report.FailReq != 0 {
		report.FailRate = float64(report.FailReq) / float64(report.TotalReq) * 100
	}

	for _, cost := range report.Costs {
		if cost > float64(slowQueryThreshold) {
			report.SlowReq++
		}
	}
	// 慢查询率，即成功的响应中，慢查询的比例
	if report.SlowReq != 0 {
		report.SlowRate = float64(report.SlowReq) / float64(report.TotalReq-report.FailReq) * 100
	}
	return report
}

func SceneGeneralOfDay(ctx context.Context, daysLookback int) (*SceneOverviews, error) {
	overviews := &SceneOverviews{}
	cnts, err := SummreqCntQuery(ctx, daysLookback)
	if err != nil {
		hlog.CtxErrorf(ctx, "err: %v", err)
		return nil, err
	}
	multiOv, err := MultiGeneralOfDay(ctx, daysLookback)
	overviews.MultiOverview = *multiOv

	abstractOverview := SceneOverview{Name: "单文档：全文速览", Costs: []float64{}, TotalReq: int64(cnts["0"]), FailReq: 0}
	outlineOverview := SceneOverview{Name: "单文档：智能大纲", Costs: []float64{}, TotalReq: int64(cnts["1"]), FailReq: 0}
	viewpointOverview := SceneOverview{Name: "单文档：关键信息", Costs: []float64{}, TotalReq: int64(cnts["3"]), FailReq: 0}
	qaOverview := SceneOverview{Name: "问答：问答", Costs: []float64{}, TotalReq: 0, FailReq: 0}
	qaRecommendOverview := SceneOverview{Name: "问答：问题推荐", Costs: []float64{}, TotalReq: 0, FailReq: 0}

	coreLogs, _ := CommonCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_VIEWPOINT)
	for _, log := range coreLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST {
			viewpointOverview.Costs = append(viewpointOverview.Costs, log.Cost)
		}
	}
	abstractLogs, err := SummaryCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_ABSTRACT)
	for _, log := range abstractLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_ABSTRACT_ETE_COST {
			abstractOverview.Costs = append(abstractOverview.Costs, log.Cost)
		}
	}

	outlineLogs, err := SummaryCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_OUTLINE)
	for _, log := range outlineLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST {
			outlineOverview.Costs = append(outlineOverview.Costs, log.Cost)
		}
	}

	qaLogs, err := NginxIngressBasicQuery(ctx, daysLookback, consts.HOST_QA_BACKEND)
	for _, log := range qaLogs {
		if log.CleanUrl == "/api/chat/qa" {
			qaOverview.TotalReq += 1
			if log.Status == "200" {
				qaOverview.Costs = append(qaOverview.Costs, log.Cost)
			}
		} else if log.CleanUrl == "/api/chat/recommend" {
			qaRecommendOverview.TotalReq += 1
			if log.Status == "200" {
				qaRecommendOverview.Costs = append(qaRecommendOverview.Costs, log.Cost)
			}
		}
	}

	abstractOverview = aigcCostAnlz(abstractOverview, consts.SLOWQUERY_THRESHOLD_ABSTRACT)
	outlineOverview = aigcCostAnlz(outlineOverview, consts.SLOWQUERY_THRESHOLD_OUTLINE)
	viewpointOverview = aigcCostAnlz(viewpointOverview, consts.SLOWQUERY_THRESHOLD_VIEWPOINT)
	qaOverview = aigcCostAnlz(qaOverview, consts.SLOWQUERY_THRESHOLD_QA)
	qaRecommendOverview = aigcCostAnlz(qaRecommendOverview, consts.SLOWQUERY_THRESHOLD_QA_RECOMMEND)

	overviews.AbstractOverview = abstractOverview
	overviews.OutlineOverview = outlineOverview
	overviews.ViewpointOverview = viewpointOverview
	overviews.QaOverview = qaOverview
	overviews.QaRecommendOverview = qaRecommendOverview

	overviews.Date = time.Now().AddDate(0, 0, -daysLookback).Format("2006-01-02")

	return overviews, nil
}

func MultiGeneralOfDay(ctx context.Context, daysLookback int) (*SceneOverview, error) {
	ov := &SceneOverview{}
	ov.Name = "多文档：总结"
	total, err := MultiTotalRequestQuery(ctx, daysLookback)
	if err != nil {
		return ov, err
	}
	ov.TotalReq = int64(total)
	logs, err := MultiCoreLogQuery(ctx, daysLookback, "multi")
	for _, log := range logs {
		if log.Node == consts.ALIYUN_LOG_NODE_MULTI_ALL_SUCCESS {
			ov.Costs = append(ov.Costs, log.Cost)
		}
	}
	anlz := aigcCostAnlz(*ov, consts.SLOWQUERY_THRESHOLD_MULTIDOC)
	return &anlz, nil
}

func SummaryGeneralOverview(ctx context.Context, days []int) []SceneOverviews {
	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	overviews := []SceneOverviews{}
	for _, day := range days {
		wg.Add(1)
		go func(lookbackDay int) {
			defer wg.Done()
			ov, err := SceneGeneralOfDay(ctx, lookbackDay)
			if err != nil {
				hlog.CtxErrorf(ctx, "err: %v", err)
				return
			}
			mutex.Lock()
			overviews = append(overviews, *ov)
			mutex.Unlock()
		}(day)
	}
	wg.Wait()
	sort.Slice(overviews, func(i, j int) bool {
		return overviews[i].Date > overviews[j].Date
	})
	return overviews
}
