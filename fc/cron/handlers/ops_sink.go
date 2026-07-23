package handlers

// 运维数据落表（第二优先三件套的 ③④）：
//   sinkFailedSites  失败站点日志表——每天 Top 失败站点入表，站点故障史可查询（换源决策依据）
//   syncIssueTracker 问题跟踪表——告警自动开卡：新告警建记录，重复告警更新"最近出现/当前级别"，
//                    不自动关闭（是否解决由人判断），持续天数=最近出现−首次发现，一眼看出挂了多久
// 均为旁路：失败只记日志，不影响发报。

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"empyrean_lens/conf"
	"empyrean_lens/fc/cron/handlers/report"
	slshttp "empyrean_lens/http"

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

// ─── 失败文章明细（2.2 失败数字的下钻层） ─────────────────────────────────────

// failDetailStages 四个关键失败环节；每环节每日最多采样 failDetailCap 条。
var failDetailStages = []struct{ stage, display string }{
	{"ResourceCrawled", "正文抓取失败"},
	{"ResourceParsed", "内容解析失败"},
	{"AbstractGenerated", "摘要生成失败"},
	{"Validated", "字段校验失败"},
}

const failDetailCap = 150

var bizRe = regexp.MustCompile(`__biz=([^&"]+)`)

// artInfo 资源库回查结果：verify=HTML 含微信验证页标记；isVideo/isAudio/isPay 为消息类型标记；
// escaped=清博 \xNN 转义字面残留（unescape 未生效）；htmlLen 区分空体与海报体。
type artInfo struct {
	title, url, author              string
	verify, isVideo, isAudio, isPay bool
	escaped                         bool
	htmlLen, contentLen, imgs       float64
}

var hostRe = regexp.MustCompile(`^https?://([^/]+)`)

func hostOfURL(u string) string {
	if m := hostRe.FindStringSubmatch(u); m != nil {
		return m[1]
	}
	return ""
}

// classifyFailReason 把失败原因归成可分组的细类（与 2.2 失败原因 Top 的口径一致）。
func classifyFailReason(stage, reason, host string, ai artInfo) string {
	lr := strings.ToLower(reason)
	switch stage {
	case "内容解析失败":
		if strings.Contains(lr, "worthless") {
			if host == "mp.weixin.qq.com" {
				switch {
				case ai.verify:
					return "微信验证页(确认)"
				case ai.isPay:
					return "付费文章(预览页)"
				case ai.isVideo:
					return "视频消息"
				case ai.isAudio:
					return "音频消息"
				case ai.escaped:
					return "转义损坏(清博)" // 字面 \xNN 残留，unescape 未生效
				case ai.htmlLen > 0 && ai.contentLen >= 5:
					return "超短文(阈值误杀)" // 推送带完整短正文，被"无意义"字数阈值杀掉
				case ai.htmlLen >= 1000 || ai.imgs > 0:
					// 大 HTML 无文字，或空壳但供应商带了图片列表——都是图文/海报体
					return "微信空文(纯图/视频)"
				case ai.htmlLen > 0:
					// 空壳且无图无文：分享卡片或被供应商推空的文字文章，需重抓定性
					return "空推送(原文类型未知)"
				}
				return "微信验证页(疑似)" // 无 HTML 存档，无法定性
			}
			return "空壳页(无正文)"
		}
		if strings.Contains(lr, "504") || strings.Contains(lr, "timeout") {
			return "解析服务超时"
		}
	case "正文抓取失败":
		if strings.Contains(lr, "pdf") || strings.Contains(lr, "status_code: 400") {
			return "PDF下载失败"
		}
		if strings.Contains(lr, "timeout") {
			return "抓取超时"
		}
	case "摘要生成失败":
		if strings.Contains(lr, "timeout") || strings.Contains(lr, "read tcp") || strings.Contains(lr, "dial tcp") {
			return "模型网关超时"
		}
	case "字段校验失败":
		if strings.Contains(reason, "Title") {
			return "标题缺失"
		}
	}
	return "其他"
}

// sinkFailedArticles 把四个关键环节的失败文章明细写入失败文章明细表：
// SLS 错误日志给 entry_id/环节/原因 → 资源库回查标题/URL/作者 → biz 从 URL 提取。
// 报表「失败原因 Top」的次数列即链到此表。同日先删后写幂等。
func sinkFailedArticles(ctx context.Context, day time.Time) {
	cfg := conf.GetConfig().MetricsSink
	if cfg.BitableAppToken == "" || cfg.ArticlesTableID == "" {
		return
	}
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, cstLoc)
	to := from.AddDate(0, 0, 1)

	type failInfo struct{ stage, source, reason string }
	entries := map[string]failInfo{} // entry_id → info
	var order []string               // 保持环节分组顺序
	for _, st := range failDetailStages {
		sql := fmt.Sprintf(`ResourceProcessor and error | select regexp_extract(message, 'entry_id: ([a-f0-9]+)', 1) as eid, regexp_extract(message, 'source: ([A-Za-z]+)', 1) as src, substr(regexp_extract(message, 'handle resource error: (.*) lastest status', 1), 1, 80) as reason from log where message like '%%handle resource error%%' and message like '%%lastest status: %s cost%%' limit %d`, st.stage, failDetailCap)
		resp, err := slshttp.SlsQuery(ctx, slshttp.LogStore_BusinessPod, sql, from, to, failDetailCap)
		if err != nil {
			hlog.CtxErrorf(ctx, "[articles-sink] query %s: %v", st.stage, err)
			continue
		}
		for _, row := range resp.Logs {
			eid := row["eid"]
			if eid == "" {
				continue
			}
			if _, dup := entries[eid]; !dup {
				entries[eid] = failInfo{stage: st.display, source: row["src"], reason: row["reason"]}
				order = append(order, eid)
			}
		}
	}
	if len(order) == 0 {
		return
	}

	// 资源库回查标题/URL/作者 + 消息类型定性（100 个一批）
	arts := map[string]artInfo{}
	for i := 0; i < len(order); i += 100 {
		j := min(i+100, len(order))
		origHTML := map[string]any{"$ifNull": []any{"$orig_html", ""}}
		has := func(marker string) map[string]any { // HTML 是否含类型标记
			return map[string]any{"$gt": []any{map[string]any{"$indexOfCP": []any{origHTML, marker}}, -1}}
		}
		resp, err := slshttp.MongoDbAggregate(ctx, &slshttp.MongoAggregateReq{
			DbName: "lingowhale_plugin", CollectionName: "resource",
			Pipeline: []map[string]any{
				{"$match": map[string]any{"_id": map[string]any{"$in": order[i:j]}}},
				{"$project": map[string]any{
					"title": 1, "orig_url": 1, "author_name": 1,
					"verify":      has("secitptpage"),
					"content_len": map[string]any{"$strLenCP": map[string]any{"$ifNull": []any{"$content", ""}}},
					"imgs":        map[string]any{"$size": map[string]any{"$ifNull": []any{"$source_multimedia.img_urls", []any{}}}},
					"is_video": map[string]any{"$or": []any{has("video_iframe"), has("t=pages/video")}},
					"is_audio": map[string]any{"$or": []any{has("mpaudio"), has("plain-music")}},
					"is_pay":   has("pay_preview"),
					"escaped":  has(`\x3c`),
					"html_len": map[string]any{"$strLenCP": origHTML},
				}},
			},
		})
		if err != nil {
			hlog.CtxErrorf(ctx, "[articles-sink] mongo enrich: %v", err)
			continue
		}
		for _, doc := range resp.Data {
			id := ""
			if oid, ok := doc["_id"].(map[string]any); ok {
				id, _ = oid["$oid"].(string)
			} else if s, ok := doc["_id"].(string); ok {
				id = s
			}
			title, _ := doc["title"].(string)
			u, _ := doc["orig_url"].(string)
			author, _ := doc["author_name"].(string)
			verify, _ := doc["verify"].(bool)
			isVideo, _ := doc["is_video"].(bool)
			isAudio, _ := doc["is_audio"].(bool)
			isPay, _ := doc["is_pay"].(bool)
			escaped, _ := doc["escaped"].(bool)
			htmlLen, _ := doc["html_len"].(float64)
			contentLen, _ := doc["content_len"].(float64)
			imgs, _ := doc["imgs"].(float64)
			arts[id] = artInfo{title: title, url: u, author: author, verify: verify,
				isVideo: isVideo, isAudio: isAudio, isPay: isPay, escaped: escaped,
				htmlLen: htmlLen, contentLen: contentLen, imgs: imgs}
		}
	}

	chName := map[string]string{"Renminwang": "人民网", "Qingbo": "清博", "Subscription": "自有RSS/网站", "FromMonitoring": "自采集"}
	token, err := tenantToken(ctx, cfg.FeishuAppID, cfg.FeishuAppSecret)
	if err != nil {
		hlog.CtxErrorf(ctx, "[articles-sink] token: %v", err)
		return
	}
	dayMs := from.UnixMilli()
	if err := deleteDayRecordsIn(ctx, token, cfg.ArticlesTableID, dayMs); err != nil {
		hlog.CtxErrorf(ctx, "[articles-sink] delete old: %v", err)
		return
	}
	records := make([]map[string]any, 0, len(order))
	for _, eid := range order {
		fi, ai := entries[eid], arts[eid]
		ch := chName[fi.source]
		if ch == "" {
			ch = "其他"
		}
		biz := ""
		if m := bizRe.FindStringSubmatch(ai.url); m != nil {
			biz = m[1]
		}
		host := hostOfURL(ai.url)
		fields := map[string]any{
			"日期": dayMs, "环节": fi.stage, "渠道": ch, "标题": ai.title,
			"作者/账号": ai.author, "biz": biz, "域名": host,
			"原因分类": classifyFailReason(fi.stage, fi.reason, host, ai),
			"失败原因": fi.reason, "entry_id": eid,
		}
		if ai.url != "" {
			fields["URL"] = map[string]any{"link": ai.url, "text": ai.url}
		}
		records = append(records, map[string]any{"fields": fields})
	}
	if err := bitableBatchCreate(ctx, token, cfg.ArticlesTableID, records); err != nil {
		hlog.CtxErrorf(ctx, "[articles-sink] create: %v", err)
		return
	}
	hlog.CtxInfof(ctx, "[articles-sink] %d failed articles written for %s", len(records), day.Format("2006-01-02"))

	// 最终状态：当日行写入即标一次，前一日行再刷一遍（给重试/补偿留满窗口）
	annotateFinalStatus(ctx, token, day)
	annotateFinalStatus(ctx, token, day.AddDate(0, 0, -1))
}

// annotateFinalStatus 按 entry_id 回查资源库当前状态，回填「最终状态」列。
// 口径以落库与否为准：99=Ready→已入库；-11=判重死→重复已有（内容经他源已入库，语鲸可见）；
// 其余负数→未入库；正数中间态→处理中。
func annotateFinalStatus(ctx context.Context, token string, day time.Time) {
	cfg := conf.GetConfig().MetricsSink
	dayMs := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, cstLoc).UnixMilli()
	searchURL := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/search?page_size=500",
		feishuOpenBase, cfg.BitableAppToken, cfg.ArticlesTableID)
	rid2eid := map[string]string{}
	pageToken := ""
	for {
		url := searchURL
		if pageToken != "" {
			url += "&page_token=" + pageToken
		}
		resp, err := feishuCall(ctx, "POST", url, token, map[string]any{
			"filter": map[string]any{"conjunction": "and", "conditions": []map[string]any{{
				"field_name": "日期", "operator": "is",
				"value": []string{"ExactDate", fmt.Sprintf("%d", dayMs)},
			}}},
		})
		if err != nil {
			hlog.CtxErrorf(ctx, "[articles-status] search %s: %v", day.Format("01-02"), err)
			return
		}
		var data struct {
			Items []struct {
				RecordID string         `json:"record_id"`
				Fields   map[string]any `json:"fields"`
			} `json:"items"`
			HasMore   bool   `json:"has_more"`
			PageToken string `json:"page_token"`
		}
		_ = sonic.Unmarshal(resp.Data, &data)
		for _, it := range data.Items {
			if eid := textOf(it.Fields["entry_id"]); eid != "" {
				rid2eid[it.RecordID] = eid
			}
		}
		if !data.HasMore {
			break
		}
		pageToken = data.PageToken
	}
	if len(rid2eid) == 0 {
		return
	}

	// 资源库批量查当前状态
	eids := make([]string, 0, len(rid2eid))
	seen := map[string]bool{}
	for _, eid := range rid2eid {
		if !seen[eid] {
			seen[eid] = true
			eids = append(eids, eid)
		}
	}
	status := map[string]float64{}
	for i := 0; i < len(eids); i += 100 {
		j := min(i+100, len(eids))
		resp, err := slshttp.MongoDbAggregate(ctx, &slshttp.MongoAggregateReq{
			DbName: "lingowhale_plugin", CollectionName: "resource",
			Pipeline: []map[string]any{
				{"$match": map[string]any{"_id": map[string]any{"$in": eids[i:j]}}},
				{"$project": map[string]any{"status": 1}},
			},
		})
		if err != nil {
			hlog.CtxErrorf(ctx, "[articles-status] mongo: %v", err)
			return
		}
		for _, doc := range resp.Data {
			id := ""
			if oid, ok := doc["_id"].(map[string]any); ok {
				id, _ = oid["$oid"].(string)
			}
			st, _ := doc["status"].(float64)
			status[id] = st
		}
	}
	verdict := func(st float64, known bool) string {
		switch {
		case !known:
			return "❌未入库"
		case st == 99:
			return "✅已入库"
		case st == -11:
			return "♻️重复已有"
		case st > 0:
			return "⏳处理中"
		default:
			return "❌未入库"
		}
	}
	var updates []map[string]any
	for rid, eid := range rid2eid {
		st, ok := status[eid]
		updates = append(updates, map[string]any{"record_id": rid,
			"fields": map[string]any{"最终状态": verdict(st, ok)}})
	}
	for i := 0; i < len(updates); i += 400 {
		j := min(i+400, len(updates))
		url := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/batch_update",
			feishuOpenBase, cfg.BitableAppToken, cfg.ArticlesTableID)
		if _, err := feishuCall(ctx, "POST", url, token, map[string]any{"records": updates[i:j]}); err != nil {
			hlog.CtxErrorf(ctx, "[articles-status] update: %v", err)
			return
		}
	}
	hlog.CtxInfof(ctx, "[articles-status] %s: %d rows annotated", day.Format("2006-01-02"), len(updates))
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
