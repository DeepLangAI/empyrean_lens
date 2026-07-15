package handlers

// 运维数据落表（第二优先三件套的 ③④）：
//   sinkFailedSites  失败站点日志表——每天 Top 失败站点入表，站点故障史可查询（换源决策依据）
//   syncIssueTracker 问题跟踪表——告警自动开卡：新告警建记录，重复告警更新"最近出现/当前级别"，
//                    不自动关闭（是否解决由人判断），持续天数=最近出现−首次发现，一眼看出挂了多久
// 均为旁路：失败只记日志，不影响发报。

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"empyrean_lens/conf"
	"empyrean_lens/fc/cron/handlers/report"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// parseNum 解析卡片里的展示数字（"1,458" / "91.8%" / "—"）。
func parseNum(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(s, ",", ""), "%"))
	if s == "" || s == "—" || s == "-" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// sinkFailedSites 把 1.3 的「Top 失败站点」表写入失败站点日志表（同日先删后写）。
func sinkFailedSites(ctx context.Context, day time.Time, results []*report.Result) {
	cfg := conf.GetConfig().MetricsSink
	if cfg.BitableAppToken == "" || cfg.SitesTableID == "" {
		return
	}
	var rows []map[string]string
	for _, r := range results {
		if r == nil || r.Output == nil || r.Section.Key != "sub" {
			continue
		}
		for _, t := range r.Output.Tables {
			if t.Title == "Top 失败站点" {
				rows = t.Rows
			}
		}
	}
	if len(rows) == 0 {
		return
	}
	token, err := tenantToken(ctx, conf.GetConfig().MetricsSink.FeishuAppID, conf.GetConfig().MetricsSink.FeishuAppSecret)
	if err != nil {
		hlog.CtxErrorf(ctx, "[sites-sink] token: %v", err)
		return
	}
	dayMs := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, cstLoc).UnixMilli()
	if err := deleteDayRecordsIn(ctx, token, cfg.SitesTableID, dayMs); err != nil {
		hlog.CtxErrorf(ctx, "[sites-sink] delete old: %v", err)
		return
	}
	records := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		records = append(records, map[string]any{"fields": map[string]any{
			"日期": dayMs, "站点": row["domain"],
			"失败任务数": parseNum(row["fails"]), "失败率": parseNum(row["pct"]),
			"失败URL数": parseNum(row["urls"]), "诊断": row["advice"],
		}})
	}
	if err := bitableBatchCreate(ctx, token, cfg.SitesTableID, records); err != nil {
		hlog.CtxErrorf(ctx, "[sites-sink] create: %v", err)
		return
	}
	hlog.CtxInfof(ctx, "[sites-sink] %d sites written for %s", len(records), day.Format("2006-01-02"))
}

// syncIssueTracker 告警自动开卡。匹配键=关联指标Key（勾稽类告警无 Key，暂不跟踪）。
func syncIssueTracker(ctx context.Context, day time.Time, results []*report.Result) {
	cfg := conf.GetConfig().MetricsSink
	if cfg.BitableAppToken == "" || cfg.IssuesTableID == "" {
		return
	}
	// 当日命中：按指标 Key 去重，保留最高级别
	type hitInfo struct {
		msg   string
		level report.Level
	}
	todays := map[string]hitInfo{}
	for _, r := range results {
		if r == nil {
			continue
		}
		for _, h := range r.Hits {
			if h.MetricKey == "" {
				continue
			}
			if old, ok := todays[h.MetricKey]; !ok || h.Level > old.level {
				todays[h.MetricKey] = hitInfo{msg: h.Msg, level: h.Level}
			}
		}
	}
	if len(todays) == 0 {
		return
	}
	token, err := tenantToken(ctx, cfg.FeishuAppID, cfg.FeishuAppSecret)
	if err != nil {
		hlog.CtxErrorf(ctx, "[issue-sync] token: %v", err)
		return
	}

	// 未解决的存量问题：关联指标Key → record_id
	open := map[string]string{}
	searchURL := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/search?page_size=500",
		feishuOpenBase, cfg.BitableAppToken, cfg.IssuesTableID)
	resp, err := feishuCall(ctx, "POST", searchURL, token, map[string]any{
		"filter": map[string]any{"conjunction": "or", "conditions": []map[string]any{
			{"field_name": "状态", "operator": "is", "value": []string{"观察中"}},
			{"field_name": "状态", "operator": "is", "value": []string{"处理中"}},
		}},
	})
	if err != nil {
		hlog.CtxErrorf(ctx, "[issue-sync] search: %v", err)
		return
	}
	var data struct {
		Items []struct {
			RecordID string         `json:"record_id"`
			Fields   map[string]any `json:"fields"`
		} `json:"items"`
	}
	_ = sonic.Unmarshal(resp.Data, &data)
	for _, it := range data.Items {
		if k := textOf(it.Fields["关联指标Key"]); k != "" {
			open[k] = it.RecordID
		}
	}

	dayMs := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, cstLoc).UnixMilli()
	levelIcon := func(l report.Level) string {
		if l >= report.LevelCrit {
			return "🔴"
		}
		return "🟡"
	}
	var creates []map[string]any
	var updates []map[string]any
	for key, h := range todays {
		if rid, ok := open[key]; ok {
			updates = append(updates, map[string]any{"record_id": rid, "fields": map[string]any{
				"问题描述": h.msg, "最近出现": dayMs, "当前级别": levelIcon(h.level),
			}})
		} else {
			creates = append(creates, map[string]any{"fields": map[string]any{
				"问题描述": h.msg, "关联指标Key": key, "首次发现": dayMs, "最近出现": dayMs,
				"状态": "观察中", "当前级别": levelIcon(h.level),
			}})
		}
	}
	if len(creates) > 0 {
		if err := bitableBatchCreate(ctx, token, cfg.IssuesTableID, creates); err != nil {
			hlog.CtxErrorf(ctx, "[issue-sync] create: %v", err)
		}
	}
	if len(updates) > 0 {
		url := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/batch_update",
			feishuOpenBase, cfg.BitableAppToken, cfg.IssuesTableID)
		if _, err := feishuCall(ctx, "POST", url, token, map[string]any{"records": updates}); err != nil {
			hlog.CtxErrorf(ctx, "[issue-sync] update: %v", err)
		}
	}
	hlog.CtxInfof(ctx, "[issue-sync] %s: %d new, %d refreshed", day.Format("2006-01-02"), len(creates), len(updates))
}

// textOf 取多维表格文本字段值（可能是字符串或 [{text,type}] 数组）。
func textOf(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var b strings.Builder
		for _, e := range t {
			if m, ok := e.(map[string]any); ok {
				if s, ok := m["text"].(string); ok {
					b.WriteString(s)
				}
			}
		}
		return b.String()
	}
	return ""
}

// bitableBatchCreate 批量写记录（沿用 metrics_sink 的口径）。
func bitableBatchCreate(ctx context.Context, token, tableID string, records []map[string]any) error {
	cfg := conf.GetConfig().MetricsSink
	for i := 0; i < len(records); i += 400 {
		j := min(i+400, len(records))
		url := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/batch_create",
			feishuOpenBase, cfg.BitableAppToken, tableID)
		if _, err := feishuCall(ctx, "POST", url, token, map[string]any{"records": records[i:j]}); err != nil {
			return err
		}
	}
	return nil
}

// deleteDayRecordsIn 删除指定表中某日期的全部记录（幂等重写用，同 metrics_sink 逻辑）。
func deleteDayRecordsIn(ctx context.Context, token, tableID string, dayMs int64) error {
	cfg := conf.GetConfig().MetricsSink
	searchURL := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/search?page_size=500",
		feishuOpenBase, cfg.BitableAppToken, tableID)
	filter := map[string]any{"filter": map[string]any{
		"conjunction": "and",
		"conditions": []map[string]any{{
			"field_name": "日期", "operator": "is",
			"value": []string{"ExactDate", fmt.Sprintf("%d", dayMs)},
		}},
	}}
	for {
		resp, err := feishuCall(ctx, "POST", searchURL, token, filter)
		if err != nil {
			return err
		}
		var data struct {
			Items []struct {
				RecordID string `json:"record_id"`
			} `json:"items"`
			HasMore bool `json:"has_more"`
		}
		_ = sonic.Unmarshal(resp.Data, &data)
		if len(data.Items) == 0 {
			return nil
		}
		ids := make([]string, 0, len(data.Items))
		for _, it := range data.Items {
			ids = append(ids, it.RecordID)
		}
		delURL := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/batch_delete",
			feishuOpenBase, cfg.BitableAppToken, tableID)
		if _, err := feishuCall(ctx, "POST", delURL, token, map[string]any{"records": ids}); err != nil {
			return err
		}
		if !data.HasMore {
			return nil
		}
	}
}
