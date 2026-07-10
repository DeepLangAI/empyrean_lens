package handlers

// 数据平台每日巡检报表（三层：数据来源 / 处理与监控 / 有效入库 + 漏斗勾稽）。
// 口径定义全部在 report 子包 sections*.go；本文件只做三件事：
//   1. 注入数据源能力（SLS / mcp-db 代理，wechat-spider 库也走代理）
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
// 空或 "{}"（FC 控制台触发消息的默认填充值）则默认昨天。
func (h *LingowhaleDailyReport) Handle(ctx context.Context, payload string) error {
	day := time.Now().In(cstLoc).AddDate(0, 0, -1)
	if p := strings.TrimSpace(payload); p != "" && p != "{}" {
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
	webhook := conf.GetConfig().Notice.LingowhaleDailyReportWebhook
	if webhook == "" {
		webhook = conf.GetConfig().Notice.LingowhaleStabilityWebhook
	}
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
	// 状态标签：告警计数（红/黄），无告警显示绿色"今日正常"
	var headerTags []fcTextTag
	{
		var crit, warn int
		for _, r := range results {
			if r == nil {
				continue
			}
			for _, h := range r.Hits {
				if h.Level == report.LevelCrit {
					crit++
				} else if h.Level == report.LevelWarn {
					warn++
				}
			}
		}
		if crit > 0 {
			headerTags = append(headerTags, fcTextTag{Tag: "text_tag", Text: fcText{Tag: "plain_text", Content: fmt.Sprintf("%d 项严重", crit)}, Color: "red"})
		}
		if warn > 0 {
			headerTags = append(headerTags, fcTextTag{Tag: "text_tag", Text: fcText{Tag: "plain_text", Content: fmt.Sprintf("%d 项关注", warn)}, Color: "yellow"})
		}
		if crit == 0 && warn == 0 {
			headerTags = append(headerTags, fcTextTag{Tag: "text_tag", Text: fcText{Tag: "plain_text", Content: "今日正常"}, Color: "green"})
		}
	}
	subtitle := fmt.Sprintf("统计周期 %s 00:00–24:00", day.Format("01-02"))
	newCard := func(title string, elements []interface{}) fcMsg {
		return fcMsg{MsgType: "interactive", Card: fcCard{
			Schema: "2.0", Config: fcConfig{WideScreenMode: true, WidthMode: "fill"},
			Header: fcHeader{
				Title:       fcText{Tag: "plain_text", Content: title},
				Subtitle:    &fcText{Tag: "plain_text", Content: subtitle},
				TextTagList: headerTags,
				Template:    template,
			},
			Body: fcBody{Direction: "vertical", Elements: elements},
		}}
	}
	// secHead 输出节标题行（图标 + 标题）。飞书折叠面板不支持内嵌 table 组件，
	// 折叠方案已废弃，用 width_mode=fill 加宽解决面积问题。
	secHead := func(dst *[]interface{}, r *report.Result) {
		*dst = append(*dst, md(fmt.Sprintf("%s **%s**", r.Level.Icon(), r.Section.Title)))
	}

	// ════════ 卡 1：健康总览 + 第一层 · 数据来源 ════════
	var e1 []interface{}
	supplierP99 := allMetrics["supplier.lat.12.p99"].Value
	if v := allMetrics["supplier.lat.14.p99"].Value; v > supplierP99 {
		supplierP99 = v
	}
	overviewCol := func(icon, name, bigNum, unit, sub string) map[string]any {
		return map[string]any{
			"tag": "column", "width": "weighted", "weight": 1, "vertical_align": "top",
			"elements": []any{map[string]any{"tag": "markdown",
				"content": fmt.Sprintf("%s **%s**\n**%s** %s\n%s", icon, name, bigNum, unit, sub)}},
		}
	}
	e1 = append(e1, md("**今日健康总览**"), map[string]any{
		"tag": "column_set", "flex_mode": "none", "horizontal_spacing": "8px",
		"columns": []any{
			overviewCol(layerLevel[report.LayerL1].Icon(), "数据来源", mt("funnel.top"), "条接收",
				fmt.Sprintf("环比 %s · 推送P99 %s", delta("funnel.top"), fmtMinShort(supplierP99))),
			overviewCol(layerLevel[report.LayerL2].Icon(), "处理与监控", mt("m22.clean_rate"), "内容处理成功率",
				fmt.Sprintf("解析 %s · 生成最低 %s", mt("pipe.parse.rate"), mt("gen.worst_rate"))),
			overviewCol(layerLevel[report.LayerL3].Icon(), "有效入库", mt("eff.total"), "篇净新增",
				fmt.Sprintf("环比 %s · 公众号发布后30分钟内可见 %s", delta("eff.total"), mt("eff.e2e.weixin.le30m"))),
		},
	})
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
		var sec []interface{}
		secHead(&e1, r)
		procOf := map[string]string{"12": "m22.total.Renminwang", "14": "m22.total.Qingbo"}
		nameOf := map[string]string{"12": "人民网", "14": "清博"}
		var supRows []map[string]string
		for _, src := range []string{"14", "12"} {
			arrived := mv("supplier.arrive." + src)
			// 重复拦截 = 汇聚点到达(归因) − 进入处理：两个数同点位，不受队列积压污染
			dedup := mv("supplier.push."+src) - mv(procOf[src])
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
				"p50":      mt("supplier.lat." + src + ".p50"),
				"p90":      mt("supplier.lat." + src + ".p90"),
				"p99":      mt("supplier.lat." + src + ".p99"),
				"status":   status,
			})
			_ = dedup
		}
		sec = append(sec, makeTable([]fcTableCol{
			col("supplier", "供应商", "auto"), col("push", "推送量", "auto"),
			col("delta", "环比昨日", "auto"), col("rate", "接收成功率", "auto"),
			col("p50", "时效P50", "auto"), col("p90", "P90", "auto"), col("p99", "P99", "auto"),
			col("status", "状态", "auto"),
		}, supRows))
		appendSectionExtras(&sec, byKey, "supplier", true)
		e1 = append(e1, sec...)
	}
	// 1.2 自采集指标表
	if r := byKey["self"]; r != nil {
		var sec []interface{}
		secHead(&e1, r)
		acctStatus := "🟢"
		if p, ok := prevOf("self.accounts.active"); ok && mv("self.accounts.active") < p*0.95 {
			acctStatus = "🟡"
		}
		sec = append(sec, makeTable(metricCols, []map[string]string{
			{"metric": "采集量", "today": mt("self.regular"), "delta": delta("self.regular"), "status": dropStatus("self.regular")},
			{"metric": "采集时效 P50 / P90 / P99", "today": mt("self.lat.p50") + " / " + mt("self.lat.p90") + " / " + mt("self.lat.p99"), "delta": "-", "status": "🟢"},
			{"metric": "覆盖账号数(有产出 / 监控总数)", "today": mt("self.accounts.active") + " / " + mt("self.accounts.total"), "delta": deltaCount("self.accounts.active"), "status": acctStatus},
		}))
		sec = append(sec, md("> ~~推送成功率~~ 暂缺：需 spider 侧提供发出量（我方仅能见到达的），待接入后补充"))
		appendSectionExtras(&sec, byKey, "self", true)
		e1 = append(e1, sec...)
	}
	// 1.3 订阅指标表 + Top 失败站点
	if r := byKey["sub"]; r != nil {
		var sec []interface{}
		secHead(&e1, r)
		sec = append(sec, makeTable(metricCols, []map[string]string{
			{"metric": "抓取任务总量", "today": mt("sub.tasks"), "delta": delta("sub.tasks"), "status": dropStatus("sub.tasks")},
			{"metric": "抓取成功率", "today": mt("sub.task_rate"), "delta": deltaPP("sub.task_rate"), "status": rateStatus("sub.task_rate", 88, 75)},
			{"metric": "新文时效 P50 / P90 / ≤4h占比", "today": mt("sub.fresh.p50") + " / " + mt("sub.fresh.p90") + " / " + mt("sub.fresh.le4h"), "delta": "-", "status": "🟢"},
		}))
		appendSectionExtras(&sec, byKey, "sub", false)
		e1 = append(e1, sec...)
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
			var sec []interface{}
			secHead(&e2, r)
			if key == "pipe" && r.Output != nil {
				// 第三行与总览统一口径：名字说清楚、成功率用剔除重复后的干净数
				for ti := range r.Output.Tables {
					for ri := range r.Output.Tables[ti].Rows {
						row := r.Output.Tables[ti].Rows[ri]
						if row["stage"] == "入库处理" {
							row["stage"] = "资源处理全流程"
							row["rate"] = mt("m22.clean_rate")
						}
					}
					// 生成环节：复用 2.3 已算好的口径，不重复查询。只算内容生成
					// （概述/大纲类），语音合成与日报是衍生产品不算环节。
					// 量纲是模型调用次数（非篇），P99 按任务类型差异大，明细见 2.3
					if mv("gen.text_calls") > 0 {
						r.Output.Tables[ti].Rows = append(r.Output.Tables[ti].Rows, map[string]string{
							"stage": "内容生成(概述/大纲类)", "vol": mt("gen.text_calls") + " 次调用",
							"rate": mt("gen.text_rate"), "p50": "—", "p99": "见2.3",
						})
					}
				}
			}
			if key == "m22" {
				// 矩阵顶部插入「到达量(拦截前) + ⓪入口拦截」两行，每列可竖着做减法
				arriveOf := map[string]string{
					"SubRSS": "m22.arrive.SubRSS", "SubWeb": "m22.arrive.SubWeb",
					"Renminwang": "supplier.push.12", "Qingbo": "supplier.push.14",
					"FromMonitoring": "supplier.push.11",
				}
				totalOf := map[string]string{
					"SubRSS": "m22.total.SubRSS", "SubWeb": "m22.total.SubWeb",
					"Renminwang": "m22.total.Renminwang", "Qingbo": "m22.total.Qingbo",
					"FromMonitoring": "m22.total.FromMonitoring",
				}
				for ti := range r.Output.Tables {
					t := &r.Output.Tables[ti]
					if t.Compact || len(t.Cols) == 0 || t.Cols[0].Name != "stage" {
						continue
					}
					arriveRow := map[string]string{"stage": "到达处理"}
					dedupRow := map[string]string{"stage": "− 重复推送(入口拦截)"}
					for _, c := range t.Cols[1:] {
						a, p := mv(arriveOf[c.Name]), mv(totalOf[c.Name])
						arriveRow[c.Name] = fmtTh(a)
						d := a - p
						if d < 0 {
							d = 0 // 泄洪日处理量含前日积压，钳为 0
						}
						dedupRow[c.Name] = fmtTh(d)
					}
					t.Rows = append([]map[string]string{arriveRow, dedupRow}, t.Rows...)
				}
				// 处理总量与第一层推送量的关系说明（动态算前置拒绝量，防止每个读者都要问一遍）
				rmArrive, rmProc := mv("supplier.arrive.12"), mv("m22.total.Renminwang")
				if rmArrive > 0 && rmProc > 0 {
					sec = append(sec, md(fmt.Sprintf(
						"> 三种「重复」：**重复推送**＝同一篇文章被再次推来，在处理入口直接拦下（今日人民网占到达 %.1f%%，均非丢失）；**内容判重淘汰**＝换了链接/渠道但正文是同一篇，解析后判出、淘汰后到的；**重推旧文**＝库里已有，仅更新阅读数等数据，不新增。",
						(rmArrive-rmProc)/rmArrive*100)))
				}
			}
			appendSectionExtras(&sec, byKey, key, false)
			e2 = append(e2, sec...)
		}
	}
	e2 = append(e2, hr())
	e2 = append(e2, md(fmt.Sprintf("%s **%s**", layerLevel[report.LayerL3].Icon(), report.LayerL3)))
	for _, key := range []string{"eff", "funnel"} {
		if r := byKey[key]; r != nil {
			secHead(&e2, r)
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

// fmtTh 千分位整数格式化（卡片展示用）
func fmtTh(v float64) string {
	n := int64(v + 0.5)
	neg := n < 0
	if neg {
		n = -n
	}
	str := fmt.Sprintf("%d", n)
	for i := len(str) - 3; i > 0; i -= 3 {
		str = str[:i] + "," + str[i:]
	}
	if neg {
		return "-" + str
	}
	return str
}

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
