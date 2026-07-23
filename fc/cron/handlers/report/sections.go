package report

// 本文件是业务口径的唯一集中地（第一层 · 数据来源层）。
// 每条 SQL 均于 2026-07-08 在真实数据上验证并交叉勾稽，验证过程与已知坑见
// docs/daily-report-口径-第一层数据来源.md。改埋点/改口径只动本文件，不碰引擎。
//
// ── 通用铁律（详见文档第 0 节）───────────────────────────────────────────────
//  1. SLS 关键词只做粗筛，判定全部写在 SQL like（分词陷阱：add 命中不了 add_wechat_article）；
//  2. 计数锚点用短日志（ResponseRath/nginx），带全文 body 的 RequestRout 会被采集端截丢；
//  3. 大关键词域的全天聚合可能被 SLS 静默截断，加法列用 VerifyAdditive 拆半自验；
//  4. 量纲必须声明：任务(动作) vs URL(篇) 差 4~30 倍（失败重试膨胀）；
//  5. /resource/add 的 body JSON 无空格（"source":14），send_add_resource_msg 有空格
//     （"source": 11），regexp 一律写 \s*。

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Sections 返回全部节定义（含派生的漏斗节，引擎会最后执行它）。
func Sections() []*Section {
	return []*Section{
		secSupplierPush(),
		secSelfCollect(),
		secSubscription(),
		secImages(),
		secPipelineHealth(),
		secFailureMatrix(),
		secGenService(),
		secEffective(),
		secFunnel(),
	}
}

// resource/add 的 source 枚举（resource 仓库 biz/model/resource/resource.go DataSource）。
var srcNames = map[string]string{
	"1":  "订阅",
	"11": "自采集",
	"12": "人民网",
	"14": "清博",
	"15": "小宇宙",
}

// ─── 1.1 供应商推送 ─────────────────────────────────────────────────────────

func secSupplierPush() *Section {
	return &Section{
		Key:   "supplier",
		Layer: LayerL1,
		Title: "1.1 供应商推送（清博/人民网）",
		Queries: []Query{
			{
				// 全渠道到达 /resource/add 的量（内部统一汇聚点），按 source 归因。
				// 同时为漏斗提供 src1/src11/src15 的到达量。
				Name: "push", Source: SourceSLS, Limit: 10,
				SQL: `RequestRout and resource and add | select regexp_extract(message, '"source":([0-9]+)', 1) as src, count(*) as cnt from log where message like '%RequestRout:/iapi/resource/v1/resource/add,%' group by src order by cnt desc limit 10`,
			},
			{
				// 接收成功率只能在网关层算（不丢日志、能看到非 200）。
				// 供应商↔IP 的对应关系会漂移，这里只按 IP 排序展示，归因以 source 为准。
				// 2026-07 起推送接口迁移 /api/feed/v1/resource → /api/resource/v1，
				// 新旧路径并行期两边都算，旧地址下线后自然只剩新路径。
				Name: "recv", Source: SourceSLSNginx, Limit: 10,
				SQL: `wechat_article | select client_ip, count(*) as total, count_if(status = 200) as ok from log where url in ('/api/feed/v1/resource/wechat_article/add', '/api/resource/v1/wechat_article/add') group by client_ip order by total desc limit 10`,
			},
			{
				// 去重口径两端：受理篇数（接收 handler 日志带 url）与进入处理篇数（/resource/add 带 orig_url）。
				// 次数口径会被供应商重复推送（2026-07 实测人民网同秒推 3~4 份、整体重复率 37%）
				// 和消费重试双重污染，按 URL 去重后才是"文章"量纲。
				Name: "recv_uniq", Source: SourceSLS, Limit: 1, Optional: true,
				SQL: `add wechat article | select count(distinct regexp_extract(message, 'url: (http[^ ,]+)', 1)) as urls from log where message like '%add wechat article%'`,
			},
			{
				Name: "proc_uniq", Source: SourceSLS, Limit: 1, Optional: true,
				SQL: `RequestRout and resource and add | select count(distinct regexp_extract(message, '"orig_url":"([^"]+)"', 1)) as urls from log where message like '%RequestRout:/iapi/resource/v1/resource/add,%' and regexp_extract(message, '"source":([0-9]+)', 1) in ('12', '14')`,
			},
			{
				// 推送时效：body 自带 pub_time（unix 秒）。样本有偏（长文的请求日志易被截丢，
				// 覆盖率 ~72%），根治需埋点：让 ResponseRath 带 pub_time+source。
				Name: "lat", Source: SourceSLS, Limit: 10,
				SQL: `RequestRout and resource and add | select regexp_extract(message, '"source":([0-9]+)', 1) as src, round(approx_percentile(greatest(__time__ - cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint), 0), 0.50)/60.0, 1) as p50, round(approx_percentile(greatest(__time__ - cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint), 0), 0.90)/60.0, 1) as p90, round(approx_percentile(greatest(__time__ - cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint), 0), 0.99)/60.0, 1) as p99 from log where message like '%/iapi/resource/v1/resource/add%' and cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint) > 0 group by src order by src limit 10`,
			},
		},
		Extract: extractSupplier,
		Thresholds: []Threshold{
			{
				// 当前 1.2~1.5% ≈ 全部真实丢失（跨零点尾巴实测仅 ~8 篇/日）：同秒重复副本
				// 遇网络抖动一起死 + MNS 消费失败无重投 → 每天 ~1,400 篇从未进入处理。
				// 阈值 1% 让它在修复前保持黄牌（2026-07-23 拍板：1,300+/日不该安静）；
				// 后端修复重投后应归零，>1% 复发即报。
				MetricKey: "supplier.unprocessed.rate",
				Eval: func(cur float64, prev *float64) Level {
					if cur > 3 {
						return LevelCrit
					}
					if cur > 1 {
						return LevelWarn
					}
					return LevelOK
				},
				Msg: "供应商文章未进入处理 %s（URL 去重，基本为真实丢失）——根因为消费失败无重投（跟踪卡处理中），修复后本值应归零；恶化查 api-inner 连通性",
			},
			{
				// 人民网同秒重复推送 bug 的观测指标；基线 ~37%，显著抬升说明上游恶化
				MetricKey: "supplier.dup.rate",
				Eval: func(cur float64, prev *float64) Level {
					if cur > 55 {
						return LevelWarn
					}
					return LevelOK
				},
				Msg: "供应商推送重复率 %s（基线 ~37%%，主要为人民网同秒多份）——上游发送端异常恶化",
			},
			dropOrZero("supplier.arrive.12", "人民网推送量异常：%s（较昨日与上周同日均跌超 30% 或归零）"),
			dropOrZero("supplier.arrive.14", "清博推送量异常：%s（较昨日与上周同日均跌超 30% 或归零）"),
			recvRateThreshold("12", "人民网"),
			recvRateThreshold("14", "清博"),
		},

	}
}

func extractSupplier(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}
	// 内部归因量（/resource/add 按 source）：作为"内部转发成功"的分子
	//（严格的转发成功是 add_wechat_article 的 Respcode:0，但该日志无供应商标识；
	//   归因量与其差 <0.01%，且可按供应商拆分，采用归因量。）
	push := map[string]float64{}
	var addTotal float64
	for _, row := range r["push"] {
		push[row["src"]] = num(row["cnt"])
		addTotal += num(row["cnt"])
		out.Metrics = append(out.Metrics, Metric{
			Key: "supplier.push." + row["src"], Display: srcNames[row["src"]] + "入系统", Value: num(row["cnt"]),
			Text: fmtI(num(row["cnt"])), Dimension: DimDoc,
		})
	}
	out.Metrics = append(out.Metrics,
		Metric{Key: "supplier.push.suppliers", Display: "供应商入系统合计", Value: push["12"] + push["14"], Text: fmtI(push["12"] + push["14"]), Dimension: DimDoc},
		Metric{Key: "resource_add.total", Display: "汇聚点到达总量", Value: addTotal, Text: fmtI(addTotal), Dimension: DimDoc},
	)

	// 网关到达量（推送量口径 = 供应商当日实际到达接收端的请求数）
	supplierIP := map[string]string{
		"61.184.1.10":    "12", // 人民网（IP 变更时更新；未登记 IP 超阈会出注记）
		"14.103.184.222": "14", // 清博
	}
	arrive, okBySrc := map[string]float64{}, map[string]float64{}
	var recvTotal, recvOK, unattributed float64
	for _, row := range r["recv"] {
		total, ok := num(row["total"]), num(row["ok"])
		recvTotal += total
		recvOK += ok
		if src, known := supplierIP[row["client_ip"]]; known {
			arrive[src] += total
			okBySrc[src] += ok
		} else {
			unattributed += total
		}
	}
	out.Metrics = append(out.Metrics,
		Metric{Key: "supplier.recv.total", Display: "供应商网关接收", Value: recvTotal, Text: fmtI(recvTotal), Dimension: DimTask},
		Metric{Key: "supplier.recv.ok", Display: "供应商网关接收成功", Value: recvOK, Text: fmtI(recvOK), Dimension: DimTask},
	)
	// 去重口径（文章量纲）：受理篇数 vs 进入处理篇数。差值≈真实丢失（跨零点尾巴实测仅个位数；
	// 旧的次数口径被人民网重复副本撞锁死亡放大近 5 倍，2026-07-23 定性后弃用告警）。
	recvUniq, procUniq := num(first(r["recv_uniq"])["urls"]), num(first(r["proc_uniq"])["urls"])
	if recvUniq > 0 {
		unproc := recvUniq - procUniq
		if unproc < 0 {
			unproc = 0
		}
		unprocPct := pct(unproc, recvUniq)
		dupRate := pct(recvOK-recvUniq, maxf(recvOK, 1))
		out.Metrics = append(out.Metrics,
			Metric{Key: "supplier.unprocessed.rate", Display: "供应商文章当日未进处理(URL去重)", Value: unprocPct, Text: fmt.Sprintf("%s 篇（%.1f%%）", fmtI(unproc), unprocPct), Dimension: DimPercent},
			Metric{Key: "supplier.dup.rate", Display: "供应商推送重复率", Value: dupRate, Text: fmtPct1(dupRate), Dimension: DimPercent},
		)
		// 展示层注记：已知问题（消费失败零重投）修复前保持每日可见，阈值只管恶化（>3%）
		out.Notes = append(out.Notes, fmt.Sprintf(
			"当日未进处理 %s 篇（%.1f%%，URL 去重口径）≈ 真实丢失，根因为消费失败无重投（跟踪卡处理中）；推送重复率 %.1f%%（人民网同秒多份）",
			fmtI(unproc), unprocPct, dupRate))
	}
	if unattributed > recvTotal*0.005 {
		out.Notes = append(out.Notes, fmt.Sprintf("⚠️ 有 %s 次推送来自未登记 IP（供应商 IP 变更？需更新归属表）", fmtI(unattributed)))
	}

	lat := map[string]map[string]string{}
	for _, row := range r["lat"] {
		lat[row["src"]] = row
	}

	// 模板同款表：供应商/推送量/环比昨日/接收成功率/时效P50/P90/P99/状态
	var rows []map[string]string
	for _, src := range []string{"14", "12"} { // 模板顺序：清博在前
		arrived, entered := arrive[src], push[src]
		// 接收成功率 = 同步受理（nginx 200）÷ 到达：同步动作不受下游队列积压影响，
		// 差值即真实丢失（502 等）。归因量(entered)受积压影响，只用于在途量计算。
		rate := pct(okBySrc[src], maxf(arrived, 1))
		inflight := okBySrc[src] - entered
		if inflight < 0 {
			inflight = 0 // 泄洪日：今日处理量含昨日积压，视为无在途
		}
		out.Metrics = append(out.Metrics,
			Metric{Key: "supplier.arrive." + src, Display: srcNames[src] + "推送量", Value: arrived, Text: fmtI(arrived), Dimension: DimDoc},
			Metric{Key: "supplier.recv.rate." + src, Display: srcNames[src] + "接收成功率", Value: rate, Text: fmtPct2(rate), Dimension: DimPercent},
			Metric{Key: "supplier.inflight." + src, Display: srcNames[src] + "在途积压", Value: inflight, Text: fmtI(inflight), Dimension: DimDoc},
		)
		_ = inflight
		status := "🟢"
		switch {
		case arrived == 0 || rate < 95:
			status = "🔴"
		case rate < 99.5:
			status = "🟡"
		}
		row := map[string]string{
			"supplier": srcNames[src],
			"push":     fmtI(arrived),
			"delta":    DeltaPct(arrived, prev, "supplier.arrive."+src),
			"rate":     fmtPct2(rate),
			"p50":      "—", "p90": "—", "p99": "—",
			"status": status,
		}
		if l, ok := lat[src]; ok {
			row["p50"], row["p90"], row["p99"] = fmtMin(num(l["p50"])), fmtMin(num(l["p90"])), fmtMin(num(l["p99"]))
			out.Metrics = append(out.Metrics,
				Metric{Key: "supplier.lat." + src + ".p50", Display: srcNames[src] + "时效P50", Value: num(l["p50"]), Text: fmtMin(num(l["p50"])), Dimension: DimMinutes},
				Metric{Key: "supplier.lat." + src + ".p90", Display: srcNames[src] + "时效P90", Value: num(l["p90"]), Text: fmtMin(num(l["p90"])), Dimension: DimMinutes},
				Metric{Key: "supplier.lat." + src + ".p99", Display: srcNames[src] + "时效P99", Value: num(l["p99"]), Text: fmtMin(num(l["p99"])), Dimension: DimMinutes},
			)
		}
		rows = append(rows, row)
	}
	out.Tables = append(out.Tables, Table{
		Cols: []TableCol{
			{Name: "supplier", Display: "供应商"}, {Name: "push", Display: "推送量"},
			{Name: "delta", Display: "环比昨日"}, {Name: "rate", Display: "接收成功率"},
			{Name: "p50", Display: "时效P50"}, {Name: "p90", Display: "时效P90"},
			{Name: "p99", Display: "时效P99"}, {Name: "status", Display: "状态"},
		},
		Rows: rows,
	})
	return out, nil
}

func fmtPct2(v float64) string { return fmt.Sprintf("%.2f%%", v) }

// recvRateThreshold：接收成功率（内部转发成功÷到达）。供应商无重试，非 100% 即真实丢失。
func recvRateThreshold(src, name string) Threshold {
	return Threshold{
		MetricKey: "supplier.recv.rate." + src,
		Eval: func(cur float64, prev *float64) Level {
			if cur < 95 {
				return LevelCrit
			}
			if cur < 99.5 {
				return LevelWarn
			}
			return LevelOK
		},
		Msg: name + "接收成功率 %s（供应商无重试，差值即真实丢失）",
	}
}

// ─── 1.2 公众号自采集（wechat_spider） ──────────────────────────────────────

func secSelfCollect() *Section {
	return &Section{
		Key:   "self",
		Layer: LayerL1,
		Title: "1.2 公众号自采集（wechat_spider）",
		Queries: []Query{
			{
				// spider → send_add_resource_msg → P0 常驻队列。该接口事实上 spider 专属，
				// SQL 仍显式锁 source:11 防未来共用污染。注意此 body 的 JSON 冒号后有空格。
				Name: "push", Source: SourceSLS, Limit: 1,
				SQL:            `send_add_resource_msg | select count_if(message like '%RequestRout:/iapi/resource/v1/resource/send_add_resource_msg,%' and message like '%"source": 11%') as push_cnt, count_if(message like '%ResponseRath:/iapi/resource/v1/resource/send_add_resource_msg,%' and message like '%Respcode:0,%') as accept_cnt from log`,
				VerifyAdditive: []string{"push_cnt", "accept_cnt"},
			},
			{
				// spider 库常规链路（回灌脚本不写 article 表，天然不含回灌）：
				// 按 publish_time（发布日）统计，与 spider 侧日报同口径（2026-07-13 拍板）。
				// 注意：发布日口径有补采长尾，报表 08:00 生成时约九成已采到，数值是当时截面，
				// 之后同口径重查会略大。时效分桶 = 发布→推送延迟（旧版 Mongo 无 percentile，分桶插值）。
				Name: "spider", Source: SourceMongo, DB: "wechat-spider", Collection: "article", MongoNaiveCST: true,
				PipelineJSON: `[
				  {"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}},
				  {"$project": {"lag": {"$divide": [{"$subtract": ["$push_time", "$publish_time"]}, 60000]}}},
				  {"$group": {"_id": null, "n": {"$sum": 1},
				    "b15":   {"$sum": {"$cond": [{"$lte": ["$lag", 15]}, 1, 0]}},
				    "b60":   {"$sum": {"$cond": [{"$lte": ["$lag", 60]}, 1, 0]}},
				    "b180":  {"$sum": {"$cond": [{"$lte": ["$lag", 180]}, 1, 0]}},
				    "b720":  {"$sum": {"$cond": [{"$lte": ["$lag", 720]}, 1, 0]}},
				    "b1440": {"$sum": {"$cond": [{"$lte": ["$lag", 1440]}, 1, 0]}}}}
				]`,
			},
			{
				Name: "acct_total", Source: SourceMongo, DB: "wechat-spider", Collection: "target_account", MongoNaiveCST: true,
				PipelineJSON: `[{"$count": "total"}]`,
			},
			{
				// 缺失核验：与语鲸 topic-monitor 监控9 同口径（Feed 接口逐账号核验标题），
				// 实现在 handlers/feedcheck.go。接口故障时降级为无此行（Optional）。
				Name: "feedcheck", Source: SourceCustom, Optional: true,
			},
			{
				// 覆盖视角（对齐 spider 侧日报 tasks/report_task.py，2026-07-13 起由本报表 cover）：
				// 已覆盖 = store_time 非空（语鲸入库时间回写进 article 表），时效 = store_time − publish_time；
				// 未覆盖 = store_time 空（spider 兜底推送），push_result=true 为推送成功；无 push_time 记 0 分钟。
				Name: "cover", Source: SourceMongo, DB: "wechat-spider", Collection: "article", MongoNaiveCST: true,
				PipelineJSON: `[
				  {"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}},
				  {"$project": {
				    "covered": {"$cond": [{"$ifNull": ["$store_time", null]}, 1, 0]},
				    "lag_min": {"$divide": [{"$subtract": [{"$ifNull": ["$store_time", {"$ifNull": ["$push_time", "$publish_time"]}]}, "$publish_time"]}, 60000]},
				    "push_ok": {"$cond": [{"$eq": ["$push_result", true]}, 1, 0]}}},
				  {"$group": {"_id": "$covered", "n": {"$sum": 1},
				    "le1h": {"$sum": {"$cond": [{"$lte": ["$lag_min", 60]}, 1, 0]}},
				    "le3h": {"$sum": {"$cond": [{"$lte": ["$lag_min", 180]}, 1, 0]}},
				    "push_ok": {"$sum": "$push_ok"}}}
				]`,
			},
			{
				Name: "acct_active", Source: SourceMongo, DB: "wechat-spider", Collection: "article", MongoNaiveCST: true,
				PipelineJSON: `[
				  {"$match": {"publish_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}},
				  {"$group": {"_id": "$target_account"}}, {"$count": "active"}
				]`,
			},
			{
				// 正文抓取通道（uid=resource_server 即 resource 为自采文章补抓正文）。
				// URL 量纲：失败重试会把日志条数膨胀 8~33 倍，必须去重。
				Name: "crawl", Source: SourceSLS, Limit: 1,
				SQL: `crawl_normal OR crawl_error | select count_if(has_ok = 1) as ok_urls, count_if(has_err = 1 and has_ok = 0) as net_fail, count_if(has_err = 1 and has_ok = 1) as recovered from (select regexp_extract(extra, '"url":"([^"]*)"', 1) as url, max(case when message = 'crawl_normal' then 1 else 0 end) as has_ok, max(case when message = 'crawl_error' then 1 else 0 end) as has_err from log where extra like '%"uid":"resource_server"%' and extra like '%"domain"%' group by 1) where url is not null and url != ''`,
			},
		},
		Extract: extractSelfCollect,
		Thresholds: []Threshold{
			{
				MetricKey: "self.accept.rate",
				Eval:      func(cur float64, prev *float64) Level { return warnBelow(cur, 100, 99) },
				Msg:       "自采推送受理成功率 %s（推送量−受理量 = 受理失败篇数）",
			},
			{
				MetricKey: "self.backfill_ratio",
				Eval: func(cur float64, prev *float64) Level {
					if cur > 1.5 {
						return LevelWarn
					}
					return LevelOK
				},
				Msg: "自采到达量为常规链路 %s 倍：当日有历史回灌在跑，下游各层环比会连带波动",
			},
			{
				MetricKey: "self.crawl.rate",
				Eval:      func(cur float64, prev *float64) Level { return warnBelow(cur, 90, 80) },
				Msg:       "自采正文抓取成功率 %s（URL 级）",
			},
			{
				MetricKey: "self.missing.rate",
				Eval: func(cur float64, prev *float64) Level {
					if cur > 3 {
						return LevelCrit
					}
					if cur > 1 {
						return LevelWarn
					}
					return LevelOK
				},
				Msg: "监控账号文章缺失率 %s（当日落库文章在订阅 Feed 中查不到，与语鲸监控9同口径）",
			},
		},
	}
}

func extractSelfCollect(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}
	p := first(r["push"])
	pushCnt, accept := num(p["push_cnt"]), num(p["accept_cnt"])
	acceptRate := pct(accept, maxf(pushCnt, 1))
	out.Metrics = append(out.Metrics,
		Metric{Key: "self.push", Display: "自采推送到达", Value: pushCnt, Text: fmtI(pushCnt), Dimension: DimTask},
		Metric{Key: "self.accept", Display: "自采受理", Value: accept, Text: fmtI(accept), Dimension: DimTask},
		Metric{Key: "self.accept.rate", Display: "自采推送受理成功率", Value: acceptRate, Text: fmtPct1(acceptRate), Dimension: DimPercent},
	)

	sp := first(r["spider"])
	regular := num(sp["n"])
	if regular > 0 {
		out.Metrics = append(out.Metrics,
			Metric{Key: "self.regular", Display: "自采按发布日采集量", Value: regular, Text: fmtI(regular), Dimension: DimDoc},
			Metric{Key: "self.backfill_ratio", Display: "到达/常规比", Value: pushCnt / regular, Text: fmt.Sprintf("%.1f", pushCnt/regular), Dimension: DimNone},
		)
		// 分桶边界: 15/60/180/720/1440 分钟，插值出 P50/P90
		bounds := []float64{15, 60, 180, 720, 1440}
		cums := []float64{num(sp["b15"]), num(sp["b60"]), num(sp["b180"]), num(sp["b720"]), num(sp["b1440"])}
		p50 := bucketPercentile(regular, 0.5, bounds, cums)
		p90 := bucketPercentile(regular, 0.9, bounds, cums)
		p99 := bucketPercentile(regular, 0.99, bounds, cums)
		out.Metrics = append(out.Metrics,
			Metric{Key: "self.lat.p50", Display: "自采集时效P50", Value: p50, Text: fmtMin(p50), Dimension: DimMinutes},
			Metric{Key: "self.lat.p90", Display: "自采集时效P90", Value: p90, Text: fmtMin(p90), Dimension: DimMinutes},
			Metric{Key: "self.lat.p99", Display: "自采集时效P99", Value: p99, Text: fmtMin(p99), Dimension: DimMinutes},
		)
	}

	total, active := num(first(r["acct_total"])["total"]), num(first(r["acct_active"])["active"])
	if total > 0 {
		out.Metrics = append(out.Metrics,
			Metric{Key: "self.accounts.total", Display: "公众号监控账号总数", Value: total, Text: fmtI(total), Dimension: DimAccount},
			Metric{Key: "self.accounts.active", Display: "公众号有产出账号", Value: active, Text: fmtI(active), Dimension: DimAccount},
		)
	}

	// 覆盖视角（_id: 1=已覆盖，0=未覆盖）；分桶 le1h/le3h 是累计值
	var covN, covLe1h, covLe3h, uncovN, uncovPushOK float64
	for _, row := range r["cover"] {
		if row["_id"] == "1" {
			covN, covLe1h, covLe3h = num(row["n"]), num(row["le1h"]), num(row["le3h"])
		} else {
			uncovN, uncovPushOK = num(row["n"]), num(row["push_ok"])
		}
	}
	if covN+uncovN > 0 {
		out.Metrics = append(out.Metrics,
			Metric{Key: "self.covered", Display: "自采语鲸已覆盖", Value: covN, Text: fmtI(covN), Dimension: DimDoc},
			Metric{Key: "self.covered.le1h", Display: "自采覆盖时效≤1h占比", Value: pct(covLe1h, maxf(covN, 1)), Text: fmtPct1(pct(covLe1h, maxf(covN, 1))), Dimension: DimPercent},
			Metric{Key: "self.covered.buckets", Display: "自采覆盖时效分桶", Value: covN,
				Text: fmt.Sprintf("≤1h %s ｜ 1-3h %s ｜ >3h %s", fmtI(covLe1h), fmtI(covLe3h-covLe1h), fmtI(covN-covLe3h)), Dimension: DimDoc},
			Metric{Key: "self.uncovered", Display: "自采未覆盖(spider兜底)", Value: uncovN, Text: fmtI(uncovN), Dimension: DimDoc},
			Metric{Key: "self.spider_push.rate", Display: "spider推送成功率", Value: pct(uncovPushOK, maxf(uncovN, 1)), Text: fmtPct1(pct(uncovPushOK, maxf(uncovN, 1))), Dimension: DimPercent},
		)
	}

	// 缺失口径 = 语鲸监控9同款：Feed 接口逐账号核验（可核验 = 进入 + 缺失）
	if fc := first(r["feedcheck"]); fc["checked"] != "" {
		checked, hits, missing := num(fc["checked"]), num(fc["hit"]), num(fc["missing"])
		missingPct := pct(missing, maxf(checked, 1))
		out.Metrics = append(out.Metrics,
			Metric{Key: "self.feed.checked", Display: "自采Feed核验文章数", Value: checked, Text: fmtI(checked), Dimension: DimDoc},
			Metric{Key: "self.feed.hit", Display: "自采进入语鲸(Feed核验)", Value: hits, Text: fmtI(hits), Dimension: DimDoc},
			Metric{Key: "self.missing", Display: "自采缺失数", Value: missing, Text: fmtI(missing), Dimension: DimDoc},
			Metric{Key: "self.missing.rate", Display: "自采缺失率", Value: missingPct, Text: fmtPct1(missingPct), Dimension: DimPercent},
		)
		if errs := num(fc["errs"]); errs > 0 {
			out.Notes = append(out.Notes, fmt.Sprintf("⚠️ Feed 核验有 %s 个账号接口失败（已剔除，不计缺失）", fmtI(errs)))
		}
	}

	c := first(r["crawl"])
	okURLs, netFail := num(c["ok_urls"]), num(c["net_fail"])
	crawlRate := pct(okURLs, maxf(okURLs+netFail, 1))
	out.Metrics = append(out.Metrics,
		Metric{Key: "self.crawl.ok", Display: "自采正文抓取成功", Value: okURLs, Text: fmtI(okURLs), Dimension: DimURL},
		Metric{Key: "self.crawl.fail", Display: "自采正文净失败", Value: netFail, Text: fmtI(netFail), Dimension: DimURL},
		Metric{Key: "self.crawl.rate", Display: "自采正文抓取成功率", Value: crawlRate, Text: fmtPct1(crawlRate), Dimension: DimPercent},
	)
	return out, nil
}

// ─── 1.3 网站/RSS 抓取（订阅渠道，uid=1） ───────────────────────────────────

func secSubscription() *Section {
	return &Section{
		Key:   "sub",
		Layer: LayerL1,
		Title: "1.3 网站/RSS 抓取（订阅渠道）",
		Queries: []Query{
			{
				// 渠道边界按 uid 划分（uid=1 即订阅服务），含 RSS 源产出的微信 URL。
				// 任务量纲（轮询语义：每次轮询=一个任务，本节不去重）+ 内容覆盖（URL 量纲）双轨。
				Name: "tasks", Source: SourceSLS, Limit: 1,
				SQL: `crawl_normal OR crawl_error | select count_if(message = 'crawl_normal') as ok_tasks, count_if(message = 'crawl_error') as fail_tasks, count(distinct case when message = 'crawl_normal' then regexp_extract(extra, '"url":"([^"]*)"', 1) end) as ok_urls, count(distinct case when message = 'crawl_error' then regexp_extract(extra, '"url":"([^"]*)"', 1) end) as fail_urls from log where extra like '%"domain"%' and extra like '%"uid":"1"%'`,
			},
			{
				// Top 失败站点：失败任务/失败率/失败 URL 数三列配合读——
				// 失败率高+URL 多=整站挂；失败率高+URL 少=个别源死循环重试。
				Name: "domains", Source: SourceSLS, Limit: 8,
				SQL: `crawl_normal OR crawl_error | select regexp_extract(extra, '"domain":"([^"]*)"', 1) as domain, count_if(message = 'crawl_error') as fail_tasks, count(*) as total_tasks, round(count_if(message = 'crawl_error') * 100.0 / count(*), 1) as fail_pct, count(distinct case when message = 'crawl_error' then regexp_extract(extra, '"url":"([^"]*)"', 1) end) as fail_urls from log where extra like '%"domain"%' and extra like '%"uid":"1"%' group by domain having count_if(message = 'crawl_error') > 0 order by fail_tasks desc limit 8`,
			},
			{
				// 调度完成率分母：Σ(1440/frequency)。只看成功率发现不了调度器卡死，这个能。
				Name: "sched", Source: SourceMongo, DB: "subscription", Collection: "sub",
				PipelineJSON: `[{"$match": {"status": 1, "frequency": {"$gt": 0}}}, {"$group": {"_id": null, "sources": {"$sum": 1}, "expected": {"$sum": {"$divide": [1440, "$frequency"]}}}}]`,
				Optional:     true,
			},
			{
				// 新文时效（lag≤24h 截断，剔除历史回补长尾）；backfill 单独计数。
				Name: "fresh", Source: SourceSLS, Limit: 1,
				SQL: `RequestRout and resource and add | select count_if(lag_min <= 1440) as fresh_cnt, count_if(lag_min > 1440) as backfill_cnt, round(approx_percentile(case when lag_min <= 1440 then lag_min end, 0.50), 0) as p50, round(approx_percentile(case when lag_min <= 1440 then lag_min end, 0.90), 0) as p90, round(count_if(lag_min <= 240) * 100.0 / greatest(count_if(lag_min <= 1440), 1), 1) as le4h_pct from (select (__time__ - cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint)) / 60.0 as lag_min from log where message like '%/iapi/resource/v1/resource/add%' and message like '%"source":1,%' and cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint) > 0) where lag_min >= 0`,
			},
		},
		Extract: extractSubscription,
		Thresholds: []Threshold{
			{
				// 长期水位就是 75%~80%：调度器批量扫描每 30min 一轮、判定不带容差，
				// A 级源(frequency=30)固定错过隔次扫描，实际 60min 一轮（2026-07 排查定性，业务接受）。
				// 阈值只抓显著恶化（调度停摆/队列积压），不再对慢性缺口天天报警。
				MetricKey: "sub.sched.completion",
				Eval:      func(cur float64, prev *float64) Level { return warnBelow(cur, 55, 40) },
				Msg:       "调度完成率 %s：显著低于长期水位（75%%~80%%），疑似调度停摆或队列积压",
			},
			{
				MetricKey: "sub.task_rate",
				Eval:      func(cur float64, prev *float64) Level { return warnBelow(cur, 88, 75) },
				Msg:       "订阅抓取任务成功率 %s",
			},
			{
				// 单站点疑似整站故障（失败率>80% 且失败源>50 个）进告警区。
				// 多站同挂大概率是我方出口/网络问题，升红。
				MetricKey: "sub.outage.sites",
				Eval: func(cur float64, prev *float64) Level {
					if cur >= 3 {
						return LevelCrit
					}
					if cur >= 1 {
						return LevelWarn
					}
					return LevelOK
				},
				Msg: "订阅源疑似整站故障：%s——影响面大，建议确认站点状态或换源；多站同挂先查我方出口",
			},
		},
	}
}

func extractSubscription(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}
	t := first(r["tasks"])
	okT, failT := num(t["ok_tasks"]), num(t["fail_tasks"])
	taskRate := pct(okT, maxf(okT+failT, 1))
	out.Metrics = append(out.Metrics,
		Metric{Key: "sub.tasks", Display: "自有RSS/网站抓取任务总量", Value: okT + failT, Text: fmtI(okT + failT), Dimension: DimTask},
		Metric{Key: "sub.task_rate", Display: "自有RSS/网站抓取成功率", Value: taskRate, Text: fmtPct1(taskRate), Dimension: DimPercent},
		Metric{Key: "sub.ok_urls", Display: "内容覆盖", Value: num(t["ok_urls"]), Text: fmtI(num(t["ok_urls"])), Dimension: DimURL},
	)

	if s := first(r["sched"]); s != nil {
		expected := num(s["expected"])
		if expected > 0 {
			completion := pct(okT+failT, expected)
			out.Metrics = append(out.Metrics,
				Metric{Key: "sub.sched.completion", Display: "自有RSS/网站调度完成率", Value: completion, Text: fmtPct1(completion), Dimension: DimPercent},
			)
			out.Notes = append(out.Notes, fmt.Sprintf("调度基准：%s 个活跃信源 × 自适应频率 → 期望轮询 %s 次/日", fmtI(num(s["sources"])), fmtI(expected)))
		}
	}

	if f := first(r["fresh"]); f != nil {
		out.Metrics = append(out.Metrics,
			Metric{Key: "sub.fresh.p50", Display: "自有RSS/网站新文时效P50", Value: num(f["p50"]), Text: fmtMin(num(f["p50"])), Dimension: DimMinutes},
			Metric{Key: "sub.fresh.p90", Display: "自有RSS/网站新文时效P90", Value: num(f["p90"]), Text: fmtMin(num(f["p90"])), Dimension: DimMinutes},
			Metric{Key: "sub.fresh.le4h", Display: "自有RSS/网站新文≤4h占比", Value: num(f["le4h_pct"]), Text: fmtPct1(num(f["le4h_pct"])), Dimension: DimPercent},
			Metric{Key: "sub.backfill", Display: "自有RSS/网站历史回补", Value: num(f["backfill_cnt"]), Text: fmtI(num(f["backfill_cnt"])), Dimension: DimDoc},
		)
	}

	// Top 失败站点 + 连续失败天数（读近 7 日快照里的 sub.domfail.<domain>）
	var rows []map[string]string
	var outages []string // 疑似整站故障清单（进告警）
	for _, d := range r["domains"] {
		domain := d["domain"]
		fails := num(d["fail_tasks"])
		streak := Streak(prev, "sub.domfail."+domain, func(v float64) bool { return v > 300 })
		if fails > 300 {
			streak++ // 含今日
		}
		advice := "观察"
		failPct, failURLs := num(d["fail_pct"]), num(d["fail_urls"])
		switch {
		case failPct > 80 && failURLs > 50:
			advice = "疑似整站故障"
			tag := fmt.Sprintf("%s(%s个源 %s)", domain, fmtI(failURLs), fmtPct1(failPct))
			if streak >= 2 {
				tag = fmt.Sprintf("%s(%s个源 %s·连续%d天)", domain, fmtI(failURLs), fmtPct1(failPct), streak)
			}
			outages = append(outages, tag)
		case failPct > 80:
			advice = "个别源失效，建议换源"
		}
		if streak >= 3 {
			advice += fmt.Sprintf("（连续 %d 天）", streak)
		}
		rows = append(rows, map[string]string{
			"domain": domain, "fails": fmtI(fails), "pct": fmtPct1(failPct),
			"urls": fmtI(failURLs), "advice": advice,
		})
		// 失败量入快照，供次日算连续天数
		out.Metrics = append(out.Metrics, Metric{Key: "sub.domfail." + domain, Display: domain + " 失败", Value: fails, Text: fmtI(fails), Dimension: DimTask})
	}
	outageText := "无"
	if len(outages) > 0 {
		outageText = strings.Join(outages, "、")
	}
	out.Metrics = append(out.Metrics, Metric{Key: "sub.outage.sites", Display: "疑似整站故障站点数", Value: float64(len(outages)), Text: outageText, Dimension: DimNone})
	if len(rows) > 0 {
		out.Tables = append(out.Tables, Table{
			Title: "Top 失败站点",
			Cols: []TableCol{
				{Name: "domain", Display: "站点"}, {Name: "fails", Display: "失败任务"},
				{Name: "pct", Display: "失败率"}, {Name: "urls", Display: "失败URL数"},
				{Name: "advice", Display: "诊断"},
			},
			Rows: rows,
		})
	}
	return out, nil
}

// ─── 1.4 图片抓取 ───────────────────────────────────────────────────────────

func secImages() *Section {
	return &Section{
		Key:   "img",
		Layer: LayerL1,
		Title: "1.4 图片抓取",
		Queries: []Query{
			{
				// 任务量纲：end 日志无 img_url，做不了图片级去重（埋点需求：end 带 img_url）。
				Name: "crawl", Source: SourceSLS, Limit: 1,
				SQL:            `async_crawl_img end | select count(*) as total, round(approx_percentile(cast(regexp_extract(message, 'cost: ([0-9.]+)', 1) as double), 0.50), 1) as p50, round(approx_percentile(cast(regexp_extract(message, 'cost: ([0-9.]+)', 1) as double), 0.99), 1) as p99 from log`,
				VerifyAdditive: []string{"total"},
			},
			{
				// 失败日志带 img_url，可去重（同图跨任务反复失败，条数膨胀 ~3.6 倍）。
				Name: "fail", Source: SourceSLS, Limit: 1,
				SQL: `所有方式均失败 | select count(*) as msgs, count(distinct regexp_extract(message, 'img_url=(\S+)', 1)) as uniq_imgs from log`,
			},
			{
				Name: "beds", Source: SourceSLS, Limit: 5, Optional: true,
				SQL: `所有方式均失败 | select regexp_extract(message, 'img_url=https?://([^/]+)', 1) as domain, count(*) as cnt from log group by domain order by cnt desc limit 5`,
			},
		},
		Extract: extractImages,
		Thresholds: []Threshold{
			{
				MetricKey: "img.fail_rate",
				Eval: func(cur float64, prev *float64) Level {
					if cur > 10 {
						return LevelCrit
					}
					if cur > 5 {
						return LevelWarn
					}
					return LevelOK
				},
				Msg: "图片任务失败率 %s",
			},
		},
	}
}

func extractImages(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}
	c, f := first(r["crawl"]), first(r["fail"])
	total, failMsgs := num(c["total"]), num(f["msgs"])
	failRate := pct(failMsgs, maxf(total, 1))
	out.Metrics = append(out.Metrics,
		Metric{Key: "img.total", Display: "图片任务量", Value: total, Text: fmtI(total), Dimension: DimTask},
		Metric{Key: "img.fail_rate", Display: "任务失败率", Value: failRate, Text: fmtPct1(failRate), Dimension: DimPercent},
		Metric{Key: "img.fail_uniq", Display: "失败图片数", Value: num(f["uniq_imgs"]), Text: fmtI(num(f["uniq_imgs"])), Dimension: DimURL},
		Metric{Key: "img.p50", Display: "耗时P50", Value: num(c["p50"]), Text: num2s(c["p50"]) + "s", Dimension: DimSeconds},
		Metric{Key: "img.p99", Display: "耗时P99", Value: num(c["p99"]), Text: num2s(c["p99"]) + "s", Dimension: DimSeconds},
	)
	var rows []map[string]string
	for _, b := range r["beds"] {
		rows = append(rows, map[string]string{"domain": b["domain"], "cnt": fmtI(num(b["cnt"]))})
	}
	if len(rows) > 0 {
		out.Tables = append(out.Tables, Table{
			Compact: true,
			Title: "Top 失败图床（失败尾部含追踪像素，属上游清洗问题非抓取故障）",
			Cols:  []TableCol{{Name: "domain", Display: "图床"}, {Name: "cnt", Display: "失败量"}},
			Rows:  rows,
		})
	}
	return out, nil
}

// ─── 通用小助手 ──────────────────────────────────────────────────────────────

func first(rows Rows) map[string]string {
	if len(rows) == 0 {
		return map[string]string{}
	}
	return rows[0]
}

func pct(n, d float64) float64 {
	if d == 0 {
		return 0
	}
	return n / d * 100
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func fmtI(v float64) string {
	n := int64(v + 0.5)
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var outParts []string
	for len(s) > 3 {
		outParts = append([]string{s[len(s)-3:]}, outParts...)
		s = s[:len(s)-3]
	}
	outParts = append([]string{s}, outParts...)
	res := outParts[0]
	for _, p := range outParts[1:] {
		res += "," + p
	}
	return res
}

func fmtPct1(v float64) string { return fmt.Sprintf("%.1f%%", v) }

func num2s(s string) string {
	return fmt.Sprintf("%.1f", num(s))
}

// fmtMin 把分钟数展示为人类可读（<90min 用分钟，否则小时）。
func fmtMin(min float64) string {
	if min <= 0 {
		return "—"
	}
	if min < 90 {
		return fmt.Sprintf("%.0f min", min)
	}
	return fmt.Sprintf("%.1f h", min/60)
}

// bucketPercentile 用累计分桶线性插值估算分位数（Mongo <7.0 无 $percentile 的替代）。
func bucketPercentile(n, q float64, bounds, cums []float64) float64 {
	target := n * q
	lo, loCum := 0.0, 0.0
	for i, b := range bounds {
		if cums[i] >= target {
			span := cums[i] - loCum
			if span <= 0 {
				return b
			}
			return lo + (b-lo)*(target-loCum)/span
		}
		lo, loCum = b, cums[i]
	}
	return bounds[len(bounds)-1]
}

// dropOrZero：较基线跌超 30% 或归零 → 🔴（量类指标的标准规则）。
// 基线取昨日与上周同日中较低者：周末量天然低于工作日，只比昨日会出假警。
func dropOrZero(key, msg string) Threshold {
	return Threshold{
		MetricKey: key,
		Eval: func(cur float64, prev *float64) Level {
			if cur == 0 {
				return LevelCrit
			}
			if prev != nil && *prev > 0 && cur < *prev*0.7 {
				return LevelCrit
			}
			return LevelOK
		},
		Msg:               msg,
		BaselineWeeklyMin: true,
	}
}

// warnBelow：低于 warn 给 🟡、低于 crit 给 🔴。
func warnBelow(cur, warn, crit float64) Level {
	if cur < crit {
		return LevelCrit
	}
	if cur < warn {
		return LevelWarn
	}
	return LevelOK
}

// sortRowsByNumDesc 按某数值列降序（渲染辅助）。
func sortRowsByNumDesc(rows []map[string]string, col string) {
	sort.SliceStable(rows, func(i, j int) bool { return num(rows[i][col]) > num(rows[j][col]) })
}
