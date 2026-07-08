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

	msg := renderDailyReport(day, results)
	webhook := conf.GetConfig().Notice.LingowhaleStabilityWebhook // 暂复用 stability 群，可拆独立 webhook
	err := retry.Do(func() error { return sendFeishuWebhook(ctx, webhook, msg) },
		retry.Attempts(3), retry.Delay(2*time.Second), retry.Context(ctx))
	if err != nil {
		hlog.CtxErrorf(ctx, "[daily-report] send feishu failed: %v", err)
		return err
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

func renderDailyReport(day time.Time, results []*report.Result) fcMsg {
	overall := report.OverallLevel(results)
	var elements []interface{}

	// 今日 Top 需关注（阈值/勾稽命中，按级别降序取 3 条）
	hits := report.TopHits(results, 3)
	if len(hits) == 0 {
		elements = append(elements, md("**今日无需关注事项**，各层指标健康 🟢"))
	} else {
		var b strings.Builder
		b.WriteString("**今日需关注**\n")
		for i, h := range hits {
			fmt.Fprintf(&b, "%d. %s %s\n", i+1, h.Level.Icon(), h.Msg)
		}
		elements = append(elements, md(b.String()))
	}
	elements = append(elements, hr())

	for _, r := range results {
		if r == nil {
			continue
		}
		title := fmt.Sprintf("%s **%s**", r.Level.Icon(), r.Section.Title)
		if r.Err != nil {
			elements = append(elements, md(title+fmt.Sprintf("\n> ⚠️ 本节生成失败（已隔离，不影响其他节）：%v", r.Err)), hr())
			continue
		}
		elements = append(elements, md(title))

		// 指标行：一节的关键指标拼一行（表格已含的细节不重复）
		if line := metricLine(r); line != "" {
			elements = append(elements, md(line))
		}
		for _, t := range r.Output.Tables {
			if t.Title != "" {
				elements = append(elements, md("> "+t.Title))
			}
			var cols []fcTableCol
			for _, c := range t.Cols {
				cols = append(cols, col(c.Name, c.Display, "auto"))
			}
			elements = append(elements, makeTable(cols, t.Rows))
		}
		for _, n := range r.Output.Notes {
			elements = append(elements, md("> "+n))
		}
		elements = append(elements, hr())
	}

	header := fmt.Sprintf("数据平台每日巡检报表 · %s", day.Format("2006-01-02"))
	template := map[report.Level]string{
		report.LevelOK: "green", report.LevelWarn: "yellow",
		report.LevelCrit: "red", report.LevelBroken: "yellow",
	}[overall]

	return fcMsg{
		MsgType: "interactive",
		Card: fcCard{
			Schema: "2.0",
			Config: fcConfig{WideScreenMode: true},
			Header: fcHeader{Title: fcText{Tag: "plain_text", Content: header}, Template: template},
			Body:   fcBody{Direction: "vertical", Elements: elements},
		},
	}
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
