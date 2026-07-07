package handlers

import (
	"context"
	"empyrean_lens/conf"
	slshttp "empyrean_lens/http"
	"fmt"
	"strconv"
	"strings"
	"time"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	retry "github.com/avast/retry-go"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// SLS 查询语句

const (
	// 采集总览：仅统计含 domain 字段的 log_for_analysis 日志，避免 wrapper ERROR 重复计数
	sqCrawlOverview = `crawl_normal OR crawl_error | select message, count(*) as cnt from log where extra like '%"domain"%' group by message`

	// crawl_timing_normal 每条成功抓取均打点，crawl_took 单位秒，136K+ 样本/天
	// crawl_took 存在 extra JSON 字符串中，未建索引，用 regexp_extract 提取
	sqCrawlLatency = `crawl_timing_normal | select approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.50) as p50, approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.90) as p90, approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.95) as p95, approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.99) as p99`

	// 公众号专项 P50/P90/P95/P99：全文搜索过滤微信域名
	sqWeixinLatency = `crawl_timing_normal mp.weixin.qq.com | select approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.50) as p50, approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.90) as p90, approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.95) as p95, approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.99) as p99`

	// 微信第三方服务：每个供应商一行，success/fail/success_rate 在一条 SQL 里算
	// success  = "crawling X weixin success: ..." 日志（每 URL 一条）
	// fail     = kakalong 用 "failed after N attempts"，tikhub/renmin 用 "retry count"（均为最终失败日志）
	sqWeixinVendors = `kakalong weixin OR tikhub weixin OR renmin weixin | select
  case
    when message like '%kakalong%' then 'Kakalong'
    when message like '%tikhub%'   then 'Tikhub'
    when message like '%renmin%'   then 'Renmin'
  end as vendor,
  count_if(message like '%success%') as success,
  count_if(message like '%failed after%' or message like '%retry count%') as fail,
  round(count_if(message like '%success%') * 100.0 /
    nullif(count_if(message like '%success%') + count_if(message like '%failed after%' or message like '%retry count%'), 0), 2) as success_rate
from log
group by vendor
having vendor is not null
order by success desc`

	// 图片抓取：async_crawl_img 为主路径（消息队列，每条对应一张图片）
	// cost 字段在 message 中，格式 "async_crawl_img end, cost: X.X"
	sqImgCrawl = `async_crawl_img end | select count(*) as total, approx_percentile(cast(regexp_extract(message, 'cost: ([0-9.]+)', 1) as double), 0.50) as p50, approx_percentile(cast(regexp_extract(message, 'cost: ([0-9.]+)', 1) as double), 0.90) as p90, approx_percentile(cast(regexp_extract(message, 'cost: ([0-9.]+)', 1) as double), 0.95) as p95, approx_percentile(cast(regexp_extract(message, 'cost: ([0-9.]+)', 1) as double), 0.99) as p99`

	// 图片最终失败：get_img 所有重试（requests + playwright）均失败时打点，每张图片仅一条
	sqImgFailure = `所有方式均失败 | select count(*) as cnt`

	// 失败率最高的站点（跳过空域名）
	sqTopFailedDomains = `crawl_error | select regexp_extract(extra, '"domain":"([^"]*)"', 1) as domain, count(*) as cnt from log group by domain having domain is not null and domain != '' and domain != 'null' order by cnt desc limit 6`

	// 资源入库渠道分布：ResourceProcessor 日志，按 source + status 分组统计
	// message 格式："ResourceProcessor [source: Renminwang; ...; root_path: xxx] end process resource. status:success, cost:Xs"
	sqResourceIngestion = `ResourceProcessor | select
  regexp_extract(message, 'source: (\w+)', 1) as source,
  regexp_extract(message, 'status:(\w+)', 1) as status,
  count(*) as total
from log
where message like '%end process resource%'
group by source, status
order by source, total desc
limit 50`

	// 微信公众号抓取统计
	sqCrawlWeixin = `crawl_normal OR crawl_error | select message, count(*) as cnt from log where extra like '%mp.weixin.qq.com%' and extra like '%"domain"%' group by message`

	// 数据生成服务（repeater 容器），由调用方提供
	sqGenService = `__tag__:_container_name_: lingowhale-repeater-go-prod and __tag__:_namespace_: repeater and (OutRequest or level: error)
| select
    service_type,
    sum(total)  as total_calls,
    sum(errors) as error_ops,
    round(100.0 * (1.0 - cast(sum(errors) as double) / nullif(cast(sum(total) as double), 0)), 2) as success_pct
from (
    select
        regexp_extract(message, 'OutRequest (\S+)', 1) as service_type,
        1 as total,
        0 as errors
    from log
    where message like 'OutRequest %'
      and regexp_extract(message, 'OutRequest (\S+)', 1)
          not in ('upload_oss', 'add_voice', 'model_daily_voice')
    union all
    select service_type, 0 as total, 1 as errors
    from (
        select
            operation_id,
            case
                when message like 'StreamErrResp%'
                 and message not like '%code:21001%'          then 'single_abstract'
                when message like '解析模型返回数据失败%'
                  or message like '调用下游服务失败%'          then regexp_extract(message, 'model:(\w+)', 1)
                when message like 'StreamErr %'               then regexp_extract(message, 'StreamErr (\w+)', 1)
                when message like '多文档大纲模型生成失败%'    then 'multi_summary'
                when message like 'EditAudio error%'
                  or message like 'HSText2Voice%'             then 'hs_tts'
            end as service_type
        from log
        where level = 'error'
          and message not like '%origContent is empty%'
          and message not like '%内容过少，不支持生成%'
          and message not like '%AsyncExec error success%'
          and message not like 'FindOneByParseIDAndDataType%'
          and message not like 'SingAnalyze Error%'
          and message not like 'ErrorResponse%'
          and message not like 'server panic%'
        group by operation_id, service_type
        having service_type is not null
    )
) t
group by service_type
having service_type is not null and service_type != ''
order by total_calls desc`
)

// 生成服务类型中文名映射
var genServiceNames = map[string]string{
	"single_abstract": "概述",
	"single_outline":  "单文档全文拆解",

	"multi_abstract": "多文档概述",
	"multi_outline":  "多文档全文拆解",
	"multi_theme":    "多文档主题",
	"multi_summary":  "多文档全文拆解",

	"hs_tts":               "语音合成",
	"model_daily":          "日报",
	"smart_outline":        "要点透视",
	"single_enlightenment": "专属灵感",
}

// ─── 飞书卡片结构体（schema 2.0）─────────────────────────────

type fcMsg struct {
	MsgType string `json:"msg_type"`
	Card    fcCard `json:"card"`
}

type fcCard struct {
	Schema string   `json:"schema"`
	Config fcConfig `json:"config"`
	Header fcHeader `json:"header"`
	Body   fcBody   `json:"body"`
}

type fcConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type fcHeader struct {
	Title    fcText `json:"title"`
	Template string `json:"template"`
}

type fcText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type fcBody struct {
	Direction string        `json:"direction"`
	Elements  []interface{} `json:"elements"`
}

type fcMd struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type fcHr struct {
	Tag string `json:"tag"`
}

type fcTable struct {
	Tag         string              `json:"tag"`
	PageSize    int                 `json:"page_size"`
	HeaderStyle fcTableHeaderStyle  `json:"header_style"`
	Columns     []fcTableCol        `json:"columns"`
	Rows        []map[string]string `json:"rows"`
}

type fcTableHeaderStyle struct {
	BackgroundStyle string `json:"background_style"`
}

type fcTableCol struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	DataType    string `json:"data_type"`
	Width       string `json:"width"`
}

// ─── 辅助函数 ────────────────────────────────────────────────

var cstLoc = time.FixedZone("CST", 8*60*60)

func fmtCount(n int) string { return fmt.Sprintf("%d", n) }

func fmtPct(num, denom int) string {
	if denom == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(num)/float64(denom))
}

func fmtSec(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || s == "" || s == "null" {
		return "-"
	}
	return fmt.Sprintf("%.1f s", f)
}

func pctFromStr(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func makeTable(cols []fcTableCol, rows []map[string]string) fcTable {
	return fcTable{
		Tag:      "table",
		PageSize: 20,
		HeaderStyle: fcTableHeaderStyle{
			BackgroundStyle: "grey",
		},
		Columns: cols,
		Rows:    rows,
	}
}

func col(name, display, width string) fcTableCol {
	return fcTableCol{Name: name, DisplayName: display, DataType: "text", Width: width}
}

func md(content string) fcMd { return fcMd{Tag: "markdown", Content: content} }
func hr() fcHr               { return fcHr{Tag: "hr"} }

// ─── Handler ─────────────────────────────────────────────────

type LingowhaleStability struct{}

func NewLingowhaleStability() *LingowhaleStability {
	return &LingowhaleStability{}
}

func (h *LingowhaleStability) Handle(ctx context.Context, _ string) error {
	cfg := conf.GetConfig()

	now := time.Now().In(cstLoc)
	yesterday := now.AddDate(0, 0, -1)
	dayStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, cstLoc)
	dayEnd := dayStart.Add(24*time.Hour - time.Second)

	type qResult struct {
		logs []map[string]string
		err  error
	}

	hlog.CtxInfof(ctx, "[stability] start, date=%s", yesterday.Format("2006-01-02"))

	// 并发执行 SLS 查询
	run := func(name string, store slshttp.LogStore, query string, limit int) <-chan qResult {
		ch := make(chan qResult, 1)
		go func() {
			hlog.CtxInfof(ctx, "[stability] query start: %s", name)
			resp, err := slshttp.SlsQuery(ctx, store, query, dayStart, dayEnd, limit)
			if err != nil {
				hlog.CtxErrorf(ctx, "[stability] query failed: %s, err=%v", name, err)
				ch <- qResult{err: err}
				return
			}
			hlog.CtxInfof(ctx, "[stability] query done: %s, rows=%d", name, len(resp.Logs))
			ch <- qResult{logs: resp.Logs}
		}()
		return ch
	}

	chOverview    := run("crawl_overview",     slshttp.LogStore_BusinessPod,  sqCrawlOverview,   10)
	chLatency     := run("crawl_latency",      slshttp.LogStore_BusinessPod,  sqCrawlLatency,     1)
	chWxLatency   := run("weixin_latency",     slshttp.LogStore_BusinessPod,  sqWeixinLatency,    1)
	chTopFailed   := run("top_failed_domains", slshttp.LogStore_BusinessPod,  sqTopFailedDomains, 6)
	chWeixin      := run("crawl_weixin",       slshttp.LogStore_BusinessPod,  sqCrawlWeixin,      10)
	chWxVendors   := run("weixin_vendors",     slshttp.LogStore_BusinessPod,  sqWeixinVendors,    10)
	chImgCrawl    := run("img_crawl",          slshttp.LogStore_BusinessPod,  sqImgCrawl,          1)
	chImgFailure  := run("img_failure",        slshttp.LogStore_BusinessPod,  sqImgFailure,         1)
	chGen         := run("gen_service",        slshttp.LogStore_BusinessPod,  sqGenService,        200)
	chIngestion   := run("resource_ingestion", slshttp.LogStore_BusinessPod,  sqResourceIngestion,  20)

	rOverview    := <-chOverview
	rLatency     := <-chLatency
	rWxLatency   := <-chWxLatency
	rTopFailed   := <-chTopFailed
	rWeixin      := <-chWeixin
	rWxVendors   := <-chWxVendors
	rImgCrawl    := <-chImgCrawl
	rImgFailure  := <-chImgFailure
	rGen         := <-chGen
	rIngestion   := <-chIngestion

	hlog.CtxInfof(ctx, "[stability] all queries done")

	// ── 1. 解析采集总览 ──────────────────────────────────────
	crawlNormal, crawlError := 0, 0
	for _, l := range rOverview.logs {
		cnt, _ := strconv.Atoi(l["cnt"])
		switch l["message"] {
		case "crawl_normal":
			crawlNormal = cnt
		case "crawl_error":
			crawlError = cnt
		}
	}
	crawlTotal := crawlNormal + crawlError
	crawlSuccessRate := float64(0)
	if crawlTotal > 0 {
		crawlSuccessRate = 100 * float64(crawlNormal) / float64(crawlTotal)
	}

	// P50 / P99（来自 crawl_timing_normal，crawl_took 单位秒，全量 URL 级别样本）
	crawlP50, crawlP90, crawlP95, crawlP99 := "-", "-", "-", "-"
	if len(rLatency.logs) > 0 {
		crawlP50 = fmtSec(rLatency.logs[0]["p50"])
		crawlP90 = fmtSec(rLatency.logs[0]["p90"])
		crawlP95 = fmtSec(rLatency.logs[0]["p95"])
		crawlP99 = fmtSec(rLatency.logs[0]["p99"])
	}

	// ── 2. 微信第三方服务用量 ─────────────────────────────────────
	type wxVendorStat struct {
		vendor      string
		success     int
		fail        int
		successRate string
	}
	var wxVendors []wxVendorStat
	totalWxSuccess, totalWxFail := 0, 0
	for _, l := range rWxVendors.logs {
		s, _ := strconv.Atoi(l["success"])
		f, _ := strconv.Atoi(l["fail"])
		wxVendors = append(wxVendors, wxVendorStat{
			vendor:      l["vendor"],
			success:     s,
			fail:        f,
			successRate: l["success_rate"] + "%",
		})
		totalWxSuccess += s
		totalWxFail += f
	}
	wxVendorTotal := totalWxSuccess + totalWxFail
	totalWxVendor := totalWxSuccess // 兼容下方调用

	// ── 3. 微信公众号专项 ─────────────────────────────────────
	weixinNormal, weixinError := 0, 0
	for _, l := range rWeixin.logs {
		cnt, _ := strconv.Atoi(l["cnt"])
		switch l["message"] {
		case "crawl_normal":
			weixinNormal = cnt
		case "crawl_error":
			weixinError = cnt
		}
	}
	weixinTotal := weixinNormal + weixinError
	weixinRate := fmtPct(weixinNormal, weixinTotal)
	weixinRateVal := float64(0)
	if weixinTotal > 0 {
		weixinRateVal = 100 * float64(weixinNormal) / float64(weixinTotal)
	}

	weixinP50, weixinP90, weixinP95, weixinP99 := "-", "-", "-", "-"
	if len(rWxLatency.logs) > 0 {
		weixinP50 = fmtSec(rWxLatency.logs[0]["p50"])
		weixinP90 = fmtSec(rWxLatency.logs[0]["p90"])
		weixinP95 = fmtSec(rWxLatency.logs[0]["p95"])
		weixinP99 = fmtSec(rWxLatency.logs[0]["p99"])
	}

	// ── 4. 图片抓取（async_crawl_img 消息队列路径）─────────────
	imgTotal, imgSuccess := 0, 0
	imgP50, imgP90, imgP95, imgP99 := "-", "-", "-", "-"
	if len(rImgCrawl.logs) > 0 {
		l := rImgCrawl.logs[0]
		imgTotal, _ = strconv.Atoi(l["total"])
		imgP50 = fmtSec(l["p50"])
		imgP90 = fmtSec(l["p90"])
		imgP95 = fmtSec(l["p95"])
		imgP99 = fmtSec(l["p99"])
	}

	// ── 5. 图片抓取失败数 ─────────────────────────────────────────
	// 来源：get_img 所有方式均失败时打点，每张图片仅一条
	imgFailureCount := 0
	if len(rImgFailure.logs) > 0 {
		imgFailureCount, _ = strconv.Atoi(rImgFailure.logs[0]["cnt"])
	}
	// 成功数 = async_crawl_img 总调用 - 最终失败数
	imgSuccess = imgTotal - imgFailureCount

	// ── 6. Top 失败站点 ───────────────────────────────────────
	type domainStat struct {
		domain string
		cnt    int
	}
	var topDomains []domainStat
	for _, l := range rTopFailed.logs {
		domain := l["domain"]
		if domain == "" || domain == "null" {
			continue
		}
		cnt, _ := strconv.Atoi(l["cnt"])
		topDomains = append(topDomains, domainStat{domain: domain, cnt: cnt})
	}

	// ── 6. 解析数据生成服务 ───────────────────────────────────
	type genStat struct {
		serviceType string
		total       int
		errors      int
		successPct  string
	}
	var genStats []genStat
	totalGenCalls := 0
	for _, l := range rGen.logs {
		total, _ := strconv.Atoi(l["total_calls"])
		errors, _ := strconv.Atoi(l["error_ops"])
		genStats = append(genStats, genStat{
			serviceType: l["service_type"],
			total:       total,
			errors:      errors,
			successPct:  l["success_pct"],
		})
		totalGenCalls += total
	}

	// ── 7. 健康状态判断 ───────────────────────────────────────
	isHealthy := true
	var anomalies []string
	if crawlSuccessRate < 90 {
		isHealthy = false
		anomalies = append(anomalies, fmt.Sprintf("采集成功率 %.1f%% 低于阈值 90%%", crawlSuccessRate))
	}
	if weixinRateVal < 90 {
		isHealthy = false
		anomalies = append(anomalies, fmt.Sprintf("公众号抓取成功率 %.1f%% 低于阈值 90%%", weixinRateVal))
	}
	for _, g := range genStats {
		pct := pctFromStr(g.successPct)
		if pct > 0 && pct < 90 {
			isHealthy = false
			name := genServiceNames[g.serviceType]
			if name == "" {
				name = g.serviceType
			}
			anomalies = append(anomalies, fmt.Sprintf("生成服务[%s]成功率 %.1f%% 低于阈值 90%%", name, pct))
		}
	}

	healthIcon := "✅"
	healthText := "正常"
	headerTemplate := "green"
	if !isHealthy {
		healthIcon = "⚠️"
		healthText = "注意"
		headerTemplate = "yellow"
	}
	hlog.CtxInfof(ctx, "[stability] health=%s, crawlTotal=%d, crawlSuccess=%.1f%%, genServices=%d, anomalies=%d",
		healthText, crawlTotal, crawlSuccessRate, len(genStats), len(anomalies))

	// ── 7. 构建飞书卡片 ───────────────────────────────────────
	dateStr := yesterday.Format("2006-01-02")
	elements := []interface{}{}

	// 统计周期 + 健康状态
	elements = append(elements,
		md(fmt.Sprintf("统计周期：%s 00:00 ~ 24:00", dateStr)),
		md(fmt.Sprintf("**今日健康状态：%s %s**\n采集成功率 %s · P99 %s",
			healthIcon, healthText,
			fmt.Sprintf("%.1f%%", crawlSuccessRate),
			crawlP99,
		)),
		hr(),
	)

	// 一、抓取服务（含图片）
	elements = append(elements, md("**一、抓取服务**"))
	crawlSvcRows := []map[string]string{
		{
			"service":      "内容抓取总览",
			"total":        fmtCount(crawlTotal),
			"success_rate": fmt.Sprintf("%.1f%%", crawlSuccessRate),
			"p50":          crawlP50,
			"p90":          crawlP90,
			"p95":          crawlP95,
			"p99":          crawlP99,
		},
		{
			"service":      "公众号（自采代理）",
			"total":        fmtCount(weixinTotal),
			"success_rate": weixinRate,
			"p50":          weixinP50,
			"p90":          weixinP90,
			"p95":          weixinP95,
			"p99":          weixinP99,
		},
		{
			"service":      "图片抓取",
			"total":        fmtCount(imgTotal),
			"success_rate": fmtPct(imgSuccess, imgTotal),
			"p50":          imgP50,
			"p90":          imgP90,
			"p95":          imgP95,
			"p99":          imgP99,
		},
	}
	elements = append(elements, makeTable(
		[]fcTableCol{
			col("service", "服务", "auto"),
			col("total", "总量", "auto"),
			col("success_rate", "成功率", "auto"),
			col("p50", "P50", "auto"),
			col("p90", "P90", "auto"),
			col("p95", "P95", "auto"),
			col("p99", "P99", "auto"),
		},
		crawlSvcRows,
	))
	elements = append(elements, md(fmt.Sprintf("图片下载最终失败（pod 级）：**%d** 张", imgFailureCount)))

	// 1.1 微信第三方服务
	// 总调用 = 成功 + 全部策略失败；耗时暂无打点（第三方服务未记录 crawl_took）
	elements = append(elements, md(fmt.Sprintf(
		"**1.1 微信第三方服务**（总调用：%d · 成功率：%s）",
		wxVendorTotal, fmtPct(totalWxVendor, wxVendorTotal),
	)))
	if len(wxVendors) > 0 {
		var wxVendorRows []map[string]string
		for _, v := range wxVendors {
			wxVendorRows = append(wxVendorRows, map[string]string{
				"vendor":       v.vendor,
				"success_cnt":  fmtCount(v.success),
				"fail_cnt":     fmtCount(v.fail),
				"success_rate": v.successRate,
				"p50":          "-",
				"p99":          "-",
			})
		}
		elements = append(elements, makeTable(
			[]fcTableCol{
				col("vendor", "服务", "auto"),
				col("success_cnt", "成功量", "auto"),
				col("fail_cnt", "失败量", "auto"),
				col("success_rate", "成功率", "auto"),
				col("p50", "P50", "auto"),
				col("p99", "P99", "auto"),
			},
			wxVendorRows,
		))
	} else {
		elements = append(elements, md("今日无第三方服务调用记录"))
	}

	// 1.2 Top 5 失败站点
	if len(topDomains) > 0 {
		elements = append(elements, md("**1.2 Top 失败率站点**"))
		var topRows []map[string]string
		for i, d := range topDomains {
			if i >= 5 {
				break
			}
			topRows = append(topRows, map[string]string{
				"rank":   strconv.Itoa(i + 1),
				"domain": d.domain,
				"errors": fmtCount(d.cnt),
			})
		}
		elements = append(elements, makeTable(
			[]fcTableCol{col("rank", "#", "auto"), col("domain", "站点", "auto"), col("errors", "失败量", "auto")},
			topRows,
		))
	}
	elements = append(elements, hr())

	// 三、资源入库渠道分布
	// source 名称来自 DataSource.String()，与数字 source 的对应关系：
	// Subscription=11(自有抓取)，Renminwang=12(人民网)，Qingbo=14(清博)，FromMonitoring=监控抓取，SimResourceCluster=相似聚类
	sourceDisplayNames := map[string]string{
		"Subscription":       "自有抓取",
		"Renminwang":         "人民网",
		"Qingbo":             "清博",
		"FromMonitoring":     "监控抓取",
		"SimResourceCluster": "相似聚类",
		"XiaoYuZhou":         "小宇宙",
		"Hongmai":            "红麦",
	}

	// 按 source 聚合 success/fail/total
	type ingestionStat struct {
		success int
		fail    int
	}
	ingestionMap := map[string]*ingestionStat{}
	for _, l := range rIngestion.logs {
		src := l["source"]
		if src == "" {
			continue
		}
		if ingestionMap[src] == nil {
			ingestionMap[src] = &ingestionStat{}
		}
		cnt, _ := strconv.Atoi(l["total"])
		if l["status"] == "success" {
			ingestionMap[src].success += cnt
		} else {
			ingestionMap[src].fail += cnt
		}
	}

	// 按总量降序排列
	type ingestionRow struct {
		src  string
		stat *ingestionStat
	}
	var ingestionList []ingestionRow
	for src, stat := range ingestionMap {
		ingestionList = append(ingestionList, ingestionRow{src: src, stat: stat})
	}
	// 简单排序：总量大的在前
	for i := 0; i < len(ingestionList); i++ {
		for j := i + 1; j < len(ingestionList); j++ {
			ti := ingestionList[i].stat.success + ingestionList[i].stat.fail
			tj := ingestionList[j].stat.success + ingestionList[j].stat.fail
			if tj > ti {
				ingestionList[i], ingestionList[j] = ingestionList[j], ingestionList[i]
			}
		}
	}

	ingestionTotal := 0
	var ingestionTableRows []map[string]string
	for _, row := range ingestionList {
		total := row.stat.success + row.stat.fail
		ingestionTotal += total
		displayName := sourceDisplayNames[row.src]
		if displayName == "" {
			displayName = row.src
		}
		successRate := fmtPct(row.stat.success, total)
		ingestionTableRows = append(ingestionTableRows, map[string]string{
			"channel":      displayName,
			"total":        fmtCount(total),
			"success":      fmtCount(row.stat.success),
			"fail":         fmtCount(row.stat.fail),
			"success_rate": successRate,
		})
	}

	elements = append(elements, md(fmt.Sprintf("**三、资源入库**（总量：%s）", fmtCount(ingestionTotal))))
	if len(ingestionTableRows) > 0 {
		elements = append(elements, makeTable(
			[]fcTableCol{
				col("channel", "渠道", "auto"),
				col("total", "总量", "auto"),
				col("success", "成功", "auto"),
				col("fail", "失败", "auto"),
				col("success_rate", "成功率", "auto"),
			},
			ingestionTableRows,
		))
	} else if rIngestion.err != nil {
		elements = append(elements, md(fmt.Sprintf("> 资源入库数据查询失败：%v", rIngestion.err)))
	}
	elements = append(elements, hr())

	// 四、数据生成服务
	elements = append(elements, md(fmt.Sprintf("**四、数据生成服务**\n调用总量：%d", totalGenCalls)))
	if len(genStats) > 0 {
		var genRows []map[string]string
		for _, g := range genStats {
			name := genServiceNames[g.serviceType]
			if name == "" {
				name = g.serviceType
			}
			pct := pctFromStr(g.successPct)
			successDisplay := fmt.Sprintf("%.1f%%", pct)
			if g.total > 0 && pct < 95 {
				successDisplay = fmt.Sprintf("<font color='red'>%.1f%%</font>", pct)
			}
			genRows = append(genRows, map[string]string{
				"task_type":    name,
				"total":        fmtCount(g.total),
				"success_rate": successDisplay,
				"errors":       fmtCount(g.errors),
			})
		}
		elements = append(elements, makeTable(
			[]fcTableCol{
				col("task_type", "任务类型", "auto"),
				col("total", "总量", "auto"),
				col("success_rate", "成功率", "auto"),
				col("errors", "失败数", "auto"),
			},
			genRows,
		))
	} else if rGen.err != nil {
		elements = append(elements, md(fmt.Sprintf("> 生成服务数据查询失败：%v", rGen.err)))
	}
	elements = append(elements, hr())

	// 四、异常与告警
	elements = append(elements, md("**五、异常与告警**"))
	if len(anomalies) == 0 {
		elements = append(elements, md("今日无 P1/P2 级异常"))
	} else {
		alertLines := make([]string, 0, len(anomalies))
		for _, a := range anomalies {
			alertLines = append(alertLines, "• "+a)
		}
		elements = append(elements, md("<font color='red'>"+strings.Join(alertLines, "\n")+"</font>"))
	}

	// 报表注脚
	elements = append(elements, md(fmt.Sprintf(
		"ℹ️ 报表生成时间：%s · 数据来源：SLS 日志 · 阈值：成功率 < 90%% 标红",
		now.Format("2006-01-02 15:04"),
	)))

	msg := fcMsg{
		MsgType: "interactive",
		Card: fcCard{
			Schema: "2.0",
			Config: fcConfig{WideScreenMode: true},
			Header: fcHeader{
				Title:    fcText{Tag: "plain_text", Content: "📊 数据平台每日巡检报表"},
				Template: headerTemplate,
			},
			Body: fcBody{
				Direction: "vertical",
				Elements:  elements,
			},
		},
	}

	hlog.CtxInfof(ctx, "[stability] sending feishu card, webhook=%s", cfg.Notice.LingowhaleStabilityWebhook)
	if err := sendFeishuWebhook(ctx, cfg.Notice.LingowhaleStabilityWebhook, msg); err != nil {
		hlog.CtxErrorf(ctx, "[stability] send feishu webhook failed: %v", err)
		return err
	}
	hlog.CtxInfof(ctx, "[stability] feishu card sent successfully")
	return nil
}

// sendFeishuWebhook 向飞书自定义机器人 Webhook 发送消息，失败自动重试 3 次
func sendFeishuWebhook(ctx context.Context, webhookURL string, msg fcMsg) error {
	bodyBytes, err := sonic.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal feishu msg: %w", err)
	}

	type webhookResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	return retry.Do(
		func() error {
			resp := &webhookResp{}
			_, err := httplib.Do(ctx, webhookURL, map[string]string{
				consts.HeaderContentType: consts.MIMEApplicationJSON,
			}, bodyBytes, resp)
			if err != nil {
				hlog.CtxErrorf(ctx, "[stability] webhook http error: %v", err)
				return err
			}
			if resp.Code != 0 {
				err = fmt.Errorf("feishu webhook error code=%d msg=%s", resp.Code, resp.Msg)
				hlog.CtxErrorf(ctx, "[stability] %v", err)
				return err
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(500*time.Millisecond),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
	)
}
