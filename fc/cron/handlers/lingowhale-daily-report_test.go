package handlers

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/fc/cron/handlers/report"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

// Dry-run：真实执行全部查询与渲染，但不发送飞书卡片。
// 运行：MODE_ENV=dev DAILY_REPORT_DRYRUN=2026-07-07 go test ./fc/cron/handlers/ -run TestDailyReport_DryRun -v -timeout 600s
func TestDailyReport_DryRun(t *testing.T) {
	dayStr := os.Getenv("DAILY_REPORT_DRYRUN")
	if dayStr == "" {
		t.Skip("set DAILY_REPORT_DRYRUN=YYYY-MM-DD to run (executes real queries)")
	}
	conf.InitConfig()
	day, err := time.ParseInLocation("2006-01-02", dayStr, cstLoc)
	if err != nil {
		t.Fatalf("bad date: %v", err)
	}
	ctx := context.Background()

	prevDays := loadSnapshots(ctx, day, snapshotDays)
	t.Logf("loaded %d prev snapshots", len(prevDays))

	results, snapshot := report.Run(ctx, day, report.Sections(), queryFunc, prevDays)
	if os.Getenv("DAILY_REPORT_SAVE") == "1" {
		saveSnapshot(ctx, day, snapshot)
		t.Logf("snapshot SAVED to redis for %s (%d metrics)", dayStr, len(snapshot))
	}
	if os.Getenv("DAILY_REPORT_SINK") == "1" {
		sinkMetrics(ctx, day, results)
		t.Logf("metrics SINKED to bitable for %s", dayStr)
	}
	if os.Getenv("DAILY_REPORT_OPS") == "1" {
		sinkFailedSites(ctx, day, results)
		syncIssueTracker(ctx, day, results)
		t.Logf("ops sinks done for %s", dayStr)
	}
	if os.Getenv("DAILY_REPORT_WIKI") == "1" {
		url := sinkWikiDaily(ctx, day, renderDailyReport(day, results, prevDays))
		t.Logf("wiki doc: %s", url)
	}

	for _, r := range results {
		if r == nil {
			continue
		}
		if r.Err != nil {
			t.Errorf("section %s FAILED: %v", r.Section.Key, r.Err)
			continue
		}
		t.Logf("── %s %s ── metrics=%d tables=%d hits=%d",
			r.Level.Icon(), r.Section.Title, len(r.Output.Metrics), len(r.Output.Tables), len(r.Hits))
		for _, m := range r.Output.Metrics {
			t.Logf("   %-28s = %-14s [%s]", m.Key, m.Text, m.Dimension)
		}
		for _, h := range r.Hits {
			t.Logf("   HIT %s %s", h.Level.Icon(), h.Msg)
		}
		for _, n := range r.Output.Notes {
			t.Logf("   NOTE %s", n)
		}
	}
	t.Logf("snapshot metrics: %d", len(snapshot))

	// 渲染卡片并落盘（人工检查结构）
	msgs := renderDailyReport(day, results, prevDays)
	data, _ := sonic.MarshalIndent(msgs, "", "  ")
	out := fmt.Sprintf("/tmp/daily-report-%s.json", dayStr)
	_ = os.WriteFile(out, data, 0644)
	t.Logf("card json written to %s (%d bytes)", out, len(data))

	// 设置 DAILY_REPORT_WEBHOOK 时实际发送（验收用）。与线上一致：有文档链接发摘要卡，否则发全文
	if wh := os.Getenv("DAILY_REPORT_WEBHOOK"); wh != "" {
		toSend := msgs
		if docURL := sinkWikiDaily(ctx, day, msgs); docURL != "" {
			aiText := aiDailySummary(ctx, day, results, prevDays)
			t.Logf("AI 解读: %s", aiText)
			toSend = []fcMsg{renderSummaryCard(day, results, prevDays, docURL, aiText)}
		}
		for i, m := range toSend {
			if err := sendFeishuWebhook(ctx, wh, m); err != nil {
				t.Fatalf("send card %d failed: %v", i+1, err)
			}
		}
		t.Logf("%d cards SENT to %s", len(toSend), wh)
	} else {
		t.Log("NOT sent (set DAILY_REPORT_WEBHOOK to send)")
	}
}
