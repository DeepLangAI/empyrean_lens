package handlers

import (
	"strconv"
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/fc/cron/handlers/report"
	"os"
	"testing"
	"time"
)

// 一次性验证：推送接口新旧路径各自的流量与来源 IP（供应商切换进度）。
// 运行：MODE_ENV=dev SCRATCH_SWITCH=1 go test ./fc/cron/handlers/ -run TestScratchUrlSwitch -v -count=1
func TestScratchUrlSwitch(t *testing.T) {
	if os.Getenv("SCRATCH_SWITCH") == "" {
		t.Skip("set SCRATCH_SWITCH=1")
	}
	conf.InitConfig()
	ctx := context.Background()
	q := report.Query{Source: report.SourceSLSNginx, Limit: 20,
		SQL: `wechat_article | select url, client_ip, count(*) as total, count_if(status = 200) as ok from log where url in ('/api/feed/v1/resource/wechat_article/add', '/api/resource/v1/wechat_article/add') group by url, client_ip order by total desc limit 20`}

	now := time.Now().In(cstLoc)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cstLoc)
	windows := map[string][2]time.Time{
		"昨天全天":   {midnight.AddDate(0, 0, -1), midnight},
		"今天至今":   {midnight, now},
		"最近30分钟": {now.Add(-30 * time.Minute), now},
	}
	for name, w := range windows {
		rows, err := queryFunc(ctx, q, w[0], w[1])
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		t.Logf("── %s", name)
		for _, r := range rows {
			t.Logf("   %-45s %-16s total=%s ok=%s", r["url"], r["client_ip"], r["total"], r["ok"])
		}
	}
}

// 一次性验证：新路径推来的文章是否正常走完处理链路（网关→汇聚点→入库管线）。
// 运行：MODE_ENV=dev SCRATCH_SWITCH=1 go test ./fc/cron/handlers/ -run TestScratchNewPathPipeline -v -count=1
func TestScratchNewPathPipeline(t *testing.T) {
	if os.Getenv("SCRATCH_SWITCH") == "" {
		t.Skip("set SCRATCH_SWITCH=1")
	}
	conf.InitConfig()
	ctx := context.Background()

	queries := []struct {
		name string
		q    report.Query
	}{
		{"网关（按路径×IP）", report.Query{Source: report.SourceSLSNginx, Limit: 20,
			SQL: `wechat_article | select url, client_ip, count(*) as total, count_if(status = 200) as ok from log where url in ('/api/feed/v1/resource/wechat_article/add', '/api/resource/v1/wechat_article/add') group by url, client_ip order by total desc limit 20`}},
		{"严格转发成功（add_wechat_article Respcode）", report.Query{Source: report.SourceSLS, Limit: 1,
			SQL: `ResponseRath and add_wechat_article | select count(*) as total, count_if(message not like '%Respcode:0,%') as fail from log where message like '%add_wechat_article%'`}},
		{"汇聚点（/resource/add 按 source）", report.Query{Source: report.SourceSLS, Limit: 10,
			SQL: `RequestRout and resource and add | select regexp_extract(message, '"source":([0-9]+)', 1) as src, count(*) as cnt from log where message like '%RequestRout:/iapi/resource/v1/resource/add,%' group by src order by cnt desc limit 10`}},
		{"入库管线（end process 按 source×status）", report.Query{Source: report.SourceSLS, Limit: 20,
			SQL: `ResourceProcessor | select regexp_extract(message, 'source: ([A-Za-z]+)', 1) as source, regexp_extract(message, 'status:(\w+)', 1) as status, count(*) as cnt from log where message like '%end process resource%' group by source, status order by source limit 20`}},
		{"管线失败阶段（handle resource error 按 source×stage）", report.Query{Source: report.SourceSLS, Limit: 20,
			SQL: `ResourceProcessor and error | select regexp_extract(message, 'source: ([A-Za-z]+)', 1) as source, regexp_extract(message, 'lastest status: ([A-Za-z]+)', 1) as stage, count(*) as cnt from log where message like '%handle resource error%' and message like '%source: Renminwang%' or message like '%handle resource error%' and message like '%source: Qingbo%' group by source, stage order by source, cnt desc limit 20`}},
	}

	now := time.Now().In(cstLoc)
	windows := []struct {
		name     string
		from, to time.Time
	}{
		{"最近1小时", now.Add(-time.Hour), now},
		{"昨天同时段", now.Add(-time.Hour).AddDate(0, 0, -1), now.AddDate(0, 0, -1)},
	}
	for _, w := range windows {
		t.Logf("════ %s（%s ~ %s）", w.name, w.from.Format("15:04"), w.to.Format("15:04"))
		for _, item := range queries {
			rows, err := queryFunc(ctx, item.q, w.from, w.to)
			if err != nil {
				t.Errorf("%s: %v", item.name, err)
				continue
			}
			t.Logf("── %s", item.name)
			for _, r := range rows {
				t.Logf("   %v", r)
			}
		}
	}
}

// 按 trace_id 捞原始日志（近 2 天窗口）。
// 运行：MODE_ENV=dev SCRATCH_TRACE=<trace_id> go test ./fc/cron/handlers/ -run TestScratchTrace -v -count=1
func TestScratchTrace(t *testing.T) {
	tid := os.Getenv("SCRATCH_TRACE")
	if tid == "" {
		t.Skip("set SCRATCH_TRACE=<trace_id>")
	}
	conf.InitConfig()
	ctx := context.Background()
	now := time.Now().In(cstLoc)
	q := report.Query{Source: report.SourceSLS, Limit: 50,
		SQL: tid + ` | select __time__ as ts, message from log order by ts limit 50`}
	rows, err := queryFunc(ctx, q, now.AddDate(0, 0, -2), now)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	t.Logf("rows=%d", len(rows))
	for _, r := range rows {
		msg := r["message"]
		if len(msg) > 3000 {
			msg = msg[:3000] + "…"
		}
		t.Logf("[%s] %s", r["ts"], msg)
	}
}

// 按 trace_id 服务端抽取 provider 字段（消息体太长会被采集截断，抽取在 SLS 侧做）。
// 运行：MODE_ENV=dev SCRATCH_TRACE=<trace_id> go test ./fc/cron/handlers/ -run TestScratchTraceProvider -v -count=1
func TestScratchTraceProvider(t *testing.T) {
	tid := os.Getenv("SCRATCH_TRACE")
	if tid == "" {
		t.Skip("set SCRATCH_TRACE=<trace_id>")
	}
	conf.InitConfig()
	ctx := context.Background()
	now := time.Now().In(cstLoc)
	q := report.Query{Source: report.SourceSLS, Limit: 10,
		SQL: tid + ` | select regexp_extract(message, '"provider":\s*([0-9]+)', 1) as provider, regexp_extract(message, '"url":\s*"([^"]+)"', 1) as url from log where message like '%RequestRout:/api/%wechat_article/add,%' limit 10`}
	rows, err := queryFunc(ctx, q, now.AddDate(0, 0, -2), now)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	for _, r := range rows {
		t.Logf("provider=%s url=%s", r["provider"], r["url"])
	}
}

// 交叉验证：SendTopicMsg 的 MsgType 分布应与网关按 IP 的供应商量级对齐。
// 运行：MODE_ENV=dev SCRATCH_SWITCH=1 go test ./fc/cron/handlers/ -run TestScratchMsgTypeVolume -v -count=1
func TestScratchMsgTypeVolume(t *testing.T) {
	if os.Getenv("SCRATCH_SWITCH") == "" {
		t.Skip("set SCRATCH_SWITCH=1")
	}
	conf.InitConfig()
	ctx := context.Background()
	now := time.Now().In(cstLoc)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cstLoc)
	q := report.Query{Source: report.SourceSLS, Limit: 10,
		// HongmaiArticle 字段是 ResourceInnerMsg 独有的，用它锁定这一种消息
		SQL: `SendTopicMsg and HongmaiArticle | select regexp_extract(message, '"MsgType":([0-9]+)', 1) as msg_type, count(*) as cnt from log where message like '%SendTopicMsg%' and message like '%HongmaiArticle%' group by msg_type order by cnt desc limit 10`}
	rows, err := queryFunc(ctx, q, midnight.AddDate(0, 0, -1), midnight)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	for _, r := range rows {
		t.Logf("MsgType=%s cnt=%s", r["msg_type"], r["cnt"])
	}
}

// 对账：spider 库 article 表昨天各口径计数（push_time / publish_time / create_time）。
// 运行：MODE_ENV=dev SCRATCH_SPIDER=2026-07-12 go test ./fc/cron/handlers/ -run TestScratchSpiderCount -v -count=1
func TestScratchSpiderCount(t *testing.T) {
	dayStr := os.Getenv("SCRATCH_SPIDER")
	if dayStr == "" {
		t.Skip("set SCRATCH_SPIDER=YYYY-MM-DD")
	}
	conf.InitConfig()
	ctx := context.Background()
	day, _ := time.ParseInLocation("2006-01-02", dayStr, cstLoc)
	from, to := day, day.Add(24*time.Hour)

	cases := map[string]string{
		"push_time 当日":    `[{"$match": {"push_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$count": "n"}]`,
		"publish_time 当日": `[{"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$count": "n"}]`,
		"create_time 当日":  `[{"$match": {"create_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$count": "n"}]`,
		"publish当日且无push":  `[{"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}, "push_time": null}}, {"$count": "n"}]`,
		"样例字段":            `[{"$sort": {"_id": -1}}, {"$limit": 1}]`,
	}
	for name, pj := range cases {
		rows, err := queryFunc(ctx, report.Query{Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", PipelineJSON: pj}, from, to)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if name == "样例字段" {
			for _, r := range rows {
				keys := make([]string, 0, len(r))
				for k := range r {
					keys = append(keys, k)
				}
				t.Logf("%s: keys=%v", name, keys)
			}
			continue
		}
		t.Logf("%s: %v", name, rows)
	}
}

// 对账二轮：状态字段过滤 + UTC 自然日窗口。
// 运行：MODE_ENV=dev SCRATCH_SPIDER=2026-07-12 go test ./fc/cron/handlers/ -run TestScratchSpiderCount2 -v -count=1
func TestScratchSpiderCount2(t *testing.T) {
	dayStr := os.Getenv("SCRATCH_SPIDER")
	if dayStr == "" {
		t.Skip("set SCRATCH_SPIDER=YYYY-MM-DD")
	}
	conf.InitConfig()
	ctx := context.Background()
	day, _ := time.ParseInLocation("2006-01-02", dayStr, cstLoc)
	from, to := day, day.Add(24*time.Hour)
	// UTC 自然日 = CST 日 +8h 窗口
	fromUTC, toUTC := day.Add(8*time.Hour), day.Add(32*time.Hour)

	type c struct {
		name     string
		from, to time.Time
		pj       string
	}
	cases := []c{
		{"publish当日 invalid统计", from, to, `[{"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$group": {"_id": {"invalid": "$invalid", "exist": "$exist"}, "n": {"$sum": 1}}}]`},
		{"publish当日 push_result分布", from, to, `[{"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$group": {"_id": "$push_result", "n": {"$sum": 1}}}]`},
		{"push_time UTC自然日", fromUTC, toUTC, `[{"$match": {"push_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$count": "n"}]`},
		{"publish_time UTC自然日", fromUTC, toUTC, `[{"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$count": "n"}]`},
	}
	for _, cc := range cases {
		rows, err := queryFunc(ctx, report.Query{Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", PipelineJSON: cc.pj}, cc.from, cc.to)
		if err != nil {
			t.Errorf("%s: %v", cc.name, err)
			continue
		}
		t.Logf("%s: %v", cc.name, rows)
	}
}

// 对账三轮：update_time 窗口。
// 运行：MODE_ENV=dev SCRATCH_SPIDER=2026-07-12 go test ./fc/cron/handlers/ -run TestScratchSpiderCount3 -v -count=1
func TestScratchSpiderCount3(t *testing.T) {
	dayStr := os.Getenv("SCRATCH_SPIDER")
	if dayStr == "" {
		t.Skip("set SCRATCH_SPIDER=YYYY-MM-DD")
	}
	conf.InitConfig()
	ctx := context.Background()
	day, _ := time.ParseInLocation("2006-01-02", dayStr, cstLoc)
	pj := `[{"$match": {"update_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$count": "n"}]`
	for name, w := range map[string][2]time.Time{
		"update_time CST日": {day, day.Add(24 * time.Hour)},
		"update_time UTC日": {day.Add(8 * time.Hour), day.Add(32 * time.Hour)},
	} {
		rows, err := queryFunc(ctx, report.Query{Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", PipelineJSON: pj}, w[0], w[1])
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		t.Logf("%s: %v", name, rows)
	}
}

// 对账四轮：发布于目标日的文章，按推送时刻分小时累计（找 spider 报告生成时点的截面值）。
// 运行：MODE_ENV=dev SCRATCH_SPIDER=2026-07-12 go test ./fc/cron/handlers/ -run TestScratchSpiderCount4 -v -count=1
func TestScratchSpiderCount4(t *testing.T) {
	dayStr := os.Getenv("SCRATCH_SPIDER")
	if dayStr == "" {
		t.Skip("set SCRATCH_SPIDER=YYYY-MM-DD")
	}
	conf.InitConfig()
	ctx := context.Background()
	day, _ := time.ParseInLocation("2006-01-02", dayStr, cstLoc)
	pj := `[
	  {"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}},
	  {"$group": {"_id": {"$dateToString": {"format": "%m-%d %H", "date": "$push_time", "timezone": "+08:00"}}, "n": {"$sum": 1}}},
	  {"$sort": {"_id": 1}}
	]`
	rows, err := queryFunc(ctx, report.Query{Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", PipelineJSON: pj}, day, day.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	cum := 0.0
	for _, r := range rows {
		n, _ := strconv.ParseFloat(r["n"], 64)
		cum += n
		t.Logf("push@%s  +%.0f  cum=%.0f", r["_id"], n, cum)
	}
}

// 验证 spider 库时间字段的存储时区（对照当前时刻）。
// 运行：MODE_ENV=dev SCRATCH_SPIDER=2026-07-12 go test ./fc/cron/handlers/ -run TestScratchSpiderTZ -v -count=1
func TestScratchSpiderTZ(t *testing.T) {
	if os.Getenv("SCRATCH_SPIDER") == "" {
		t.Skip("set SCRATCH_SPIDER")
	}
	conf.InitConfig()
	ctx := context.Background()
	now := time.Now()
	pj := `[{"$sort": {"push_time": -1}}, {"$limit": 2}, {"$project": {"push_time": 1, "create_time": 1, "publish_time": 1, "title": 1, "_id": 0}}]`
	rows, err := queryFunc(ctx, report.Query{Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", PipelineJSON: pj}, now, now)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	t.Logf("now: CST=%s UTC=%s", now.In(cstLoc).Format("01-02 15:04"), now.UTC().Format("01-02 15:04"))
	for _, r := range rows {
		t.Logf("push_time=%v create_time=%v publish_time=%v title=%.20s", r["push_time"], r["create_time"], r["publish_time"], r["title"])
	}
}

// 对账五轮：按 naive-CST 语义的真自然日窗口，发布日文章按推送时刻累计。
// 运行：MODE_ENV=dev SCRATCH_SPIDER=2026-07-12 go test ./fc/cron/handlers/ -run TestScratchSpiderCount5 -v -count=1
func TestScratchSpiderCount5(t *testing.T) {
	dayStr := os.Getenv("SCRATCH_SPIDER")
	if dayStr == "" {
		t.Skip("set SCRATCH_SPIDER=YYYY-MM-DD")
	}
	conf.InitConfig()
	ctx := context.Background()
	day, _ := time.ParseInLocation("2006-01-02", dayStr, cstLoc)
	// naive-CST 存储：真 CST 自然日边界 = 存储值的 00:00Z（= day+8h 的 UTC 表达）
	from, to := day.Add(8*time.Hour), day.Add(32*time.Hour)
	pj := `[
	  {"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}},
	  {"$group": {"_id": {"$dateToString": {"format": "%m-%d %H", "date": "$push_time"}}, "n": {"$sum": 1}}},
	  {"$sort": {"_id": 1}}
	]`
	rows, err := queryFunc(ctx, report.Query{Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", PipelineJSON: pj}, from, to)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	cum := 0.0
	for _, r := range rows {
		n, _ := strconv.ParseFloat(r["n"], 64)
		cum += n
		t.Logf("push@%s(CST墙钟)  +%.0f  cum=%.0f", r["_id"], n, cum)
	}
}

// 验证 lingowhale.content_info 与 subscription.sub 时间字段的存储时区。
// 运行：MODE_ENV=dev SCRATCH_SPIDER=1 go test ./fc/cron/handlers/ -run TestScratchOtherTZ -v -count=1
func TestScratchOtherTZ(t *testing.T) {
	if os.Getenv("SCRATCH_SPIDER") == "" {
		t.Skip("set SCRATCH_SPIDER")
	}
	conf.InitConfig()
	ctx := context.Background()
	now := time.Now()
	t.Logf("now: CST=%s UTC=%s", now.In(cstLoc).Format("01-02 15:04"), now.UTC().Format("01-02 15:04"))
	for name, qq := range map[string]report.Query{
		"content_info": {Source: report.SourceMongo, DB: "lingowhale", Collection: "content_info",
			PipelineJSON: `[{"$sort": {"create_time": -1}}, {"$limit": 1}, {"$project": {"create_time": 1, "pub_time": 1, "_id": 0}}]`},
	} {
		rows, err := queryFunc(ctx, qq, now, now)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		t.Logf("%s: %v", name, rows)
	}
}
