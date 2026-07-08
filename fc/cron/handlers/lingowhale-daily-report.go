package handlers

// 数据平台每日巡检报表（三层：数据来源 / 处理与监控 / 有效入库 + 漏斗勾稽）。
// 口径定义全部在 report 子包 sections*.go；本文件只做三件事：
//   1. 注入数据源能力（SLS / mcp-db 代理 / wechat-spider 直连）
//   2. Redis 快照读写（环比 + 连续失败天数，key=daily-report:{date}，TTL 90 天）
//   3. 飞书卡片渲染与发送
// 与老 lingowhale-stability 并存过渡，稳定后可下老报表。

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal/redis"
	"empyrean_lens/fc/cron/handlers/report"
	slshttp "empyrean_lens/http"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

const (
	snapshotKeyFmt = "daily-report:%s" // daily-report:2026-07-07
	snapshotTTL    = 90 * 24 * time.Hour
	snapshotDays   = 7 // 读近 N 日快照（连续失败天数用）
)

type LingowhaleDailyReport struct{}

func NewLingowhaleDailyReport() *LingowhaleDailyReport { return &LingowhaleDailyReport{} }

// Handle 生成并发送报表。payload 可传 "2026-07-07" 指定报表日（重跑/补发），
// 空则默认昨天。
func (h *LingowhaleDailyReport) Handle(ctx context.Context, payload string) error {
	day := time.Now().In(cstLoc).AddDate(0, 0, -1)
	if p := strings.TrimSpace(payload); p != "" {
		parsed, err := time.ParseInLocation("2006-01-02", p, cstLoc)
		if err != nil {
			return fmt.Errorf("bad payload date %q: %w", p, err)
		}
		day = parsed
	}
	hlog.CtxInfof(ctx, "[daily-report] run for %s", day.Format("2006-01-02"))

	prevDays := loadSnapshots(ctx, day, snapshotDays)
	results, snapshot := report.Run(ctx, day, report.Sections(), queryFunc, prevDays)
	saveSnapshot(ctx, day, snapshot)

	msgs := renderDailyReport(day, results, prevDays)
	webhook := conf.GetConfig().Notice.LingowhaleStabilityWebhook // 暂复用 stability 群，可拆独立 webhook
	for i, msg := range msgs {
		err := retry.Do(func() error { return sendFeishuWebhook(ctx, webhook, msg) },
			retry.Attempts(3), retry.Delay(2*time.Second), retry.Context(ctx))
		if err != nil {
			hlog.CtxErrorf(ctx, "[daily-report] send feishu card %d failed: %v", i+1, err)
			return err
		}
	}
	hlog.CtxInfof(ctx, "[daily-report] sent, sections=%d", len(results))
	return nil
}

// ─── 数据源注入 ──────────────────────────────────────────────────────────────

func queryFunc(ctx context.Context, q report.Query, from, to time.Time) (report.Rows, error) {
	switch q.Source {
	case report.SourceSLS, report.SourceSLSNginx:
		store := slshttp.LogStore_BusinessPod
		if q.Source == report.SourceSLSNginx {
			store = slshttp.LogStore_NginxIngress
		}
		limit := q.Limit
		if limit == 0 {
			limit = 20
		}
		resp, err := slshttp.SlsQuery(ctx, store, q.SQL, from, to, limit)
		if err != nil {
			return nil, err
		}
		return resp.Logs, nil

	case report.SourceMongo:
		pipeline, err := parsePipeline(q.PipelineJSON, from, to)
		if err != nil {
			return nil, err
		}
		resp, err := slshttp.MongoDbAggregate(ctx, &slshttp.MongoAggregateReq{
			DbName:         q.DB,
			CollectionName: q.Collection,
			Pipeline:       pipeline,
		})
		if err != nil {
			return nil, err
		}
		return flattenAnyRows(resp.Data), nil

	case report.SourceSpiderDB:
		pj := injectDayWindow(q.PipelineJSON, from, to)
		return slshttp.SpiderAggregate(ctx, q.Collection, pj)
	}
	return nil, fmt.Errorf("unknown query source: %s", q.Source)
}

// injectDayWindow 替换 pipeline 里的 {{DAY_START}}/{{DAY_END}} 为 UTC 边界。
// 报表日为北京时间自然日，Mongo 存 UTC，因此 2026-07-07 → [07-06T16:00Z, 07-07T16:00Z)。
func injectDayWindow(pipelineJSON string, from, to time.Time) string {
	s := strings.ReplaceAll(pipelineJSON, "{{DAY_START}}", from.UTC().Format(time.RFC3339))
	return strings.ReplaceAll(s, "{{DAY_END}}", to.UTC().Format(time.RFC3339))
}

func parsePipeline(pipelineJSON string, from, to time.Time) ([]map[string]any, error) {
	var pipeline []map[string]any
	if err := sonic.UnmarshalString(injectDayWindow(pipelineJSON, from, to), &pipeline); err != nil {
		return nil, fmt.Errorf("bad pipeline json: %w", err)
	}
	return pipeline, nil
}

func flattenAnyRows(data []map[string]interface{}) report.Rows {
	rows := make(report.Rows, 0, len(data))
	for _, doc := range data {
		row := make(map[string]string, len(doc))
		for k, v := range doc {
			row[k] = fmt.Sprintf("%v", v)
		}
		rows = append(rows, row)
	}
	return rows
}

// ─── Redis 快照 ─────────────────────────────────────────────────────────────

var redisInitOnce sync.Once

// initRedisSafe：dal/redis.Init 连不上会 panic；快照是增强能力（环比/连续天数），
// Redis 不可用只降级为"无环比"，不阻塞报表。
func initRedisSafe(ctx context.Context) (ok bool) {
	defer func() {
		if p := recover(); p != nil {
			hlog.CtxErrorf(ctx, "[daily-report] redis init failed, run without snapshots: %v", p)
			ok = false
		}
	}()
	redisInitOnce.Do(redis.Init)
	return true
}

func loadSnapshots(ctx context.Context, day time.Time, n int) []report.Snapshot {
	if !initRedisSafe(ctx) {
		return nil
	}
	out := make([]report.Snapshot, 0, n)
	for i := 1; i <= n; i++ {
		key := fmt.Sprintf(snapshotKeyFmt, day.AddDate(0, 0, -i).Format("2006-01-02"))
		val, err := redis.GetVal(ctx, key).Result()
		if err != nil {
			out = append(out, nil) // 断档保留位置，Streak 遇 nil 即停
			continue
		}
		var s report.Snapshot
		if sonic.UnmarshalString(val, &s) != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, s)
	}
	return out
}

func saveSnapshot(ctx context.Context, day time.Time, s report.Snapshot) {
	if len(s) == 0 || !initRedisSafe(ctx) {
		return
	}
	data, err := sonic.MarshalString(s)
	if err != nil {
		return
	}
	key := fmt.Sprintf(snapshotKeyFmt, day.Format("2006-01-02"))
	if err := redis.KeySet(ctx, key, data, snapshotTTL); err != nil {
		hlog.CtxErrorf(ctx, "[daily-report] save snapshot failed: %v", err)
	}
}

// ─── 渲染 ───────────────────────────────────────────────────────────────────

func renderDailyReport(day time.Time, results []*report.Result, prevDays []report.Snapshot) []fcMsg {
	overall := report.OverallLevel(results)
	allMetrics := map[string]report.Metric{}
	byKey := map[string]*report.Result{}
	layerLevel := map[string]report.Level{}
	for _, r := range results {
		if r == nil {
			continue
		}
		byKey[r.Section.Key] = r
		if r.Output != nil {
			for _, m := range r.Output.Metrics {
				allMetrics[m.Key] = m
			}
		}
		l := r.Level
		if l == report.LevelBroken {
			l = report.LevelWarn
		}
		if l > layerLevel[r.Section.Layer] {
			layerLevel[r.Section.Layer] = l
		}
	}
	mt := func(key string) string {
		if m, ok := allMetrics[key]; ok {
			return m.Text
		}
		return "—"
	}
	mv := func(key string) float64 { return allMetrics[key].Value }
	prevOf := func(key string) (float64, bool) {
		if len(prevDays) == 0 || prevDays[0] == nil {
			return 0, false
		}
		v, ok := prevDays[0][key]
		return v, ok
	}
	delta := func(key string) string {
		if m, ok := allMetrics[key]; ok {
			return report.DeltaPct(m.Value, prevDays, key)
		}
		return "—"
	}
	deltaPP := func(key string) string {
		if p, ok := prevOf(key); ok {
			return fmt.Sprintf("%+.1fpp", mv(key)-p)
		}
		return "—"
	}
	deltaCount := func(key string) string {
		if p, ok := prevOf(key); ok {
			return fmt.Sprintf("%+.0f", mv(key)-p)
		}
		return "—"
	}
	dropStatus := func(key string) string {
		cur := mv(key)
		if cur == 0 {
			return "🔴"
		}
		if p, ok := prevOf(key); ok && p > 0 && cur < p*0.7 {
			return "🔴"
		}
		return "🟢"
	}
	rateStatus := func(key string, warn, crit float64) string {
		switch {
		case mv(key) < crit:
			return "🔴"
		case mv(key) < warn:
			return "🟡"
		default:
			return "🟢"
		}
	}
	metricCols := []fcTableCol{
		col("metric", "指标", "40%"), col("today", "今日", "30%"),
		col("delta", "环比", "16%"), col("status", "状态", "14%"),
	}
	template := map[report.Level]string{
		report.LevelOK: "green", report.LevelWarn: "yellow",
		report.LevelCrit: "red", report.LevelBroken: "yellow",
	}[overall]
	newCard := func(title string, elements []interface{}) fcMsg {
		return fcMsg{MsgType: "interactive", Card: fcCard{
			Schema: "2.0", Config: fcConfig{WideScreenMode: true},
			Header: fcHeader{Title: fcText{Tag: "plain_text", Content: title}, Template: template},
			Body:   fcBody{Direction: "vertical", Elements: elements},
		}}
	}

	// ════════ 卡 1：健康总览 + 第一层 · 数据来源 ════════
	var e1 []interface{}
	supplierP99 := allMetrics["supplier.lat.12.p99"].Value
	if v := allMetrics["supplier.lat.14.p99"].Value; v > supplierP99 {
		supplierP99 = v
	}
	overview := "**今日健康总览**\n" +
		fmt.Sprintf("%s **① 数据来源**　接收 %s 条 · 环比 %s · 推送时效 P99 %s\n",
			layerLevel[report.LayerL1].Icon(), mt("funnel.top"), delta("funnel.top"), fmtMinShort(supplierP99)) +
		fmt.Sprintf("%s **② 处理与监控**　入库处理成功率 %s · 解析成功率 %s · 生成最差成功率 %s\n",
			layerLevel[report.LayerL2].Icon(), mt("pipe.proc.rate"), mt("pipe.parse.rate"), mt("gen.worst_rate")) +
		fmt.Sprintf("%s **③ 有效入库**　有效入库 %s 篇 · 有效率 %s · 公众号端到端≤30min %s",
			layerLevel[report.LayerL3].Icon(), mt("eff.total"), mt("funnel.eff_rate_top"), mt("eff.e2e.weixin.le30m"))
	e1 = append(e1, md(overview))
	if hits := report.TopHits(results, 3); len(hits) > 0 {
		var b strings.Builder
		b.WriteString("**今日 Top 3 需关注**\n")
		for i, h := range hits {
			fmt.Fprintf(&b, "%d. %s %s\n", i+1, h.Level.Icon(), h.Msg)
		}
		e1 = append(e1, md(b.String()))
	}
	e1 = append(e1, hr())
	e1 = append(e1, md(fmt.Sprintf("%s **%s**", layerLevel[report.LayerL1].Icon(), report.LayerL1)))

	// 1.1 供应商（模板表 + 重复拦截列：推送量 − 丢失 − 重复拦截 = 进入处理，与 2.2 首行衔接）
	if r := byKey["supplier"]; r != nil {
		e1 = append(e1, md(fmt.Sprintf("%s **%s**", r.Level.Icon(), r.Section.Title)))
		procOf := map[string]string{"12": "m22.total.Renminwang", "14": "m22.total.Qingbo"}
		nameOf := map[string]string{"12": "人民网", "14": "清博"}
		var supRows []map[string]string
		for _, src := range []string{"14", "12"} {
			arrived := mv("supplier.arrive." + src)
			dedup := arrived - mv(procOf[src])
			if dedup < 0 {
				dedup = 0
			}
			status := "🟢"
			if rate := mv("supplier.recv.rate." + src); arrived == 0 || rate < 95 {
				status = "🔴"
			} else if rate < 99.5 {
				status = "🟡"
			}
			supRows = append(supRows, map[string]string{
				"supplier": nameOf[src],
				"push":     mt("supplier.arrive." + src),
				"delta":    delta("supplier.arrive." + src),
				"rate":     mt("supplier.recv.rate." + src),
				"dedup":    fmtCount(int(dedup)),
				"p50":      mt("supplier.lat." + src + ".p50"),
				"p90":      mt("supplier.lat." + src + ".p90"),
				"p99":      mt("supplier.lat." + src + ".p99"),
				"status":   status,
			})
		}
		e1 = append(e1, makeTable([]fcTableCol{
			col("supplier", "供应商", "auto"), col("push", "推送量", "auto"),
			col("delta", "环比昨日", "auto"), col("rate", "接收成功率", "auto"),
			col("dedup", "重复拦截", "auto"),
			col("p50", "时效P50", "auto"), col("p90", "P90", "auto"), col("p99", "P99", "auto"),
			col("status", "状态", "auto"),
		}, supRows))
		e1 = append(e1, md("> 推送量 −（1−接收成功率）丢失 − 重复拦截 = 进入处理（见 2.2 首行）。重复拦截 = 供应商连推的重复副本在入口被拦，非丢失。"))
		appendSectionExtras(&e1, byKey, "supplier", true)
	}
	// 1.2 自采集指标表
	if r := byKey["self"]; r != nil {
		e1 = append(e1, md(fmt.Sprintf("%s **%s**", r.Level.Icon(), r.Section.Title)))
		acctStatus := "🟢"
		if p, ok := prevOf("self.accounts.active"); ok && mv("self.accounts.active") < p*0.95 {
			acctStatus = "🟡"
		}
		e1 = append(e1, makeTable(metricCols, []map[string]string{
			{"metric": "采集量", "today": mt("self.push"), "delta": delta("self.push"), "status": dropStatus("self.push")},
			{"metric": "采集时效 P50 / P90 / P99", "today": mt("self.lat.p50") + " / " + mt("self.lat.p90") + " / " + mt("self.lat.p99"), "delta": "-", "status": "🟢"},
			{"metric": "覆盖账号数(有产出 / 监控总数)", "today": mt("self.accounts.active") + " / " + mt("self.accounts.total"), "delta": deltaCount("self.accounts.active"), "status": acctStatus},
		}))
		e1 = append(e1, md("> ~~推送成功率~~ 暂缺：需 spider 侧提供发出量（我方仅能见到达的），待接入后补充"))
		appendSectionExtras(&e1, byKey, "self", true)
	}
	// 1.3 订阅指标表 + Top 失败站点
	if r := byKey["sub"]; r != nil {
		e1 = append(e1, md(fmt.Sprintf("%s **%s**", r.Level.Icon(), r.Section.Title)))
		e1 = append(e1, makeTable(metricCols, []map[string]string{
			{"metric": "抓取任务总量", "today": mt("sub.tasks"), "delta": delta("sub.tasks"), "status": dropStatus("sub.tasks")},
			{"metric": "抓取成功率", "today": mt("sub.task_rate"), "delta": deltaPP("sub.task_rate"), "status": rateStatus("sub.task_rate", 88, 75)},
			{"metric": "新文时效 P50 / P90 / ≤4h占比", "today": mt("sub.fresh.p50") + " / " + mt("sub.fresh.p90") + " / " + mt("sub.fresh.le4h"), "delta": "-", "status": "🟢"},
		}))
		appendSectionExtras(&e1, byKey, "sub", false)
	}
	// 1.4 图片（附属）
	if r := byKey["img"]; r != nil {
		e1 = append(e1, md(fmt.Sprintf("%s **%s**（附属资源，不进内容漏斗）", r.Level.Icon(), r.Section.Title)))
		e1 = append(e1, md(fmt.Sprintf(
			"- 下载任务：**%s** ｜ 成功率 **%s**\n"+
				"- 失败图片：**%s** 张（去重）\n"+
				"- 耗时：P50 **%s** ｜ P99 **%s**",
			mt("img.total"), fmtPct1f(100-mv("img.fail_rate")), mt("img.fail_uniq"), mt("img.p50"), mt("img.p99"))))
		appendSectionExtras(&e1, byKey, "img", true)
	}
	card1 := newCard(fmt.Sprintf("数据平台每日巡检报表 · %s ｜ ① 数据来源层", day.Format("2006-01-02")), e1)

	// ════════ 卡 2：第二层 + 第三层 + 告警 ════════
	var e2 []interface{}
	e2 = append(e2, md(fmt.Sprintf("%s **%s**", layerLevel[report.LayerL2].Icon(), report.LayerL2)))
	for _, key := range []string{"pipe", "m22", "gen"} {
		if r := byKey[key]; r != nil {
			e2 = append(e2, md(fmt.Sprintf("%s **%s**", r.Level.Icon(), r.Section.Title)))
			if key == "m22" {
				// 处理总量与第一层推送量的关系说明（动态算前置拒绝量，防止每个读者都要问一遍）
				rmArrive, rmProc := mv("supplier.arrive.12"), mv("m22.total.Renminwang")
				qbArrive, qbProc := mv("supplier.arrive.14"), mv("m22.total.Qingbo")
				if rmArrive > 0 && rmProc > 0 {
					e2 = append(e2, md(fmt.Sprintf(
						"> **为什么处理总量 < 第一层推送量**：供应商每篇文章平均会连推多次，重复副本在入口被拦下（原件均已处理，无文章丢失，与 1.1 接收成功率不矛盾）。人民网今日到达 %s，其中重复副本 %s（%.1f%%），实际处理 %s；清博重复副本 %s（%.1f%%）。\n> **为什么「重复更新」行供应商是 —**：内部渠道的重复隔了几小时才再次提交，会进入处理流程、查库判出、计入「重复更新」；供应商的重复是秒级连推，在入口就被拦，不进本表。",
						fmtCount(int(rmArrive)), fmtCount(int(rmArrive-rmProc)), (rmArrive-rmProc)/rmArrive*100, fmtCount(int(rmProc)),
						fmtCount(int(qbArrive-qbProc)), (qbArrive-qbProc)/maxFloat(qbArrive, 1)*100)))
				}
			}
			appendSectionExtras(&e2, byKey, key, false)
		}
	}
	e2 = append(e2, hr())
	e2 = append(e2, md(fmt.Sprintf("%s **%s**", layerLevel[report.LayerL3].Icon(), report.LayerL3)))
	for _, key := range []string{"eff", "funnel"} {
		if r := byKey[key]; r != nil {
			e2 = append(e2, md(fmt.Sprintf("%s **%s**", r.Level.Icon(), r.Section.Title)))
			appendSectionExtras(&e2, byKey, key, false)
		}
	}
	e2 = append(e2, hr())
	allHits := report.TopHits(results, 100)
	var b strings.Builder
	b.WriteString("**四、异常与告警**\n")
	if len(allHits) == 0 {
		b.WriteString("今日无告警事项 🟢")
	} else {
		for _, h := range allHits {
			fmt.Fprintf(&b, "- %s %s\n", h.Level.Icon(), h.Msg)
		}
	}
	for _, r := range results {
		if r != nil && r.Err != nil {
			fmt.Fprintf(&b, "- ⚠️ 「%s」生成失败（已隔离）：%v\n", r.Section.Title, r.Err)
		}
	}
	e2 = append(e2, md(b.String()))
	card2 := newCard(fmt.Sprintf("数据平台每日巡检报表 · %s ｜ ②处理与监控 ③有效入库", day.Format("2006-01-02")), e2)

	return []fcMsg{card1, card2}
}

// appendSectionExtras 渲染某节的表格/图/注记。notesOnly 时只输出注记
//（该节核心指标已并入一层合并大表）。
func appendSectionExtras(elements *[]interface{}, byKey map[string]*report.Result, key string, notesOnly bool) {
	r := byKey[key]
	if r == nil || r.Output == nil {
		return
	}
	if r.Err != nil {
		*elements = append(*elements, md(fmt.Sprintf("> ⚠️ 「%s」生成失败（已隔离）：%v", r.Section.Title, r.Err)))
		return
	}
	if !notesOnly {
		for _, t := range r.Output.Tables {
			if t.Title != "" {
				*elements = append(*elements, md("> "+t.Title))
			}
			if t.Compact {
				*elements = append(*elements, md(compactTableMd(t)))
				continue
			}
			var cols []fcTableCol
			for _, c := range t.Cols {
				w := c.Width
				if w == "" {
					w = "auto"
				}
				cols = append(cols, col(c.Name, c.Display, w))
			}
			*elements = append(*elements, makeTable(cols, t.Rows))
		}
		for _, ch := range r.Output.Charts {
			var spec map[string]any
			if err := sonic.UnmarshalString(ch.SpecJSON, &spec); err == nil {
				*elements = append(*elements, map[string]any{"tag": "chart", "chart_spec": spec, "aspect_ratio": "16:9"})
			}
		}
	}
	for _, n := range r.Output.Notes {
		*elements = append(*elements, md("> "+n))
	}
}

// fmtMinShort 分钟数的短格式（总览行用）。
func fmtMinShort(min float64) string {
	if min <= 0 {
		return "—"
	}
	if min < 90 {
		return fmt.Sprintf("%.0f min", min)
	}
	return fmt.Sprintf("%.1f h", min/60)
}

func fmtPct1f(v float64) string { return fmt.Sprintf("%.1f%%", v) }

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// metricLine 挑选每节的"摘要指标"拼成一行 markdown（Display: Text · ...）。
// 只展示不在表格里重复的骨干指标，节奏与老报表一致。
var sectionSummaryKeys = map[string][]string{
	"supplier": {"resource_add.total"},
	"self":     {"self.push", "self.accept.rate", "self.regular", "self.lat.p50", "self.lat.p90", "self.accounts.active", "self.accounts.total", "self.crawl.rate"},
	"sub":      {"sub.tasks", "sub.task_rate", "sub.ok_urls", "sub.sched.completion", "sub.fresh.p50", "sub.fresh.le4h", "sub.backfill"},
	"img":      {"img.total", "img.fail_rate", "img.fail_uniq", "img.p50", "img.p99"},
	"m22":      {"m22.uniq_success"},
	"eff":      {"eff.total"},
}

// compactTableMd 把表格渲染成 markdown 行："**首列**：col value · col value"。
func compactTableMd(t report.Table) string {
	var b strings.Builder
	for _, row := range t.Rows {
		if len(t.Cols) == 0 {
			continue
		}
		fmt.Fprintf(&b, "**%s**", row[t.Cols[0].Name])
		for _, c := range t.Cols[1:] {
			v := row[c.Name]
			if v == "" || v == "—" {
				continue
			}
			fmt.Fprintf(&b, " · %s %s", c.Display, v)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func metricLine(r *report.Result) string {
	keys := sectionSummaryKeys[r.Section.Key]
	if len(keys) == 0 {
		return ""
	}
	byKey := map[string]report.Metric{}
	for _, m := range r.Output.Metrics {
		byKey[m.Key] = m
	}
	var parts []string
	for _, k := range keys {
		if m, ok := byKey[k]; ok {
			dim := ""
			if m.Dimension != "" && m.Dimension != report.DimPercent && m.Dimension != report.DimMinutes && m.Dimension != report.DimSeconds {
				dim = " " + string(m.Dimension)
			}
			parts = append(parts, fmt.Sprintf("%s **%s**%s", m.Display, m.Text, dim))
		}
	}
	return strings.Join(parts, " · ")
}
