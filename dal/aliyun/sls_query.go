package aliyun

import (
	"bytes"
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type NginxLog struct {
	CleanUrl string    `json:"clean_url"`
	Time     time.Time `json:"time"`
	Method   string    `json:"method"`
	Status   string    `json:"status"`
	Host     string    `json:"host"`
	Cost     float64   `json:"cost"`
}

type NginxErrorLog struct {
	NginxLog
	UserId   string `json:"user_id"`
	TraceId  string `json:"trace_id"`
	ClientIp string `json:"client_ip"`
}

func QueryLogsWithRetry(ctx context.Context, logstore *sls.LogStore, from, to int64, query string) (*sls.GetLogsResponse, error) {
	var err error
	var resp *sls.GetLogsResponse
	for i := 0; i < consts.LOG_QUERY_RETRY_TIMES; i++ {
		resp, err = logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
		if err != nil { // retry
			hlog.CtxErrorf(ctx, "%v, query log failed, retrying: %v", i+1, err)
			time.Sleep(time.Duration(i) * time.Second)
			continue
		}
		err = nil
		break
	}
	return resp, err
}

func FormatWithTemplate(tplStr string, data map[string]string) string {
	baseData := map[string]string{
		"BaseContainerName": consts.BaseContainerName,
		"BaseChannelName":   consts.BaseChannelName,
	}
	if data != nil {
		for k, v := range data {
			baseData[k] = v
		}
	}

	tpl, err := template.New("").Parse(tplStr)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = tpl.Execute(&buf, baseData)
	if err != nil {
		panic(err)
	}
	result := buf.String()
	return result
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
("content.channel": "{{.BaseChannelName}}-prod" or "content.channel": "{{.BaseChannelName}}-pre")|
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
	) and "content.vhost" = '%v'
	order by "content.time"
	LIMIT %v
`
	query = FormatWithTemplate(query, nil)
	apiDetails := consts.MODEL_NGINX_INGRESS_APIS[host]
	formatedApis := []string{}
	for _, api := range apiDetails {
		formatedApis = append(formatedApis, fmt.Sprintf("'%s'", api.Api))
	}
	query = fmt.Sprintf(query, strings.Join(formatedApis, ",\n"), host, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	// 查询日志
	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)
	if err != nil {
		hlog.CtxErrorf(ctx, "ModelNginxIngressBasicQuery get logs error: %v", err)
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
			hlog.CtxDebugf(ctx, "time %v <= from %v", t, time.Unix(from, 0))
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
    REGEXP_REPLACE(url, '\?.*$', '') AS clean_url, time, method, status, host, http_referer, request_time cost,channel
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
	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "NginxIngressBasicQuery query log error: %v", err)
		return nil, err
	}

	shouldHaveChannel := utils.Contains([]string{
		consts.HOST_CRAWLER, consts.HOST_WCD, consts.HOST_EDU,
		consts.HOST_PRE_CRAWLER, consts.HOST_PRE_WCD, consts.HOST_PRE_EDU,
	}, host)
	hlog.CtxInfof(ctx, "日期%v，查nginxIngress，host: %v, 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), host, resp.Count)
	nlogs := []NginxLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse("02/Jan/2006:15:04:05", log["time"])
		if t.Format("2006-01-02") != lookbackDay.Format("2006-01-02") {
			continue
		}
		channel := log["channel"]
		// 如果channel存在，且不是目标channel，则跳过
		if (shouldHaveChannel || !utils.Contains([]string{"-", "", "null"}, channel)) && !utils.Contains(
			[]string{consts.BaseChannelName + "-pre", consts.BaseChannelName + "-prod"},
			channel,
		) {
			continue
		}
		//if host == "api-repeater.lingoreader.cn" && log["clean_url"] == "/doc/multi/outline" {
		//	if log["http_referer"] != "https://lingowhale.com/" {
		//		continue
		//	}
		//}

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

func MultiTotalRequestQuery(ctx context.Context, daysLookback int) (int, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return 0, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()
	query := `
	(__tag__:_container_name_ : {{.BaseContainerName}}-python-prod or __tag__:_container_name_ : {{.BaseContainerName}}-python-pre) and files merge multi. article_list | select * from log
limit %v
	`
	query = FormatWithTemplate(query, nil)

	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)

	//logs, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	logs, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "MultiTotalRequestQuery query log error: %v", err)
		return 0, err
	}
	cnt := len(logs.Logs)

	return cnt, nil
}

func MultiNodeLogQuery(ctx context.Context, daysLookback int) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_ : {{.BaseContainerName}}-python-prod or __tag__:_container_name_ : {{.BaseContainerName}}-python-pre) and multi_node | select * from (
    select regexp_extract(message, 'multi_node (.*?)(\.|,|\s)', 1) node_name, asctime time, user_id, trace_id
    from log
) order by time desc limit %v
`
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	hlog.CtxInfof(ctx, "date: %v, 查多文档节点: %v", lookbackDay.Format(consts.DateTemplate), query)
	//logs, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	logs, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "MultiNodeLogQuery query log error: %v", err)
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
			CoreName: "多文档",
			Node:     log["node_name"],
			//Cost:     cost,
			TraceId: log["trace_id"],
			Time:    t,
			UserId:  log["user_id"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil

}

type CoreLog struct {
	CoreName string
	Node     string
	Cost     float64
	TraceId  string
	Time     time.Time
	UserId   string
	Status   int
	Env      string
}

func MultiCoreLogQuery(ctx context.Context, daysLookback int, coreName string) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
((__tag__:_container_name_: {{.BaseContainerName}}-python-prod or __tag__:_container_name_: {{.BaseContainerName}}-python-pre) and message: "%v core node node_name") |
select  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 1) as node_name,  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 2) as multi_id,  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 3) as entry_id,  
regexp_extract(message, 'multi core node node_name:(.*),\s+multi_id:(.*),\s+entry_id:(.*),\s+cost:(.*) seconds', 4) as cost,  
trace_id, user_id, asctime time, "__tag__:_container_name_" env
from log order by time desc
limit %v
`
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	//logs, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	logs, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "MultiCoreLogQuery query log error: %v", err)
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
			Env:      log["env"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil
}

func QaMiddlewareReqLogQuery(ctx context.Context, daysLookback int, apis []string) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName}}-chat-go-pre) and "Request rout" | 
select * from ( 
select  regexp_extract(message, 'Request rout:(.*), Method:POST, RequestBody:.*', 1) url,
time, trace_id, user_id
from log  
) where url in (
%v
) limit %v
`
	query = FormatWithTemplate(query, nil)
	formatedApis := []string{}
	for _, api := range apis {
		formatedApis = append(formatedApis, fmt.Sprintf("'%s'", api))
	}
	query = fmt.Sprintf(query, strings.Join(formatedApis, ",\n"), consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "qa core sql query: %v", query)
	//logs, err := logstore.GetLogs("", from, to, query, 100, 0, false)
	logs, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "QaMiddlewareReqLogQuery query log error: %v", err)
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
func QaMiddlewareRespLogQuery(ctx context.Context, daysLookback int, apis []string) ([]NginxLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName}}-chat-go-pre) and "Response rout" | 
select * from ( 
select  
regexp_extract(message, 'Response rout:(.*), code:(.*), cost:(.*) s', 1) url,
regexp_extract(message, 'Response rout:(.*), code:(.*), cost:(.*) s', 2) status,
regexp_extract(message, 'Response rout:(.*), code:(.*), cost:(.*) s', 3) cost,
time, trace_id, user_id, "__tag__:_container_name_" env
from log  
) where url in (
%v
) 
order by time desc
limit %v
`
	query = FormatWithTemplate(query, nil)
	formatedApis := []string{}
	for _, api := range apis {
		formatedApis = append(formatedApis, fmt.Sprintf("'%s'", api))
	}
	query = fmt.Sprintf(query, strings.Join(formatedApis, ",\n"), consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "qa core sql query: %v", query)
	//logs, err := logstore.GetLogs("", from, to, query, 100, 0, false)
	logs, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "QaMiddlewareRespLogQuery query log error: %v", err)
		return nil, err
	}
	coreLogs := []NginxLog{}
	for _, log := range logs.Logs {
		t, e := time.Parse("2006-01-02 15:04:05.999", log["time"])
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		cost, e := strconv.ParseFloat(log["cost"], 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		coreLog := NginxLog{
			CleanUrl: log["url"],
			Time:     t,
			Method:   "POST",
			Status:   log["status"],
			Cost:     cost,
			//UserId:   log["user_id"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil
}

func QaCoreLogQuery(ctx context.Context, daysLookback int, coreName string) ([]CoreLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName}}-chat-go-pre) and message: "%v," |  select * from (
select 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 1) as node, 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 2) as cnt, 
regexp_extract(message, '问答模型, (.*),\s+count:(.*),\s+cost:(.*)\s+s$', 3) as cost,trace_id,user_id, time 
from log order by time desc
limit %v
) where node != 'null'
`
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "qa core sql query: %v", query)
	//logs, err := logstore.GetLogs("", from, to, query, 100, 0, false)
	logs, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "QaCoreLogQuery query log error: %v", err)
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
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}
	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_:{{.BaseContainerName}}-python-prod or __tag__:_container_name_:{{.BaseContainerName}}-python-pre) and "core link core_name:%v" |
SELECT 
regexp_extract(message, 'core link core_name:(.*?), core_node:(.*?), resource_id:(.*?), cost:(.*?) seconds', 1) as core_name,
regexp_extract(message, 'core link core_name:(.*?), core_node:(.*?), resource_id:(.*?), cost:(.*?) seconds', 2) as core_node,
regexp_extract(message, 'core link core_name:(.*?), core_node:(.*?), resource_id:(.*?), cost:(.*?) seconds', 4) as cost, asctime, user_id, trace_id, "__tag__:_container_name_" env
FROM log 
order by asctime desc
LIMIT %v
`

	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "CommonCoreLogQuery query log error: %v", err)
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
			Env:      log["env"],
		}
		coreLogs = append(coreLogs, coreLog)
	}
	return coreLogs, nil
}

type CoreErrorLogs struct {
	CoreName string
	Code     int64
	Msg      string
	Time     time.Time
	TraceId  string
	UserId   string
	Env      string
	NodeName string
}

func LingoChatCoreErrorLogs(ctx context.Context, daysLookback int) ([]CoreErrorLogs, error) {
	logs := []CoreErrorLogs{}

	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
chat core api response error and (__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName}}-chat-go-pre) | select * from (
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
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "chat core error sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "LingoChatCoreErrorLogs query log error: %v", err)
		return nil, err
	}

	// 打印查询结果
	hlog.CtxInfof(ctx, "日期%v，查chat core error 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), resp.Count)
	for _, log := range resp.Logs {
		code, err := strconv.ParseInt(log["code"], 10, 64)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse code to int error: %v", err)
			continue
		}
		t, err := time.Parse("2006-01-02 15:04:05,999", log["time"])
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		logs = append(logs, CoreErrorLogs{
			Code:     code,
			Msg:      log["msg"],
			CoreName: log["core_name"],
			Time:     t,
			UserId:   log["user_id"],
			TraceId:  log["trace_id"],
			Env:      log["env"],
		})
	}
	return logs, nil

}

func LingoCoreErrorLogs(ctx context.Context, daysLookback int) ([]CoreErrorLogs, error) {
	logs := []CoreErrorLogs{}

	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_:{{.BaseContainerName}}-python-prod or __tag__:_container_name_:{{.BaseContainerName}}-python-pre) and extend core api response error core_name| select * from (
    select 
    regexp_extract(message, 'extend core api response error, core_name:(.*?) code:(.*?), msg:(.*?)$', 1) core_name, 
    regexp_extract(message, 'extend core api response error, core_name:(.*?) code:(.*?), msg:(.*?)$', 2) code, 
    regexp_extract(message, 'extend core api response error, core_name:(.*?) code:(.*?), msg:(.*?)$', 3) msg ,
	asctime time, trace_id, user_id, "__tag__:_container_name_" env
    from log order by time desc limit %v
)
`
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "core error sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "LingoCoreErrorLogs query log error: %v", err)
		return nil, err
	}

	// 打印查询结果
	hlog.CtxInfof(ctx, "日期%v，查core error 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), resp.Count)
	for _, log := range resp.Logs {
		code, err := strconv.ParseInt(log["code"], 10, 64)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse code to int error: %v", err)
			continue
		}
		// 解析如2024-07-31 14:31:13,916的时间
		t, err := time.Parse("2006-01-02 15:04:05,999", log["time"])
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		logs = append(logs, CoreErrorLogs{
			Code:     code,
			Msg:      log["msg"],
			CoreName: log["core_name"],
			Time:     t,
			UserId:   log["user_id"],
			TraceId:  log["trace_id"],
			Env:      log["env"],
		})
	}
	return logs, nil
}

func SummreqCntQuery(ctx context.Context, daysLookback int) (map[string]int, error) {
	cnts := map[string]int{}
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_:{{.BaseContainerName}}-python-prod or __tag__:_container_name_:{{.BaseContainerName}}-python-pre) and summary start | select * from (
    select 
    regexp_extract(message, 'summary start, file_id:(.*?), url_id:(.*?) generate_type:(.*?)\.$', 1) file_id, 
    regexp_extract(message, 'summary start, file_id:(.*?), url_id:(.*?) generate_type:(.*?)\.$', 2) url_id, 
    regexp_extract(message, 'summary start, file_id:(.*?), url_id:(.*?) generate_type:(.*?)\.$', 3) generate_type,
    trace_id, user_id, asctime time
    from log order by time desc limit %v
)
`
	query = FormatWithTemplate(query, nil)

	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "summary sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "SummreqCntQuery query log error: %v", err)
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
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return 0
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
(__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName}}-chat-go-pre) and "%v" |  
select count(*) cnt from log
limit %v
`
	query = FormatWithTemplate(query, nil)
	coreName := "模型返回异常"
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "QaErrorCntQuery query log error: %v", err)
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
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `(__tag__:_container_name_ : {{.BaseContainerName}}-python-prod or __tag__:_container_name_ : {{.BaseContainerName}}-python-pre) and  message : "summary core core_name:%s" |
	select
	regexp_extract(message, '^summary core core_name:(.*?),\s+node:(.*?),\s+cost:(.*?)$', 1) as core_name,
	regexp_extract(message, '^summary core core_name:(.*?),\s+node:(.*?),\s+cost:(.*?)$', 2) as node,
	regexp_extract(message, '^summary core core_name:(.*?),\s+node:(.*?),\s+cost:(.*?)$', 3) as cost,
	trace_id, asctime, user_id, "__tag__:_container_name_" env
	from log  order by asctime desc , trace_id desc limit %v
`

	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, coreName, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "SummaryCoreLogQuery query log error: %v", err)
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
			Env:      log["env"],
		}
		coreLogs = append(coreLogs, clog)
	}
	return coreLogs, nil
}

func QaRecommendFailcntQuery(ctx context.Context, daysLookback int) int64 {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return 0
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
推荐模型返回异常 and (__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName}}-chat-go-pre) | select count(*) cnt from log
limit %v
`

	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "问题推荐失败数量查询 query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "QaRecommendFailcntQuery query log error: %v", err)
		return 0
	}

	// 打印查询结果
	hlog.CtxInfof(ctx, "日期%v，查问题推荐失败数量, 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), resp.Count)

	for _, log := range resp.Logs {
		cnt, e := strconv.ParseInt(log["cnt"], 10, 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cnt error: %v", e)
			continue
		}
		return cnt
	}
	return 0
}

func QaRecommendAllQuerry(ctx context.Context, daysLookback int) ([]CoreLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
推荐模型 推荐结束 and (__tag__:_container_name_: {{.BaseContainerName}}-chat-go-prod or __tag__:_container_name_: {{.BaseContainerName}}-chat-go-pre) | select * from (
    select 
    regexp_extract(message, '推荐模型, 推荐结束, count:(.*?), cost:(.*?) s', 1) count, 
    regexp_extract(message, '推荐模型, 推荐结束, count:(.*?), cost:(.*?) s', 2) cost, 
    trace_id, time, user_id, "__tag__:_container_name_" env
    from log  order by time desc
) limit %v
`
	query = FormatWithTemplate(query, nil)

	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	// 查询日志
	hlog.CtxDebugf(ctx, "问题推荐所有数量查询 query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "QaRecommendAllQuerry query log error: %v", err)
		return nil, err
	}

	// 打印查询结果
	hlog.CtxInfof(ctx, "日期%v，查问题推荐所有, 共%v条日志", time.Unix(from, 0).Format("2006-01-02"), resp.Count)

	result := []CoreLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse("2006-01-02 15:04:05.999", log["time"])
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		cost, e := strconv.ParseFloat(log["cost"], 64)
		if e != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		//count, e := strconv.ParseInt(log["count"], 10, 64)
		//if e != nil {
		//    hlog.CtxErrorf(ctx, "parse count error: %v", e)
		//    continue
		//}
		result = append(result, CoreLog{
			CoreName: "问题推荐",
			Node:     "调用推荐模型结束",
			Cost:     cost,
			TraceId:  log["trace_id"],
			Time:     t,
			UserId:   log["user_id"],
			Status:   consts.StatusSuccess,
		})
	}
	return result, nil

}

func NginxErrlogsQuery(ctx context.Context, host, url, date string) ([]NginxErrorLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.NGINX_LOG_STORE_NAME)

	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}

	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).Add(-8 * time.Hour).Unix()
	to := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, day.Location()).Add(-8 * time.Hour).Unix()
	query := `
| 
select user_id, trace_id, time, status, host, url, request_time cost,client_ip
from log where
%v url = '%v' and 
%v host = '%v' and 

method in ('GET', 'POST') and
status != 200
order by time desc
limit %v
`
	urlMute := ""
	if url == "" {
		urlMute = "--"
	}
	hostMute := ""
	if host == "" {
		hostMute = "--"
	}
	query = fmt.Sprintf(query, urlMute, url, hostMute, host, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "date: %v, nginx errlogs query: %v", date, query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "NginxErrlogsQuery query log error: %v", err)
		return nil, err
	}

	results := []NginxErrorLog{}
	for _, log := range resp.Logs {
		cost, err := strconv.ParseFloat(log["cost"], 64)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", err)
			continue
		}
		t, err := time.Parse("02/Jan/2006:15:04:05", log["time"])
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		result := NginxErrorLog{
			NginxLog: NginxLog{
				CleanUrl: log["url"],
				Time:     t,
				Method:   log["method"],
				Status:   log["status"],
				Host:     log["host"],
				Cost:     cost,
			},
			UserId:   log["user_id"],
			TraceId:  log["trace_id"],
			ClientIp: log["client_ip"],
		}
		results = append(results, result)
	}
	hlog.CtxDebugf(ctx, "date: %v, nginx errlogs query result lenth: %v", date, len(results))
	return results, nil
}

func ModelNginxErrlogsQuery(ctx context.Context, host, url, date string) ([]NginxErrorLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.MODEL_NGINX_LOG_STORE_NAME)

	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}

	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).Add(-8 * time.Hour).Unix()
	to := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, day.Location()).Add(-8 * time.Hour).Unix()
	//from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).Unix()
	//to := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, day.Location()).Unix()
	query := `
| 
select
"content.user_id" user_id,
"content.trace_id" trace_id,
"content.time" time,
"content.status" status,
"content.vhost" host,
"content.path" path,
"content.request_time" cost,
"__source__" client_ip
from log where

%v "content.path"  = '%v' and 
%v "content.vhost" = '%v' and 

"content.method"  in ('GET', 'POST') and
"content.status"  != 200 and
("content.channel" = '{{.BaseChannelName}}-prod' or "content.channel" = '{{.BaseChannelName}}-pre')
order by "content.time" desc
limit %v
`
	query = FormatWithTemplate(query, nil)
	pathMute := ""
	if url == "" {
		pathMute = "--"
	}
	hostMute := ""
	if host == "" {
		hostMute = "--"
	}
	query = fmt.Sprintf(query, pathMute, url, hostMute, host, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "date: %v, model nginx errlogs query: %v", date, query)
	//resp, err := logstore.GetLogs("", from, to, query, 100000, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "ModelNginxErrlogsQuery query log error: %v", err)
		return nil, err
	}

	results := []NginxErrorLog{}
	for _, log := range resp.Logs {
		cost, err := strconv.ParseFloat(log["cost"], 64)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", err)
			continue
		}
		t, err := time.Parse(time.RFC3339, log["time"])
		// 时间是UTC时间，需要+8小时
		t = t.Add(time.Hour * 8)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		result := NginxErrorLog{
			NginxLog: NginxLog{
				CleanUrl: log["path"],
				Time:     t,
				Method:   log["method"],
				Status:   log["status"],
				Host:     log["host"],
				Cost:     cost,
			},
			UserId:   log["user_id"],
			TraceId:  log["trace_id"],
			ClientIp: log["client_ip"],
		}
		results = append(results, result)
	}
	hlog.CtxDebugf(ctx, "date: %v, model nginx errlogs query result lenth: %v", date, len(results))
	return results, nil
}

type EndToEndLog struct {
	LogStoreName string
	TraceId      string
	UserId       string
	Time         string
	Message      string
	Host         string
	ApiPath      string
	Cost         float32
	ClientIp     string
	UA           string
	Channel      string
	OriginLog    map[string]string
}

func NginxLogQueryByTraceId(ctx context.Context, traceId, date string) ([]EndToEndLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.NGINX_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}
	hlog.CtxInfof(ctx, "get logstore: %v success", consts.NGINX_LOG_STORE_NAME)
	day, e := time.Parse("2006-01-02", date)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse date error: %v", e)
		return nil, err
	}
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()).Add(-8 * time.Hour).Unix()
	to := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 999999999, day.Location()).Add(-8 * time.Hour).Unix()
	query := `
| select
trace_id, user_id,
-- time,
host, url, request_time cost, client_ip, http_user_agent ua, channel, *
from log
where trace_id = '%v'
order by time desc
limit %v
`
	query = fmt.Sprintf(query, traceId, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "NginxLogQueryByTraceId query log error: %v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "日期%v，查nginxIngress, trace_id: %v 共%v条日志", date, traceId, len(resp.Logs))
	logs := []EndToEndLog{}
	for _, log := range resp.Logs {
		t, err := time.Parse("02/Jan/2006:15:04:05", log["time"])
		if t.Format("2006-01-02") != date {
			continue
		}
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		cost, err := strconv.ParseFloat(log["cost"], 32)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", err)
			continue
		}

		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}

			if strings.HasSuffix(key, "_0") {
				continue
			}
			originLog[key] = val
		}
		logs = append(logs, EndToEndLog{
			TraceId:      traceId,
			UserId:       log["user_id"],
			Time:         t.Format("2006-01-02 15:04:05.999"),
			Message:      "",
			Host:         log["host"],
			ApiPath:      log["url"],
			Cost:         float32(cost),
			ClientIp:     log["client_ip"],
			LogStoreName: consts.NGINX_LOG_STORE_NAME,
			UA:           log["ua"],
			Channel:      log["channel"],
			OriginLog:    originLog,
		})
	}
	return logs, nil
}

func NginxLogQueryByUserId(ctx context.Context, userId string, timeBegin, timeEnd time.Time) ([]EndToEndLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.NGINX_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}
	hlog.CtxInfof(ctx, "get logstore: %v success", consts.NGINX_LOG_STORE_NAME)
	from := timeBegin.Unix()
	to := timeEnd.Unix()
	query := `
%v|select 
trace_id, user_id,
host, url, request_time cost, client_ip, http_user_agent ua, channel, *
from log
order by time desc
limit %v
`
	query = fmt.Sprintf(query, userId, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "nginx sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "NginxLogQueryByTraceId query log error: %v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "日期(%v, %v)，查nginxIngress, userId: %v 共%v条日志", timeBegin, timeEnd, userId, len(resp.Logs))
	logs := []EndToEndLog{}
	for _, log := range resp.Logs {
		t, err := time.Parse("02/Jan/2006:15:04:05", log["time"])
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		cost, err := strconv.ParseFloat(log["cost"], 32)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", err)
			continue
		}

		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}
			key = strings.Trim(key, " ")

			if strings.HasSuffix(key, "_0") {
				continue
			}
			originLog[key] = val
		}
		logs = append(logs, EndToEndLog{
			TraceId:      log["trace_id"],
			UserId:       log["user_id"],
			Time:         t.Format("2006-01-02 15:04:05.999"),
			Message:      "",
			Host:         log["host"],
			ApiPath:      log["url"],
			Cost:         float32(cost),
			ClientIp:     log["client_ip"],
			LogStoreName: consts.NGINX_LOG_STORE_NAME,
			UA:           log["ua"],
			Channel:      log["channel"],
			OriginLog:    originLog,
		})
	}
	return logs, nil
}

func ModelNginxLogQueryByUserId(ctx context.Context, userId string, timeBegin, timeEnd time.Time) ([]EndToEndLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.MODEL_NGINX_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}
	hlog.CtxInfof(ctx, "get logstore: %v success", consts.MODEL_NGINX_LOG_STORE_NAME)

	from := timeBegin.Unix()
	to := timeEnd.Unix()

	query := `
%v|select 
"content.trace_id" trace_id, 
"content.user_id" user_id, 
"content.time" time, 
"content.vhost" host, 
"content.path" url,
"content.duration" cost ,
"content.http_user_agent" ua,
"content.channel" channel,
*
from log
order by "content.time" desc
limit %v
`
	query = fmt.Sprintf(query, userId, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "model nginx sql query: %v", query)
	// 查询日志
	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "ModelNginxLogQueryByTraceId query log error: %v", err)
		return nil, err
	}

	nlogs := []EndToEndLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse(time.RFC3339, log["time"])
		// 时间是UTC时间，需要+8小时
		t = t.Add(time.Hour * 8)
		//if t.Format("2006-01-02") != fromdayStr {
		//	continue
		//}
		if t.Unix() <= from {
			hlog.CtxDebugf(ctx, "time %v <= from %v", t, time.Unix(from, 0))
			continue
		}
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		cost, err := strconv.ParseFloat(log["cost"], 32)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}

			if strings.HasSuffix(key, "_0") {
				continue
			}
			if strings.HasPrefix(key, "content.") {
				key = strings.TrimPrefix(key, "content.")
			}
			originLog[key] = val
		}
		nlogs = append(nlogs, EndToEndLog{
			LogStoreName: consts.MODEL_NGINX_LOG_STORE_NAME,
			TraceId:      log["trace_id"],
			UserId:       log["user_id"],
			Time:         t.Format("2006-01-02 15:04:05.999"),
			Message:      "",
			Host:         log["host"],
			ApiPath:      log["url"],
			Cost:         float32(cost),
			ClientIp:     log["client_ip"],
			UA:           log["ua"],
			Channel:      log["channel"],
			OriginLog:    originLog,
		})
	}
	hlog.CtxInfof(ctx, "日期(%v, %v)，查modelIngress, userId: %v, 共%v条日志", timeBegin, timeEnd, userId, len(nlogs))
	return nlogs, nil
}

func ModelNginxLogQueryByTraceId(ctx context.Context, traceId, date string) ([]EndToEndLog, error) {
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.MODEL_NGINX_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.MODEL_NGINX_LOG_STORE_NAME)

	lookbackDay, err := time.Parse("2006-01-02", date)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse date error: %v", err)
		return nil, err
	}

	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	query := `
|select 
"content.trace_id" trace_id, 
"content.user_id" user_id, 
"content.time" time, 
"content.vhost" host, 
"content.path" url,
"content.duration" cost ,
"content.http_user_agent" ua,
"content.channel" channel,
*
from log
where "content.trace_id"='%v' 
order by "content.time" desc
limit %v
`
	query = fmt.Sprintf(query, traceId, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "model nginx sql query: %v", query)
	// 查询日志
	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "ModelNginxLogQueryByTraceId query log error: %v", err)
		return nil, err
	}

	nlogs := []EndToEndLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse(time.RFC3339, log["time"])
		// 时间是UTC时间，需要+8小时
		t = t.Add(time.Hour * 8)
		//if t.Format("2006-01-02") != fromdayStr {
		//	continue
		//}
		if t.Unix() <= from {
			hlog.CtxDebugf(ctx, "time %v <= from %v", t, time.Unix(from, 0))
			continue
		}
		if e != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", e)
			continue
		}
		cost, err := strconv.ParseFloat(log["cost"], 32)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse cost error: %v", e)
			continue
		}
		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}

			if strings.HasSuffix(key, "_0") {
				continue
			}
			if strings.HasPrefix(key, "content.") {
				key = strings.TrimPrefix(key, "content.")
			}
			originLog[key] = val
		}
		nlogs = append(nlogs, EndToEndLog{
			LogStoreName: consts.MODEL_NGINX_LOG_STORE_NAME,
			TraceId:      log["trace_id"],
			UserId:       log["user_id"],
			Time:         t.Format("2006-01-02 15:04:05.999"),
			Message:      "",
			Host:         log["host"],
			ApiPath:      log["url"],
			Cost:         float32(cost),
			ClientIp:     log["client_ip"],
			UA:           log["ua"],
			Channel:      log["channel"],
			OriginLog:    originLog,
		})
	}
	hlog.CtxInfof(ctx, "日期%v，查modelIngress, trace_id: %v, 共%v条日志", date, traceId, len(nlogs))
	return nlogs, nil
}

func BusinessLogQueryByTraceId(ctx context.Context, traceId, date string) ([]EndToEndLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	lookbackDay, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Add(-8 * time.Hour).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Add(-8 * time.Hour).Unix()

	query := `
|select
user_id, trace_id,
COALESCE(asctime, time) AS time,
-- asctime time,
ip client_ip, message msg,
*
from log where
trace_id = '%v'
order by asctime desc
limit %v
`
	query = fmt.Sprintf(query, traceId, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "business trace sql query: %v", query)
	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "BusinessLogQueryByTraceId query log error: %v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "日期%v，查business-pod, trace_id: %v, 共%v条日志", date, traceId, len(resp.Logs))
	logs := []EndToEndLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse("2006-01-02 15:04:05.999", log["time"])
		if e != nil {
			// 解析如2024-09-12T08:40:51.288+08:00的时间格式
			t, e = time.Parse(time.RFC3339, log["time"])
			if e != nil {
				hlog.CtxErrorf(ctx, "parse time error: %v", e)
				continue
			}
			//hlog.CtxErrorf(ctx, "parse time error: %v", e)
			//continue
		}
		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}

			if strings.HasSuffix(key, "_0") {
				continue
			}
			originLog[key] = val
		}
		logs = append(logs, EndToEndLog{
			LogStoreName: consts.BUSINESS_LOG_STORE_NAME,
			TraceId:      log["trace_id"],
			UserId:       log["user_id"],
			Time:         t.Format("2006-01-02 15:04:05.999"),
			Message:      log["msg"],
			Host:         "",
			ApiPath:      "",
			Cost:         0,
			ClientIp:     log["client_ip"],
			OriginLog:    originLog,
		})
	}
	return logs, nil
}
func BusinessLogQueryByUserId(ctx context.Context, userId string, timeBegin, timeEnd time.Time) ([]EndToEndLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)

	if err != nil {
		return nil, err
	}

	from := timeBegin.Unix()
	to := timeEnd.Unix()

	query := `
%v|select
user_id, trace_id,
COALESCE(asctime, time) AS time,
-- asctime time,
ip client_ip, message msg,
*
from log
order by asctime desc
limit %v
`
	query = fmt.Sprintf(query, userId, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "business trace sql query: %v", query)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "BusinessLogQueryByTraceId query log error: %v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "日期(%v, %v)，查business-pod, userId: %v, 共%v条日志", timeBegin, timeEnd, userId, len(resp.Logs))
	logs := []EndToEndLog{}
	for _, log := range resp.Logs {
		t, e := time.Parse("2006-01-02 15:04:05.999", log["time"])
		if e != nil {
			// 解析如2024-09-12T08:40:51.288+08:00的时间格式
			t, e = time.Parse(time.RFC3339, log["time"])
			if e != nil {
				hlog.CtxErrorf(ctx, "parse time error: %v", e)
				continue
			}
			//hlog.CtxErrorf(ctx, "parse time error: %v", e)
			//continue
		}
		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}

			if strings.HasSuffix(key, "_0") {
				continue
			}
			key = strings.Trim(key, " ")
			originLog[key] = val
		}
		logs = append(logs, EndToEndLog{
			LogStoreName: consts.BUSINESS_LOG_STORE_NAME,
			TraceId:      log["trace_id"],
			UserId:       log["user_id"],
			Time:         t.Format("2006-01-02 15:04:05.999"),
			Message:      log["msg"],
			Host:         "",
			ApiPath:      "",
			Cost:         0,
			ClientIp:     log["client_ip"],
			OriginLog:    originLog,
		})
	}
	return logs, nil
}

type TracebackDetail struct {
	ExcInfo   string
	Msg       string
	TraceId   string
	Time      time.Time
	UserId    string
	OriginLog map[string]string
}

func TracebackQuery(ctx context.Context, daysLookback int) ([]TracebackDetail, error) {
	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	//fromdayStr := lookbackDay.Format("2006-01-02")
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)
	query := `
(__tag__:_container_name_ : {{.BaseContainerName}}-python-prod or __tag__:_container_name_ : {{.BaseContainerName}}-python-pre) and  exc_info : "Traceback (most recent call last)" and not "pydantic"|  
select 
exc_info, message msg, trace_id, asctime time, user_id
from log
order by asctime desc
limit %v
`
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "business traceback sql query: %v", query)

	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "TracebackQuery query log error: %v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "日期%v，查business-pod, 共%v条日志", lookbackDay, len(resp.Logs))

	results := []TracebackDetail{}
	for _, log := range resp.Logs {
		t, err := time.Parse("2006-01-02 15:04:05,999", log["time"])
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}
			if strings.HasSuffix(key, "_0") {
				continue
			}
			originLog[key] = val
		}
		results = append(results, TracebackDetail{
			ExcInfo:   log["exc_info"],
			Msg:       log["msg"],
			TraceId:   log["trace_id"],
			Time:      t,
			UserId:    log["user_id"],
			OriginLog: originLog,
		})
	}
	return results, nil
}

func MultiNodeErrorQuery(ctx context.Context, daysLookback int) ([]CoreErrorLogs, error) {
	lookbackDay := time.Now().AddDate(0, 0, -daysLookback)
	//fromdayStr := lookbackDay.Format("2006-01-02")
	from := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 0, 0, 0, 0, lookbackDay.Location()).Unix()
	to := time.Date(lookbackDay.Year(), lookbackDay.Month(), lookbackDay.Day(), 23, 59, 59, 999999999, lookbackDay.Location()).Unix()

	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	hlog.CtxInfof(ctx, "get logstore: %v success", consts.BUSINESS_LOG_STORE_NAME)

	query := `
(__tag__:_container_name_: {{.BaseContainerName}}-python-pre or __tag__:_container_name_: {{.BaseContainerName}}-python-prod) | select * from (
    select 
    regexp_extract(message, 'extend core error, core_name:(.*?), core_node:(.*?), code:(.*?), msg:(.*?)$', 1) core_name,
    regexp_extract(message, 'extend core error, core_name:(.*?), core_node:(.*?), code:(.*?), msg:(.*?)$', 2) node_name,
    regexp_extract(message, 'extend core error, core_name:(.*?), core_node:(.*?), code:(.*?), msg:(.*?)$', 3) code,
    regexp_extract(message, 'extend core error, core_name:(.*?), core_node:(.*?), code:(.*?), msg:(.*?)$', 4) msg,asctime time,trace_id,user_id,"__tag__:_container_name_" env
    from log order by time desc
) where core_name='多文档' limit %v
`
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "multi node error sql query: %v", query)

	//resp, err := logstore.GetLogs("", from, to, query, consts.LOG_QUERY_LIMIT, 0, false)
	resp, err := QueryLogsWithRetry(ctx, logstore, from, to, query)

	if err != nil {
		hlog.CtxErrorf(ctx, "MultiNodeErrorQuery query log error: %v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "日期%v，查multi node error, 共%v条日志", lookbackDay, len(resp.Logs))

	results := []CoreErrorLogs{}
	for _, log := range resp.Logs {
		t, err := time.Parse("2006-01-02 15:04:05,999", log["time"])
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			continue
		}
		originLog := map[string]string{}
		for key, val := range log {
			if val == "null" || val == "-" || val == "" {
				continue
			}
			if strings.HasSuffix(key, "_0") {
				continue
			}
			originLog[key] = val
		}
		code, err := strconv.ParseInt(log["code"], 10, 64)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse code error: %v", err)
			continue
		}
		results = append(results, CoreErrorLogs{
			CoreName: log["core_name"],
			Code:     code,
			Msg:      log["msg"],
			TraceId:  log["trace_id"],
			Time:     t,
			UserId:   log["user_id"],
			Env:      log["env"],
			NodeName: log["node_name"],
		})
	}
	return results, nil
}

type FileProcessLog struct {
	Asctime time.Time `json:"asctime"`
	Message string    `json:"message"`
	TraceId string    `json:"trace_id"`
	UserId  string    `json:"user_id"`
	Cost    float64   `json:"cost"`
}

func ConvertFileProcessLog(ctx context.Context, logs []map[string]string) ([]FileProcessLog, error) {
	res := make([]FileProcessLog, len(logs))
	for i := range logs {
		t, _ := time.Parse(consts.DateTimeTemplate, logs[i]["asctime"])
		if t.IsZero() {
			t, _ = time.Parse(consts.DateTimeTemplate, logs[i]["time"])
		}

		c, _ := utils.GetCostFromMesage(logs[i]["message"])
		// if err != nil {
		// hlog.CtxErrorf(ctx, "ConvertFileProcessLog get cost error: %+v", err)
		// }

		uid := strings.TrimSpace(logs[i]["user_id"])
		if uid == "" {
			uid = strings.TrimSpace(logs[i]["U-Id"])
		}

		res[i] = FileProcessLog{
			Asctime: t,
			Message: strings.TrimSpace(logs[i]["message"]),
			TraceId: strings.TrimSpace(logs[i]["trace_id"]),
			UserId:  uid,
			Cost:    c,
		}
	}
	return res, nil
}

// pdf/url上传日志
func ResourceUploadQuery(ctx context.Context, resourceId, resourceType string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	query := `
	(__tag__:_container_name_ : lingowhale-python-prod or __tag__:_container_name_ : lingowhale-python-pre) and message: "%s %s %s" and funcName: core_link_print_cost | select * from log limit %v
	`
	query = FormatWithTemplate(query, nil)

	switch resourceType {
	case consts.PDF:
		query = fmt.Sprintf(query, "PDFParser", "单文件上传完成", resourceId, consts.LOG_QUERY_LIMIT)
	case consts.URL:
		query = fmt.Sprintf(query, "UrlParser", "网页上传完成", resourceId, consts.LOG_QUERY_LIMIT)
	}
	hlog.CtxDebugf(ctx, "ResourceUploadQuery query: %s", query)

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "ResourceUploadQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

// 苏秦解析日志
func PDFParserQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	query := `
	(__tag__:_container_name_ : {{.BaseContainerName}}-python-prod or __tag__:_container_name_ : {{.BaseContainerName}}-python-pre) and message: "core link core_name:PDFParser, resource_id:%s" and (message: "PDF解析完成" or message : "苏秦解析完成")  | select * from log
limit %v
	`
	query = FormatWithTemplate(query, nil)
	query = fmt.Sprintf(query, resourceId, consts.LOG_QUERY_LIMIT)
	hlog.CtxDebugf(ctx, "PDFParserQuery query: %s", query)

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "PDFParserQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func CrawlerQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	logstore, err := client.GetMetricStore(consts.FC_PROJECT_NAME, consts.FC_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	query := `
	serviceName:lingowhale_fc AND (functionName:web_url_parser_pre or functionName:web_url_parser_prod) and message: %s and not funcName
	`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "CrawlerQuery query: %s", query)

	logResp, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "CrawlerQuery query log error: %v", err)
		return nil, err
	}

	logs := logResp.Logs
	res := make([]FileProcessLog, len(logs))
	for i := range logs {
		asctime, userId, traceId, cost := utils.ExtractLogInfo(logs[i]["message"])
		res[i] = FileProcessLog{
			Asctime: asctime,
			UserId:  userId,
			TraceId: traceId,
			Cost:    cost,
			Message: strings.TrimSpace(logs[i]["message"]),
		}
	}
	return res, nil
}

func WcdParseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	query := `
	(__tag__:_container_name_ : edu-arch-go-prod or __tag__:_container_name_ : edu-arch-go-pre) and message: "ParseEduNode wcd" and message:"%s"
	`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "WcdParserQuery query: %s", query)

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "WcdParserQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func TextParseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	query := `
	(__tag__:_container_name_ : edu-arch-go-prod or __tag__:_container_name_ : edu-arch-go-pre) and message: "ParseEduNode end" and message: "%s"
	`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "TextParserQuery query: %s", query)

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "TextParserQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

// ! 这里只能用traceId查
func EduParseQuery(ctx context.Context, traceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	query := `
	(__tag__:_container_name_ : edu-arch-go-prod or __tag__:_container_name_ : edu-arch-go-pre) and message: "req path /edu_parse" and trace_id : "%s"
	`
	query = fmt.Sprintf(query, traceId)
	hlog.CtxDebugf(ctx, "EduParseQuery query: %s", query)

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "EduParseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func ParseFinishQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	query := `
	(__tag__:_container_name_ : edu-arch-go-prod or __tag__:_container_name_ : edu-arch-go-pre) and (message: "ParseEduNode parse end entryId:%s" or message: "ParseEduNode wcd text nil entryId:%s" or message: "ParseEduNode wcd worthless end entryId:%s")
	`
	query = fmt.Sprintf(query, resourceId, resourceId, resourceId)
	hlog.CtxDebugf(ctx, "ParseFinishQuery query: %s", query)

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "ParseFinishQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SingleOutlineBeginQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := fmt.Sprintf(`(__tag__:_container_name_ : lingowhale-python-prod or __tag__:_container_name_ : lingowhale-python-pre) and message: "summary start" and message: "generate_type:1." and (message: "file_id:%s" or message: "url_id:%s")`, resourceId, resourceId)
	hlog.CtxDebugf(ctx, "SingleOutlineBeginQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SingleOutlineBeginQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SingleOutlineEndQuery(ctx context.Context, traceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `(__tag__:_container_name_ : lingowhale-python-prod or __tag__:_container_name_ : lingowhale-python-pre) and message: "summary core core_name:大纲, node:大纲生成完成" and trace_id:"%s"`
	query = fmt.Sprintf(query, traceId)
	hlog.CtxDebugf(ctx, "SingleOutlineEndQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SingleOutlineEndQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SingleOverviewBeginQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := fmt.Sprintf(`(__tag__:_container_name_ : lingowhale-python-prod or __tag__:_container_name_ : lingowhale-python-pre) and message: "summary start" and message: "generate_type:0." and (message: "file_id:%s" or message: "url_id:%s")`, resourceId, resourceId)
	hlog.CtxDebugf(ctx, "SingleOverviewBeginQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SingleOverviewBeginQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SingleOverviewEndQuery(ctx context.Context, traceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `(__tag__:_container_name_ : lingowhale-python-prod or __tag__:_container_name_ : lingowhale-python-pre) and message: "summary core core_name:概述, node:生成概述结束" and trace_id:"%s"`
	query = fmt.Sprintf(query, traceId)
	hlog.CtxDebugf(ctx, "SingleOverviewEndQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SingleOverviewEndQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SingleViewpointBeginQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := fmt.Sprintf(`(__tag__:_container_name_ : lingowhale-python-prod or __tag__:_container_name_ : lingowhale-python-pre) and message: "summary start" and message: "generate_type:3." and (message: "file_id:%s" or message: "url_id:%s")`, resourceId, resourceId)
	hlog.CtxDebugf(ctx, "SingleViewpointBeginQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SingleViewpointBeginQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SingleViewpointEndQuery(ctx context.Context, traceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `(__tag__:_container_name_ : lingowhale-python-prod or __tag__:_container_name_ : lingowhale-python-pre) and message: "core link core_name:viewpoint, core_node:观点模型输出完成" and trace_id:"%s"`
	query = fmt.Sprintf(query, traceId)
	hlog.CtxDebugf(ctx, "SingleViewpointEndQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SingleViewpointEndQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiAnalysisQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `(__tag__:_container_name_: lingowhale-python-pre or __tag__:_container_name_: lingowhale-python-prod) and message: "multi core node node_name" and "%s" and "%s"`
	query = fmt.Sprintf(query, multiID, "ANALYSIS_ALL")
	hlog.CtxDebugf(ctx, "MultiThemeQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiAnalysisQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiThemeQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `(__tag__:_container_name_: lingowhale-python-pre or __tag__:_container_name_: lingowhale-python-prod) and message: "multi core node node_name" and "%s" and "%s"`
	query = fmt.Sprintf(query, multiID, "MERGE")
	hlog.CtxDebugf(ctx, "MultiThemeQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiThemeQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiOutlineQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `(__tag__:_container_name_: lingowhale-python-pre or __tag__:_container_name_: lingowhale-python-prod) and message: "multi core node node_name" and "%s" and "%s"`
	query = fmt.Sprintf(query, multiID, "THEME_ALL_SUMMARY")
	hlog.CtxDebugf(ctx, "MultiOutlineQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiOutlineQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func UploadOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest upload req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "UploadOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "UploadOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func UploadOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest upload resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "UploadOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "UploadOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func CrawlerOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest crawler req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "CrawlerOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.FC_PROJECT_NAME, consts.FC_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "CrawlerOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func CrawlerOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest crawler resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "CrawlerOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.FC_PROJECT_NAME, consts.FC_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "CrawlerOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func WcdOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest wcd req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "WcdOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "WcdOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func WcdOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest wcd resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "WcdOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "WcdOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SuqinOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest suqin req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "SuqinOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SuqinOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func SuqinOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest suqin resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "SuqinOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "SuqinOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func TextParseOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest text_parser req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "TextParseOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "TextParseOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func TextParseOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest text_parser resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "TextParseOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "TextParseOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func EduParserOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest edu_parser req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "EduParserOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "EduParserOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func EduParserOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest edu_parser resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "EduParserOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "EduParserOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func AbstractModelOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest abstract_model req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "AbstractModelOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "AbstractModelOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func AbstractModelOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest abstract_model resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "AbstractModelOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "AbstractModelOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func ViewPointModelOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest viewpoint_model req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "ViewPointModelOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "ViewPointModelOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func ViewPointModelOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest viewpoint_model resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "ViewPointModelOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "ViewPointModelOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func OutlineModelOutRequestQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest outline_model req" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "OutlineModelOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "OutlineModelOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func OutlineModelOutResponseQuery(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest outline_model resp" and "%s"`
	query = fmt.Sprintf(query, resourceId)
	hlog.CtxDebugf(ctx, "OutlineModelOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "OutlineModelOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}
func MultiSingleAnalysisModelOutRequestQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest single_analysis req" and "%s"`
	query = fmt.Sprintf(query, multiID)
	hlog.CtxDebugf(ctx, "MultiSingleAnalysisModelOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiSingleAnalysisModelOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiSingleAnalysisModelOutResponseQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest single_analysis resp" and "%s"`
	query = fmt.Sprintf(query, multiID)
	hlog.CtxDebugf(ctx, "MultiSingleAnalysisModelOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiSingleAnalysisModelOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiThemeModelOutRequestQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest multi_theme req" and "%s"`
	query = fmt.Sprintf(query, multiID)
	hlog.CtxDebugf(ctx, "MultiThemeModelOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiThemeModelOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiThemeModelOutResponseQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest multi_theme resp" and "%s"`
	query = fmt.Sprintf(query, multiID)
	hlog.CtxDebugf(ctx, "MultiThemeModelOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiThemeModelOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiOutlineModelOutRequestQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest multi_outline req" and "%s"`
	query = fmt.Sprintf(query, multiID)
	hlog.CtxDebugf(ctx, "MultiOutlineModelOutRequestQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiOutlineModelOutRequestQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}

func MultiOutlineModelOutResponseQuery(ctx context.Context, multiID string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
	query := `message: "OutRequest multi_outline resp" and "%s"`
	query = fmt.Sprintf(query, multiID)
	hlog.CtxDebugf(ctx, "MultiOutlineModelOutResponseQuery query: %s", query)

	logstore, err := client.GetMetricStore(consts.PROJECT_NAME, consts.BUSINESS_LOG_STORE_NAME)
	if err != nil {
		return nil, err
	}

	logs, err := QueryLogsWithRetry(ctx, logstore, timeBegin.Unix(), timeEnd.Unix(), query)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiOutlineModelOutResponseQuery query log error: %v", err)
		return nil, err
	}
	return ConvertFileProcessLog(ctx, logs.Logs)
}
