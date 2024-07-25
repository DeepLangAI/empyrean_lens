package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"fmt"
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
    REGEXP_REPLACE(url, '\?.*$', '') AS clean_url, time, method, status, host, http_referer
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
		nlog := NginxLog{
			CleanUrl: log["clean_url"],
			Time:     t,
			Method:   log["method"],
			Status:   log["status"],
			Host:     log["host"],
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
"core link core_name:%v" |
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
