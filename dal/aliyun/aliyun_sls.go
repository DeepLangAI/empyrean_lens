package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"fmt"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"strconv"
	"strings"
	"sync"
	"time"
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
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
	SELECT 
	"content.path" url,
	"content.method" method, 
	"content.status" status, 
	"content.time" time, 
	"content.vhost" host
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
		t, e := time.Parse(time.RFC3339, log["time"])
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
	hlog.CtxInfof(ctx, "日期%v，共%v条日志", time.Unix(from, 0).Format("2006-01-02"), len(nlogs))
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
    REGEXP_REPLACE(url, '\?.*$', '') AS clean_url, time, method, status, host
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

	hlog.CtxInfof(ctx, "日期%v，共%v条日志", time.Unix(from, 0).Format("2006-01-02"), resp.Count)
	nlogs := []NginxLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse("02/Jan/2006:15:04:05", log["time"])
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

func QaCoreLogQuery(ctx context.Context, daysLookback int) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

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

	query := `__tag__:_container_name_ : lingowhale-python-prod and  message : "summary core core_name:%s" |
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
	hlog.CtxInfof(ctx, "日期%v，共%v条日志", time.Unix(from, 0).Format("2006-01-02"), resp.Count)
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

func NginxReportThisMonth(ctx context.Context) ([]NginxLog, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
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

func CoreReportThisMonth(ctx context.Context, coreName string) ([]CoreLog, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	totalDays := int(time.Since(startOfMonth).Hours() / 24)

	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	coreLogs := []CoreLog{}
	for i := 0; i <= totalDays; i++ {
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
			qaLogs, err := QaCoreLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "QaCoreLogQuery failed: %v")
				return
			}
			mutex.Lock()
			coreLogs = append(coreLogs, vlogs...)
			coreLogs = append(coreLogs, logs...)
			coreLogs = append(coreLogs, qaLogs...)
			mutex.Unlock()
		}(i)
	}
	wg.Wait()
	return coreLogs, nil

}
