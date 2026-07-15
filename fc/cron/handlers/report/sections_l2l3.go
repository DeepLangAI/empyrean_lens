package report

// 第二层（处理与监控层）与第三层（有效入库层）+ 漏斗派生节。
// 口径依据同 docs/daily-report-口径-第一层数据来源.md；聚类（SimResourceCluster）
// 不进报表（2026-07-08 拍板：聚类是管线内部衍生逻辑，不是数据渠道）。

import (
	"fmt"
	"strings"
	"time"
)

// ─── 2.1 各环节服务健康 ─────────────────────────────────────────────────────

func secPipelineHealth() *Section {
	return &Section{
		Key:   "pipe",
		Layer: LayerL2,
		Title: "2.1 各环节服务健康",
		Queries: []Query{
			{
				Name: "crawl", Source: SourceSLS, Limit: 10,
				SQL: `crawl_normal OR crawl_error | select message, count(*) as cnt from log where extra like '%"domain"%' group by message`,
			},
			{
				Name: "crawl_timing", Source: SourceSLS, Limit: 1, Optional: true,
				SQL: `crawl_timing_normal | select round(approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.50), 1) as p50, round(approx_percentile(cast(regexp_extract(extra, '"crawl_took":([0-9.]+)', 1) as double), 0.99), 1) as p99 from log`,
			},
			{
				// 解析环节：必须用窄域 ResponseRath 关键词——edu_parse 全域 88 万条/天，
				// 宽域全天聚合实测被 SLS 静默截断（骗走 4K）。
				// 耗时双峰是真实分布：21% <500ms（md5 缓存命中）、72% ≥5s（真实全文解析）。
				Name: "parse", Source: SourceSLS, Limit: 1,
				SQL:            `ResponseRath and edu_parse | select count(*) as resp, count_if(message not like '%Respcode:0,%') as fail, round(approx_percentile(cast(regexp_extract(message, 'cost: ([0-9]+) ms', 1) as double), 0.50), 0) as p50_ms, round(approx_percentile(cast(regexp_extract(message, 'cost: ([0-9]+) ms', 1) as double), 0.99), 0) as p99_ms from log where message like '%ResponseRath:/edu_parse,%'`,
				VerifyAdditive: []string{"resp", "fail"},
			},
			{
				// 入库管线。量纲=处理动作（含管线内失败重试，preCheck 复活曾失败资源），
				// ⚠️ cost 语义待确认：管线 P50(14.4s) < 解析 P50(24.7s)，两个 cost 起止点不同，
				// 跨环节耗时对比禁用。
				Name: "proc", Source: SourceSLS, Limit: 5,
				SQL: `ResourceProcessor | select regexp_extract(message, 'status:(\w+)', 1) as status, count(*) as cnt, round(approx_percentile(cast(regexp_extract(message, 'cost:([0-9.]+)', 1) as double), 0.50), 1) as p50_s, round(approx_percentile(cast(regexp_extract(message, 'cost:([0-9.]+)', 1) as double), 0.99), 1) as p99_s from log where message like '%end process resource%' and message not like '%source: SimResourceCluster%' group by status limit 5`,
			},
		},
		Extract: extractPipeline,
		Thresholds: []Threshold{
			{
				MetricKey: "pipe.parse.rate",
				Eval:      func(cur float64, prev *float64) Level { return warnBelow(cur, 99, 97) },
				Msg:       "解析成功率 %s",
			},

		},
	}
}

func extractPipeline(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}
	var crawlOK, crawlFail float64
	for _, row := range r["crawl"] {
		if row["message"] == "crawl_normal" {
			crawlOK = num(row["cnt"])
		} else if row["message"] == "crawl_error" {
			crawlFail = num(row["cnt"])
		}
	}
	pRow := first(r["parse"])
	parseResp, parseFail := num(pRow["resp"]), num(pRow["fail"])
	var procOK, procFail, procP50, procP99 float64
	for _, row := range r["proc"] {
		if row["status"] == "success" {
			procOK = num(row["cnt"])
			procP50, procP99 = num(row["p50_s"]), num(row["p99_s"])
		} else if row["status"] == "failed" {
			procFail = num(row["cnt"])
		}
	}

	out.Metrics = append(out.Metrics,
		Metric{Key: "pipe.crawl.tasks", Display: "内容抓取任务", Value: crawlOK + crawlFail, Text: fmtI(crawlOK + crawlFail), Dimension: DimTask},
		Metric{Key: "pipe.crawl.rate", Display: "抓取成功率", Value: pct(crawlOK, maxf(crawlOK+crawlFail, 1)), Text: fmtPct1(pct(crawlOK, maxf(crawlOK+crawlFail, 1))), Dimension: DimPercent},
		Metric{Key: "pipe.parse.calls", Display: "解析调用", Value: parseResp, Text: fmtI(parseResp), Dimension: DimTask},
		Metric{Key: "pipe.parse.rate", Display: "解析成功率", Value: pct(parseResp-parseFail, maxf(parseResp, 1)), Text: fmtPct1(pct(parseResp-parseFail, maxf(parseResp, 1))), Dimension: DimPercent},
		Metric{Key: "pipe.proc.acts", Display: "入库处理动作数", Value: procOK + procFail, Text: fmtI(procOK + procFail), Dimension: DimTask},
		Metric{Key: "pipe.proc.success", Display: "入库处理成功数", Value: procOK, Text: fmtI(procOK), Dimension: DimTask},
		Metric{Key: "pipe.proc.rate", Display: "入库处理成功率", Value: pct(procOK, maxf(procOK+procFail, 1)), Text: fmtPct1(pct(procOK, maxf(procOK+procFail, 1))), Dimension: DimPercent},
	)

	t := first(r["crawl_timing"])
	rows := []map[string]string{
		{"stage": "内容抓取", "vol": fmtI(crawlOK + crawlFail) + " 任务", "rate": fmtPct1(pct(crawlOK, maxf(crawlOK+crawlFail, 1))), "p50": num2s(t["p50"]) + "s", "p99": num2s(t["p99"]) + "s"},
		{"stage": "解析(edu_parse)", "vol": fmtI(parseResp) + " 次", "rate": fmtPct1(pct(parseResp-parseFail, maxf(parseResp, 1))), "p50": fmt.Sprintf("%.1fs", num(pRow["p50_ms"])/1000), "p99": fmt.Sprintf("%.1fs", num(pRow["p99_ms"])/1000)},
		{"stage": "入库处理", "vol": fmtI(procOK+procFail) + " 动作", "rate": fmtPct1(pct(procOK, maxf(procOK+procFail, 1))), "p50": fmt.Sprintf("%.1fs", procP50), "p99": fmt.Sprintf("%.1fs", procP99)},
	}
	out.Tables = append(out.Tables, Table{
		Title: "各行量纲与覆盖不同，勿纵向加减：抓取按轮询任务计、仅订阅+自采集（人民网/清博是供应商推送，从解析进链路）；解析起全渠道都走；生成按模型调用计（一篇触发多种任务，含存量文章）。逐级损耗看 2.2",
		Cols: []TableCol{
			{Name: "stage", Display: "环节", Width: "22%"}, {Name: "vol", Display: "处理量"},
			{Name: "rate", Display: "成功率"}, {Name: "p50", Display: "P50"}, {Name: "p99", Display: "P99"},
		},
		Rows: rows,
	})
	return out, nil
}

// ─── 2.2 入库失败矩阵（阶段 × 渠道） ────────────────────────────────────────

// 管线阶段真实执行顺序（resource_processor.go Do() 的 handleFunc 链）。
// 死得越靠下成本越高（已消耗抓取/解析/生成算力）。
var stageOrder = []struct {
	stage   string
	display string
}{
	{"UrlChecked", "− URL 校验失败"},
	{"ResourceCrawled", "− 正文抓取失败"},
	{"ResourceParsed", "− 内容解析失败"},
	{"AuthorParsed", "− 作者信息写入异常"},
	{"DuplicateChecked", "− 内容判重淘汰"},
	{"AbstractGenerated", "− 摘要生成失败"},
	{"Validated", "− 字段校验失败"},
}

func secFailureMatrix() *Section {
	return &Section{
		Key:   "m22",
		Layer: LayerL2,
		Title: "2.2 资源处理（渠道 × 处理阶段）",
		Queries: []Query{
			{
				// 失败原因日志：与 end process 同 operation_id 的 error 行，lastest status=死亡阶段
				Name: "matrix", Source: SourceSLS, Limit: 40,
				SQL: `ResourceProcessor and error | select regexp_extract(message, 'source: ([A-Za-z]+)', 1) as source, regexp_extract(message, 'lastest status: ([A-Za-z]+)', 1) as stage, count(*) as cnt from log where message like '%handle resource error%' group by source, stage order by source, cnt desc limit 40`,
			},
			{
				// preCheck 重复更新（success 里拆出）：动作数 − 去重资源数
				Name: "dup", Source: SourceSLS, Limit: 10,
				SQL: `ResourceProcessor | select regexp_extract(message, 'source: ([A-Za-z]+)', 1) as source, count(*) as acts, count(distinct regexp_extract(message, 'uniq_id: ([^;]+)', 1)) as uniqs from log where message like '%end process resource%' and message like '%status:success%' and message not like '%source: SimResourceCluster%' group by source order by acts desc limit 10`,
			},
			{
				// 订阅渠道拆分（自有RSS vs 自有网站）：入库日志 source 只有 Subscription 一个值，
				// 借 root_path + sub 表网站类信源域名集合分类。按量降序，limit 截尾只丢小域名。
				Name: "subsplit_fail", Source: SourceSLS, Limit: 2000,
				SQL: `ResourceProcessor and error and Subscription | select regexp_extract(message, 'lastest status: ([A-Za-z]+)', 1) as stage, regexp_extract(message, 'root_path: ([^;/\]]+)', 1) as rp, count(*) as cnt from log where message like '%handle resource error%' and message like '%source: Subscription%' group by stage, rp order by cnt desc limit 2000`,
			},
			{
				Name: "subsplit_succ", Source: SourceSLS, Limit: 2000,
				SQL: `ResourceProcessor and Subscription | select regexp_extract(message, 'root_path: ([^;/\]]+)', 1) as rp, count(*) as acts, count(distinct regexp_extract(message, 'uniq_id: ([^;]+)', 1)) as uniqs from log where message like '%end process resource%' and message like '%status:success%' and message like '%source: Subscription%' group by rp order by acts desc limit 2000`,
			},
			{
				// 订阅渠道到达量拆分（/resource/add 请求体带 root_path，与失败拆分同一套域名分类）
				Name: "subsplit_arrive", Source: SourceSLS, Limit: 2000,
				SQL: `RequestRout and resource and add | select regexp_extract(message, '"root_path":"([^"/]+)', 1) as rp, count(*) as cnt from log where message like '%RequestRout:/iapi/resource/v1/resource/add,%' and message like '%"source":1,%' group by rp order by cnt desc limit 2000`,
			},
			{
				// 网站类信源清单（source_type≠1 即非 RSS），用于 root_path 分类
				Name: "web_sources", Source: SourceMongo, DB: "subscription", Collection: "sub",
				PipelineJSON: `[{"$match": {"source_type": {"$ne": 1}, "status": 1}}, {"$project": {"url": 1, "_id": 0}}, {"$limit": 800}]`,
				Optional:     true,
			},
		},
		Extract: extractMatrix,
		Thresholds: []Threshold{
			{
				MetricKey: "m22.clean_rate",
				Eval:      func(cur float64, prev *float64) Level { return warnBelow(cur, 88, 80) },
				Msg:       "处理成功率(剔除去重) %s（真实故障水平，与供应商重复行为无关）",
			},
		},
	}
}

func sysFailThreshold(src, name string) Threshold {
	return Threshold{
		MetricKey: "m22.sysrate." + src,
		Eval: func(cur float64, prev *float64) Level {
			if cur > 2 {
				return LevelCrit
			}
			if cur > 0.5 {
				return LevelWarn
			}
			return LevelOK
		},
		Msg: name + "有 %s 篇文章因我方程序报错未能入库",
	}
}

// 矩阵展示列：订阅拆为自有RSS/自有网站两列
var matrixCols = []struct{ key, display string }{
	{"SubRSS", "自有RSS"},
	{"SubWeb", "自有网站"},
	{"Renminwang", "人民网"},
	{"Qingbo", "清博"},
	{"FromMonitoring", "自采集"},
}

func extractMatrix(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}

	// 网站类信源域名集合（缺失时全部归 RSS 并注记）
	webHosts := map[string]bool{}
	for _, row := range r["web_sources"] {
		if h := hostOf(row["url"]); h != "" {
			webHosts[h] = true
		}
	}
	classify := func(rp string) string {
		if webHosts[hostOf(rp)] {
			return "SubWeb"
		}
		return "SubRSS"
	}

	// 失败矩阵 fail[stage][col]
	fail := map[string]map[string]float64{}
	addFail := func(stage, colKey string, v float64) {
		if fail[stage] == nil {
			fail[stage] = map[string]float64{}
		}
		fail[stage][colKey] += v
	}
	chFail := map[string]float64{}    // 原始 source 口径（阈值/勾稽用）
	chFailPre := map[string]float64{} // 供 clean_rate 提前汇总
	for _, row := range r["matrix"] {
		src, stage := row["source"], row["stage"]
		if src == "" || src == "SimResourceCluster" {
			continue
		}
		chFail[src] += num(row["cnt"])
		chFailPre[src] += num(row["cnt"])
		if src != "Subscription" { // 订阅走 subsplit
			addFail(stage, src, num(row["cnt"]))
		}
	}
	for _, row := range r["subsplit_fail"] {
		addFail(row["stage"], classify(row["rp"]), num(row["cnt"]))
	}

	// 成功动作/去重/重复更新
	acts, dup := map[string]float64{}, map[string]float64{}
	var uniqSuccessTotal float64
	for _, row := range r["dup"] {
		src := row["source"]
		uniqSuccessTotal += num(row["uniqs"])
		if src != "Subscription" {
			acts[src] = num(row["acts"])
			dup[src] = num(row["acts"]) - num(row["uniqs"])
		}
	}
	for _, row := range r["subsplit_succ"] {
		colKey := classify(row["rp"])
		acts[colKey] += num(row["acts"])
		dup[colKey] += num(row["acts"]) - num(row["uniqs"])
	}
	out.Metrics = append(out.Metrics,
		Metric{Key: "m22.uniq_success", Display: "去重后成功资源", Value: uniqSuccessTotal, Text: fmtI(uniqSuccessTotal), Dimension: DimURL},
	)
	// 订阅到达量拆分 + 拆分列处理量（供渲染层组装「到达/⓪拦截」两行）
	subArrive := map[string]float64{}
	for _, row := range r["subsplit_arrive"] {
		subArrive[classify(row["rp"])] += num(row["cnt"])
	}
	for _, colKey := range []string{"SubRSS", "SubWeb"} {
		out.Metrics = append(out.Metrics,
			Metric{Key: "m22.arrive." + colKey, Display: colKey + "到达", Value: subArrive[colKey], Text: fmtI(subArrive[colKey]), Dimension: DimDoc},
		)
	}

	// 各列失败合计（处理总量 = 成功动作 + 失败动作）
	colFail := map[string]float64{}
	for _, byCol := range fail {
		for colKey, v := range byCol {
			colFail[colKey] += v
		}
	}
	for _, colKey := range []string{"SubRSS", "SubWeb"} {
		out.Metrics = append(out.Metrics,
			Metric{Key: "m22.total." + colKey, Display: colKey + "处理量", Value: acts[colKey] + colFail[colKey], Text: fmtI(acts[colKey] + colFail[colKey]), Dimension: DimTask},
		)
	}
	for _, c := range matrixCols {
		net := acts[c.key] - dup[c.key]
		out.Metrics = append(out.Metrics,
			Metric{Key: "m22.net." + c.key, Display: c.display + "净新增", Value: net, Text: fmtI(net), Dimension: DimDoc},
			Metric{Key: "m22.dupc." + c.key, Display: c.display + "内容去重", Value: fail["DuplicateChecked"][c.key], Text: fmtI(fail["DuplicateChecked"][c.key]), Dimension: DimTask},
			Metric{Key: "m22.dupu." + c.key, Display: c.display + "存量更新", Value: dup[c.key], Text: fmtI(dup[c.key]), Dimension: DimTask},
		)
	}

	var rows []map[string]string
	addRow := func(name string, get func(colKey string) float64) {
		row := map[string]string{"stage": name}
		for _, c := range matrixCols {
			v := get(c.key)
			if v == 0 {
				row[c.key] = "—"
			} else {
				row[c.key] = fmtI(v)
			}
		}
		rows = append(rows, row)
	}
	// 账本结构：开工 − 各类失败 = 加工成功 − 更新旧文 = 净新增（每列竖着可加减）
	addRow("＝ 进入处理", func(c string) float64 { return acts[c] + colFail[c] })
	for _, s := range stageOrder {
		st := s.stage
		if fail[st] == nil {
			continue
		}
		addRow(s.display, func(c string) float64 { return fail[st][c] })
	}
	addRow("＝ 处理成功", func(c string) float64 { return acts[c] })
	addRow("− 重推旧文(仅更新数据，不新增)", func(c string) float64 { return dup[c] })
	addRow("＝ 净新增入库", func(c string) float64 { return acts[c] - dup[c] })
	// 整体成功率：处理成功 / (进入处理 − 内容判重淘汰)。判重淘汰是正确行为不算失败，
	// 重推旧文入库成功只是不新增，也不算失败——与全局 m22.clean_rate 同口径。
	rateRow := map[string]string{"stage": "整体成功率(判重淘汰不算失败)"}
	for _, c := range matrixCols {
		denom := acts[c.key] + colFail[c.key] - fail["DuplicateChecked"][c.key]
		if denom <= 0 {
			rateRow[c.key] = "—"
			continue
		}
		rate := pct(acts[c.key], denom)
		rateRow[c.key] = fmtPct1(rate)
		out.Metrics = append(out.Metrics,
			Metric{Key: "m22.rate." + c.key, Display: c.display + "整体成功率(剔除判重)", Value: rate, Text: fmtPct1(rate), Dimension: DimPercent},
		)
	}
	rows = append(rows, rateRow)

	cols := []TableCol{{Name: "stage", Display: "阶段(执行顺序)", Width: "25%"}}
	for _, c := range matrixCols {
		cols = append(cols, TableCol{Name: c.key, Display: c.display, Width: "15%"})
	}
	out.Tables = append(out.Tables, Table{Title: "自上而下逐行可加减：接收 − 去重拦截 = 进入处理 − 各阶段失败 = 处理成功 − 存量更新 = 净新增。失败阶段越靠后，已消耗的处理成本越高", Cols: cols, Rows: rows})

	// 系统异常率（阈值口径按原始 source，订阅不拆）
	chDisplay := map[string]string{"Subscription": "订阅", "Renminwang": "人民网", "Qingbo": "清博", "FromMonitoring": "自采集"}
	chActsOrig := map[string]float64{}
	for _, row := range r["dup"] {
		chActsOrig[row["source"]] = num(row["acts"])
	}
	sysStage := map[string]map[string]float64{}
	for _, row := range r["matrix"] {
		if sysStage[row["stage"]] == nil {
			sysStage[row["stage"]] = map[string]float64{}
		}
		sysStage[row["stage"]][row["source"]] += num(row["cnt"])
	}
	var dupContentTotal float64
	for _, v := range fail["DuplicateChecked"] {
		dupContentTotal += v
	}
	out.Metrics = append(out.Metrics, Metric{Key: "m22.dup_content", Display: "内容去重合计", Value: dupContentTotal, Text: fmtI(dupContentTotal), Dimension: DimTask})
	// 处理成功率（剔除内容去重）：去重是正确行为不算失败，此口径只反映真实故障
	{
		var actsTotal, succTotal float64
		for _, row := range r["dup"] {
			succTotal += num(row["acts"])
		}
		actsTotal = succTotal
		for _, v := range chFailPre {
			actsTotal += v
		}
		cleanRate := pct(succTotal, maxf(actsTotal-dupContentTotal, 1))
		out.Metrics = append(out.Metrics, Metric{Key: "m22.clean_rate", Display: "处理成功率(剔除去重)", Value: cleanRate, Text: fmtPct1(cleanRate), Dimension: DimPercent})
	}
	var failedTotal float64
	for src, v := range chFail {
		failedTotal += v
		total := chActsOrig[src] + v
		sys := sysStage["AuthorParsed"][src] + sysStage["UrlChecked"][src]
		rate := pct(sys, maxf(total, 1))
		if chDisplay[src] == "" {
			continue
		}
		out.Metrics = append(out.Metrics,
			Metric{Key: "m22.sysrate." + src, Display: chDisplay[src] + "系统错误失败率", Value: rate, Text: fmtI(sys), Dimension: DimPercent},
			Metric{Key: "m22.total." + src, Display: chDisplay[src] + "处理动作", Value: total, Text: fmtI(total), Dimension: DimTask},
		)
	}
	out.Metrics = append(out.Metrics, Metric{Key: "m22.failed", Display: "入库处理失败数", Value: failedTotal, Text: fmtI(failedTotal), Dimension: DimTask})

	if len(webHosts) == 0 {
		out.Notes = append(out.Notes, "⚠️ 网站类信源清单查询失败，本日「自有网站」并入「自有RSS」列")
	}
	if len(r["subsplit_fail"]) >= 2000 || len(r["subsplit_succ"]) >= 2000 {
		out.Notes = append(out.Notes, "⚠️ 订阅拆分明细触达行数上限，尾部小域名可能少计")
	}
	return out, nil
}

// hostOf 从 url/root_path 提取主机名（去 scheme、去 www.、取到第一个 /）。
func hostOf(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	for _, p := range []string{"https://", "http://"} {
		s = strings.TrimPrefix(s, p)
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimPrefix(s, "www.")
}

// ─── 2.3 数据生成服务 ───────────────────────────────────────────────────────

var genServiceDisplay = map[string]string{
	"single_abstract":      "概述",
	"single_outline":       "全文拆解",
	"hs_tts":               "语音合成",
	"multi_abstract":       "多文档概述",
	"multi_theme":          "多文档主题",
	"smart_outline":        "要点透视",
	"model_daily":          "日报",
	"multi_summary":        "多文档全文拆解",
	"single_enlightenment": "专属灵感",
}

func secGenService() *Section {
	return &Section{
		Key:   "gen",
		Layer: LayerL2,
		Title: "2.3 数据生成服务",
		Queries: []Query{
			{
				// 总量/成功率：OutRequest 口径（模型调用维度，失败按 operation_id/entryId 去重）。
				// 与 P99 的 API 请求维度不同，两表并列各标量纲，勿混读。
				Name: "gen", Source: SourceSLS, Limit: 50,
				SQL: genServiceSQL,
			},
			{
				Name: "lat", Source: SourceSLS, Limit: 20, Optional: true,
				SQL: `__tag__:_container_name_: lingowhale-repeater-go-prod and Response and rout | select regexp_extract(message, 'rout:([^,]+)', 1) as rout, count(*) as cnt, round(approx_percentile(cast(regexp_extract(message, 'cost:([0-9.]+)', 1) as double), 0.99), 1) as p99 from log group by rout order by cnt desc limit 20`,
			},
		},
		Extract: extractGen,
		Thresholds: []Threshold{
			{
				MetricKey: "gen.worst_rate",
				Eval:      func(cur float64, prev *float64) Level { return warnBelow(cur, 98, 95) },
				Msg:       "生成服务最差成功率 %s（详见生成服务表）",
			},
		},
	}
}

// 生成类 API 路由 → service_type 的 P99 归属（查询类毫秒级路由排除）。
var genLatRoutes = map[string]string{
	"/api/repeater/abstract":      "single_abstract",
	"/doc/single/analyze":         "single_outline",
	"/doc/multi/abstract":         "multi_abstract",
	"/api/repeater/smart_outline": "smart_outline",
	"/api/repeater/daily_summary": "model_daily",
	"/api/repeater/enlightenment": "single_enlightenment",
}

func extractGen(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}
	latByType := map[string]float64{}
	for _, row := range r["lat"] {
		if t, ok := genLatRoutes[row["rout"]]; ok {
			latByType[t] = num(row["p99"])
		}
	}
	worst := 100.0
	var sumTotal, sumErrs float64
	var textTotal, textErrs float64 // 内容生成（概述/大纲类）；语音合成、日报是衍生产品，不算 2.1 的生成环节
	var rows []map[string]string
	for _, g := range r["gen"] {
		st := g["service_type"]
		total, errs, rate := num(g["total_calls"]), num(g["error_ops"]), num(g["success_pct"])
		sumTotal += total
		sumErrs += errs
		if st != "hs_tts" && st != "model_daily" {
			textTotal += total
			textErrs += errs
		}
		name := genServiceDisplay[st]
		if name == "" {
			name = st
		}
		if total >= 1000 && rate < worst {
			worst = rate
		}
		p99 := "—"
		if v, ok := latByType[st]; ok {
			p99 = fmt.Sprintf("%.0fs", v)
		}
		rows = append(rows, map[string]string{
			"task": name, "total": fmtI(total), "rate": fmtPct1(rate), "errs": fmtI(errs), "p99": p99,
		})
		out.Metrics = append(out.Metrics,
			Metric{Key: "gen." + st + ".rate", Display: name + "成功率", Value: rate, Text: fmtPct1(rate), Dimension: DimPercent},
			Metric{Key: "gen." + st + ".total", Display: name + "调用量", Value: total, Text: fmtI(total), Dimension: DimTask},
		)
	}
	out.Metrics = append(out.Metrics, Metric{Key: "gen.worst_rate", Display: "生成最差成功率", Value: worst, Text: fmtPct1(worst), Dimension: DimPercent})
	sortRowsByNumDesc(rows, "total")
	// 整体成功率＝各类型调用量加权（1 − 失败合计/调用合计），排序后追加保持在表尾
	if textTotal > 0 {
		textRate := 100 * (1 - textErrs/textTotal)
		out.Metrics = append(out.Metrics,
			Metric{Key: "gen.text_rate", Display: "内容生成成功率(不含语音/日报)", Value: textRate, Text: fmtPct1(textRate), Dimension: DimPercent},
			Metric{Key: "gen.text_calls", Display: "内容生成调用量(不含语音/日报)", Value: textTotal, Text: fmtI(textTotal), Dimension: DimTask},
		)
	}
	if sumTotal > 0 {
		overall := 100 * (1 - sumErrs/sumTotal)
		out.Metrics = append(out.Metrics,
			Metric{Key: "gen.overall_rate", Display: "生成服务整体成功率", Value: overall, Text: fmtPct1(overall), Dimension: DimPercent},
			Metric{Key: "gen.total_calls", Display: "生成服务调用合计", Value: sumTotal, Text: fmtI(sumTotal), Dimension: DimTask},
		)
		rows = append(rows, map[string]string{"task": "整体(调用量加权)", "total": fmtI(sumTotal), "rate": fmtPct1(overall), "errs": fmtI(sumErrs), "p99": "—"})
	}
	out.Tables = append(out.Tables, Table{
		Title: "量/成功率=模型调用维度；P99=API 请求维度（两个量纲）",
		Cols: []TableCol{
			{Name: "task", Display: "任务类型"}, {Name: "total", Display: "调用量"},
			{Name: "rate", Display: "成功率"}, {Name: "errs", Display: "失败"}, {Name: "p99", Display: "P99"},
		},
		Rows: rows,
	})
	return out, nil
}

// ─── 3.1 有效入库 + 3.2 漏斗 ────────────────────────────────────────────────

func secEffective() *Section {
	return &Section{
		Key:   "eff",
		Layer: LayerL3,
		Title: "3.1 有效入库总览",
		Queries: []Query{
			{
				// content_info 是 notify 之后由下游写入的"产品可见"库。剔除聚类(12)。
				Name: "types", Source: SourceMongo, DB: "lingowhale", Collection: "content_info",
				PipelineJSON: `[{"$match": {"create_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}}, {"$group": {"_id": "$entry_type", "count": {"$sum": 1}}}, {"$sort": {"count": -1}}, {"$limit": 20}]`,
			},
			{
				Name: "weixin_split", Source: SourceMongo, DB: "lingowhale", Collection: "content_info",
				PipelineJSON: `[{"$match": {"create_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}, "entry_type": 7}}, {"$group": {"_id": {"$cond": [{"$regexMatch": {"input": {"$ifNull": ["$orig_url", ""]}, "regex": "mp\\.weixin\\.qq\\.com"}}, "weixin", "web"]}, "count": {"$sum": 1}}}]`,
			},
			{
				// 端到端时延（pub_time→create_time）分桶。P99 因 pub_time 日期精度+回补长尾失真，
				// 报表用占比三档（≤30min/≤4h/≤24h），不报 P99（决策 G）。
				Name: "e2e", Source: SourceMongo, DB: "lingowhale", Collection: "content_info",
				PipelineJSON: `[{"$match": {"create_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}, "entry_type": 7, "pub_time": {"$ne": null}}}, {"$project": {"_id": 0, "kind": {"$cond": [{"$regexMatch": {"input": {"$ifNull": ["$orig_url", ""]}, "regex": "mp\\.weixin\\.qq\\.com"}}, "weixin", "web"]}, "lag_min": {"$divide": [{"$subtract": ["$create_time", "$pub_time"]}, 60000]}}}, {"$match": {"lag_min": {"$gte": 0}}}, {"$group": {"_id": "$kind", "n": {"$sum": 1}, "le30m": {"$sum": {"$cond": [{"$lte": ["$lag_min", 30]}, 1, 0]}}, "le4h": {"$sum": {"$cond": [{"$lte": ["$lag_min", 240]}, 1, 0]}}, "le24h": {"$sum": {"$cond": [{"$lte": ["$lag_min", 1440]}, 1, 0]}}}}]`,
				Optional:     true,
			},
		},
		Extract: extractEffective,
		Thresholds: []Threshold{
			dropOrZero("eff.total", "有效入库量异常：%s（较昨日与上周同日均跌超 30% 或归零）"),
		},
	}
}

func extractEffective(day time.Time, r map[string]Rows, prev []Snapshot) (*Output, error) {
	out := &Output{}
	var article, pdf float64
	for _, row := range r["types"] {
		switch row["_id"] {
		case "7":
			article = num(row["count"])
		case "10":
			pdf = num(row["count"])
		}
	}
	var weixin, web float64
	for _, row := range r["weixin_split"] {
		if row["_id"] == "weixin" {
			weixin = num(row["count"])
		} else {
			web = num(row["count"])
		}
	}
	total := article + pdf
	out.Metrics = append(out.Metrics,
		Metric{Key: "eff.total", Display: "有效入库", Value: total, Text: fmtI(total), Dimension: DimDoc},
		Metric{Key: "eff.weixin", Display: "有效入库公众号文章", Value: weixin, Text: fmtI(weixin), Dimension: DimDoc},
		Metric{Key: "eff.web", Display: "有效入库网站文章", Value: web, Text: fmtI(web), Dimension: DimDoc},
		Metric{Key: "eff.pdf", Display: "有效入库PDF文档", Value: pdf, Text: fmtI(pdf), Dimension: DimDoc},
	)

	rows := []map[string]string{
		{"kind": "公众号文章", "cnt": fmtI(weixin), "delta": DeltaPct(weixin, prev, "eff.weixin"), "e2e": "—"},
		{"kind": "网站文章", "cnt": fmtI(web), "delta": DeltaPct(web, prev, "eff.web"), "e2e": "—"},
		{"kind": "PDF/文档", "cnt": fmtI(pdf), "delta": DeltaPct(pdf, prev, "eff.pdf"), "e2e": "—"},
	}
	for _, row := range r["e2e"] {
		n := num(row["n"])
		if n == 0 {
			continue
		}
		text := fmt.Sprintf("≤30min %s · ≤4h %s · ≤24h %s",
			fmtPct1(pct(num(row["le30m"]), n)), fmtPct1(pct(num(row["le4h"]), n)), fmtPct1(pct(num(row["le24h"]), n)))
		if row["_id"] == "weixin" {
			rows[0]["e2e"] = text
			out.Metrics = append(out.Metrics, Metric{Key: "eff.e2e.weixin.le30m", Display: "公众号端到端≤30min占比", Value: pct(num(row["le30m"]), n), Text: fmtPct1(pct(num(row["le30m"]), n)), Dimension: DimPercent})
		} else {
			rows[1]["e2e"] = text
		}
	}
	out.Tables = append(out.Tables, Table{
		Cols: []TableCol{
			{Name: "kind", Display: "内容类型"}, {Name: "cnt", Display: "有效入库"},
			{Name: "delta", Display: "环比"}, {Name: "e2e", Display: "端到端时延分布"},
		},
		Rows: rows,
	})
	return out, nil
}

func secFunnel() *Section {
	return &Section{
		Key:   "funnel",
		Layer: LayerL3,
		Title: "3.2 当日数据漏斗（内容视角，单位：篇）",
		DerivedExtract: func(day time.Time, all map[string]Metric, prev []Snapshot) (*Output, error) {
			out := &Output{}
			g := func(key string) float64 { return all[key].Value }

			// 上游失效防御：漏斗的每个输入节都可能因查询失败被隔离，缺哪个都不能
			// 拿零值硬算——那会产出"净新增 0、15 万篇未上架"级别的鬼话（2026-07-14 实测）。
			for _, dep := range []string{"supplier.recv.total", "resource_add.total", "m22.failed", "m22.uniq_success", "eff.total"} {
				if _, ok := all[dep]; !ok {
					return nil, fmt.Errorf("上游指标 %s 缺失（对应节生成失败），漏斗不可算", dep)
				}
			}

			top := g("supplier.recv.total") + g("self.push") + g("supplier.push.1") + g("supplier.push.15")
			converge := g("resource_add.total")
			acts := g("m22.failed") + sumSuccActs(all)
			uniq := g("m22.uniq_success")
			eff := g("eff.total")

			// 三类重复（扔掉是功劳，不是损失）
			dedupEntry := converge - acts // 去重拦截（幂等）
			if dedupEntry < 0 {
				dedupEntry = 0
			}
			dupContent := g("m22.dup_content")       // 内容去重（跨源重复）
			dupUpdate := sumSuccActs(all) - uniq     // 存量更新（不新增）
			if dupUpdate < 0 {
				dupUpdate = 0
			}
			dupTotal := dedupEntry + dupContent + dupUpdate

			inflight := top - converge // 自采正文抓取在途/损耗（跨天回来，非丢失）
			if inflight < 0 {
				inflight = 0
			}
			unique := top - dupTotal - inflight            // 独有新内容
			failNonDup := g("m22.failed") - dupContent     // 真实处理失败（剔除重复）
			residual := unique - failNonDup - eff          // 残差（安全拦截/跨日/勾稽零头）

			digestRate := pct(eff, maxf(unique, 1))
			dupRate := pct(dupTotal, maxf(top, 1))
			lost := uniq - eff
			if lost < 0 {
				lost = 0 // 可见库多出=昨日积压到账，非丢失
			}
			out.Metrics = append(out.Metrics,
				Metric{Key: "funnel.lost_pct", Display: "成功未上架", Value: pct(lost, maxf(uniq, 1)), Text: fmtI(lost) + " 篇", Dimension: DimPercent},
				Metric{Key: "funnel.top", Display: "一层接收合计", Value: top, Text: fmtI(top), Dimension: DimDoc},
				Metric{Key: "funnel.unique", Display: "独有新内容", Value: unique, Text: fmtI(unique), Dimension: DimDoc},
				Metric{Key: "funnel.digest_rate", Display: "消化率", Value: digestRate, Text: fmtPct1(digestRate), Dimension: DimPercent},
				Metric{Key: "funnel.dup_rate", Display: "重复率", Value: dupRate, Text: fmtPct1(dupRate), Dimension: DimPercent},
				Metric{Key: "funnel.eff_rate_top", Display: "有效率(vs接收)", Value: pct(eff, maxf(top, 1)), Text: fmtPct1(pct(eff, maxf(top, 1))), Dimension: DimPercent},
			)

			rows := []map[string]string{
				{"layer": "全部接收", "cnt": fmtI(top), "loss": "供应商 + 自采 + 订阅 + 播客"},
				{"layer": "− 重复", "cnt": fmtI(dupTotal), "loss": fmt.Sprintf("重复推送 %s · 内容判重淘汰 %s · 重推旧文 %s", fmtI(dedupEntry), fmtI(dupContent), fmtI(dupUpdate))},
				{"layer": "− 自采在途", "cnt": fmtI(inflight), "loss": "链接已收、正文未抓完，跨天回来"},
				{"layer": "＝ 独有新内容", "cnt": fmtI(unique), "loss": "当日真正的新内容"},
				{"layer": "− 处理失败", "cnt": fmtI(failNonDup), "loss": "明细见 2.2"},
			}
			if residual >= 0 {
				rows = append(rows, map[string]string{"layer": "− 其他损耗", "cnt": fmtI(residual), "loss": "安全拦截、跨日零头"})
			} else {
				rows = append(rows, map[string]string{"layer": "＋ 昨日积压今日到账", "cnt": fmtI(-residual), "loss": "昨天收的文章今天入库，补记"})
			}
			rows = append(rows, map[string]string{"layer": "＝ 净新增可见", "cnt": fmtI(eff), "loss": "语鲸用户当日可见的新增"})
			out.Tables = append(out.Tables, Table{
				Cols: []TableCol{
					{Name: "layer", Display: "层", Width: "32%"},
					{Name: "cnt", Display: "数量", Width: "18%"},
					{Name: "loss", Display: "说明", Width: "50%"},
				},
				Rows: rows,
			})
			return out, nil
		},
		Thresholds: []Threshold{
			{
				// 方向感知：处理成功但未出现在可见库 = 真问题；可见库多出（昨日积压到账）不报
				MetricKey: "funnel.lost_pct",
				Eval: func(cur float64, prev *float64) Level {
					if cur > 5 {
						return LevelCrit
					}
					if cur > 2 {
						return LevelWarn
					}
					return LevelOK
				},
				Msg: "有 %s 处理成功的文章未出现在产品可见库，需排查下游写入",
			},
		},
		Checks: []Check{
			{LeftKey: "funnel.top", RightKey: "resource_add.total", TolerancePct: 8, Msg: "一层接收与处理入口脱钩"},
		},
	}
}

func sumSuccActs(all map[string]Metric) float64 {
	var v float64
	for _, src := range []string{"Subscription", "Renminwang", "Qingbo", "FromMonitoring"} {
		v += all["m22.total."+src].Value
	}
	// m22.total.* 已含失败动作，去掉一次失败量避免双计
	return v - all["m22.failed"].Value
}

// genServiceSQL 是老 stability 验证过的复杂口径：分母=OutRequest 模型调用；
// 分子=真实失败（非 5001 按 operation_id 去重，5001 广播按 entryId 去重）。
const genServiceSQL = `__tag__:_container_name_: lingowhale-repeater-go-prod and __tag__:_namespace_: repeater and (OutRequest or level: error) | select service_type, sum(total) as total_calls, sum(errors) as error_ops, round(100.0 * (1.0 - cast(sum(errors) as double) / nullif(cast(sum(total) as double), 0)), 2) as success_pct from ( select regexp_extract(message, 'OutRequest (\S+)', 1) as service_type, 1 as total, 0 as errors from log where message like 'OutRequest %' and regexp_extract(message, 'OutRequest (\S+)', 1) not in ('upload_oss', 'add_voice', 'model_daily_voice') union all select service_type, 0 as total, 1 as errors from ( select operation_id as dedup_key, case when message like 'StreamErrResp%' and message not like '%code:21001%' and message not like '%code:5001%' then 'single_abstract' when message like '解析模型返回数据失败%' or message like '调用下游服务失败%' then regexp_extract(message, 'model:(\w+)', 1) when message like 'StreamErr %' then regexp_extract(message, 'StreamErr (\w+)', 1) when message like '多文档大纲模型生成失败%' then 'multi_summary' when message like 'EditAudio error%' or message like 'HSText2Voice%' then 'hs_tts' end as service_type from log where level = 'error' and message not like '%origContent is empty%' and message not like '%内容过少，不支持生成%' and message not like '%AsyncExec error success%' and message not like 'FindOneByParseIDAndDataType%' and message not like 'SingAnalyze Error%' and message not like 'ErrorResponse%' and message not like 'server panic%' and not (message like 'StreamErrResp%' and message like '%code:5001%') group by dedup_key, service_type having service_type is not null union all select regexp_extract(message, 'entryId:(\S+) ', 1) as dedup_key, 'single_abstract' as service_type from log where level = 'error' and message like 'StreamErrResp%' and message like '%code:5001%' group by dedup_key, service_type having dedup_key is not null and dedup_key != '' ) ) t group by service_type having service_type is not null and service_type != '' order by total_calls desc`

// residualNote 残差说明：为负说明当日入库含昨日队列积压（跨日泄洪），非异常。
func residualNote(v float64) string {
	if v < 0 {
		return "为负=昨日队列积压今日入库（跨日泄洪）"
	}
	return "安全拦截/跨日边界"
}

