package report

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Run 执行一组 Section：并发查询 → 提取 → 阈值判级 → 勾稽断言 → 汇总快照。
// 单节失败只影响该节（Result.Err），不拖垮整张报表。
// 返回值 snapshot 为当日全部指标，由调用方落盘用于次日环比。
func Run(ctx context.Context, day time.Time, sections []*Section, qf QueryFunc, prevDays []Snapshot) ([]*Result, Snapshot) {
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	to := from.Add(24 * time.Hour)

	results := make([]*Result, len(sections))

	// 第一遍：普通节并发执行
	var wg sync.WaitGroup
	for i, sec := range sections {
		if sec.DerivedExtract != nil {
			continue
		}
		wg.Add(1)
		go func(i int, sec *Section) {
			defer wg.Done()
			results[i] = runSection(ctx, sec, qf, from, to, day, prevDays)
		}(i, sec)
	}
	wg.Wait()

	// 汇总第一遍指标，供派生节与跨节勾稽使用
	all := map[string]Metric{}
	collect := func(r *Result) {
		if r == nil || r.Output == nil {
			return
		}
		for _, m := range r.Output.Metrics {
			all[m.Key] = m
		}
	}
	for _, r := range results {
		collect(r)
	}

	// 第二遍：派生节（漏斗等）
	for i, sec := range sections {
		if sec.DerivedExtract == nil {
			continue
		}
		results[i] = runDerived(sec, day, all, prevDays)
		collect(results[i])
	}

	// 阈值 + 勾稽（统一在全量指标可见后评估）
	for _, r := range results {
		if r == nil || r.Output == nil {
			continue
		}
		evalThresholds(r, all, prevDays)
		evalChecks(r, all)
	}

	snapshot := Snapshot{}
	for k, m := range all {
		snapshot[k] = m.Value
	}
	return results, snapshot
}

func runSection(ctx context.Context, sec *Section, qf QueryFunc, from, to time.Time, day time.Time, prevDays []Snapshot) (res *Result) {
	res = &Result{Section: sec}
	defer func() {
		if p := recover(); p != nil {
			res.Err = fmt.Errorf("section %s panic: %v", sec.Key, p)
			res.Level = LevelBroken
		}
	}()

	rowsByName := map[string]Rows{}
	var notes []string
	for _, q := range sec.Queries {
		rows, note, err := execQuery(ctx, qf, q, from, to)
		if err != nil {
			if q.Optional {
				notes = append(notes, fmt.Sprintf("查询 %s 失败（可选项，已跳过）：%v", q.Name, err))
				continue
			}
			res.Err = fmt.Errorf("query %s: %w", q.Name, err)
			res.Level = LevelBroken
			return res
		}
		if note != "" {
			notes = append(notes, note)
		}
		rowsByName[q.Name] = rows
	}

	out, err := sec.Extract(day, rowsByName, prevDays)
	if err != nil {
		res.Err = fmt.Errorf("extract: %w", err)
		res.Level = LevelBroken
		return res
	}
	out.Notes = append(out.Notes, notes...)
	res.Output = out

	// 口径失效检测：必选查询存在但整节没解出任何指标 → 数字不可信
	if len(out.Metrics) == 0 && len(sec.Queries) > 0 {
		res.Level = LevelBroken
		out.Notes = append(out.Notes, "⚠️ 口径疑似失效：查询有响应但未解出任何指标（埋点变更？）")
	}
	return res
}

func runDerived(sec *Section, day time.Time, all map[string]Metric, prevDays []Snapshot) (res *Result) {
	res = &Result{Section: sec}
	defer func() {
		if p := recover(); p != nil {
			res.Err = fmt.Errorf("derived section %s panic: %v", sec.Key, p)
			res.Level = LevelBroken
		}
	}()
	out, err := sec.DerivedExtract(day, all, prevDays)
	if err != nil {
		res.Err = err
		res.Level = LevelBroken
		return res
	}
	res.Output = out
	return res
}

// execQuery 执行单个查询；VerifyAdditive 非空时做拆半自验（方法论第 3 条：防 SLS 静默截断）。
func execQuery(ctx context.Context, qf QueryFunc, q Query, from, to time.Time) (Rows, string, error) {
	full, err := qf(ctx, q, from, to)
	if err != nil {
		return nil, "", err
	}
	if len(q.VerifyAdditive) == 0 {
		return full, "", nil
	}

	mid := from.Add(to.Sub(from) / 2)
	h1, err1 := qf(ctx, q, from, mid)
	h2, err2 := qf(ctx, q, mid, to)
	if err1 != nil || err2 != nil {
		return full, fmt.Sprintf("查询 %s 拆半自验未完成（%v/%v），采用全天值", q.Name, err1, err2), nil
	}

	// 仅对单行聚合结果做自验（多行分组结果的合并归属 Extract 处理，不在此处做）
	if len(full) != 1 || len(h1) != 1 || len(h2) != 1 {
		return full, "", nil
	}
	fixed := false
	for _, col := range q.VerifyAdditive {
		fv := num(full[0][col])
		sv := num(h1[0][col]) + num(h2[0][col])
		if sv == 0 {
			continue
		}
		if math.Abs(fv-sv)/sv > 0.01 {
			full[0][col] = strconv.FormatFloat(sv, 'f', -1, 64)
			fixed = true
		}
	}
	if fixed {
		return full, fmt.Sprintf("⚠️ 查询 %s 全天扫描疑似被 SLS 截断，已采用拆半之和", q.Name), nil
	}
	return full, "", nil
}

func evalThresholds(r *Result, all map[string]Metric, prevDays []Snapshot) {
	for _, t := range r.Section.Thresholds {
		m, ok := all[t.MetricKey]
		if !ok {
			continue
		}
		var pv *float64
		if t.BaselineWeeklyMin {
			pv = WeeklyMinBaseline(prevDays, t.MetricKey)
		} else if len(prevDays) > 0 && prevDays[0] != nil {
			if v, ok := prevDays[0][t.MetricKey]; ok {
				pv = &v
			}
		}
		lvl := t.Eval(m.Value, pv)
		if lvl > LevelOK {
			r.Hits = append(r.Hits, Hit{Level: lvl, Msg: fmt.Sprintf(t.Msg, m.Text), MetricKey: t.MetricKey})
			if lvl > r.Level && r.Level != LevelBroken {
				r.Level = lvl
			}
		}
	}
}

func evalChecks(r *Result, all map[string]Metric) {
	for _, c := range r.Section.Checks {
		l, lok := all[c.LeftKey]
		rt, rok := all[c.RightKey]
		if !lok || !rok {
			continue
		}
		base := math.Max(math.Abs(rt.Value), 1)
		diff := math.Abs(l.Value - rt.Value)
		if c.OneSided {
			diff = l.Value - rt.Value
		}
		if diff/base > c.TolerancePct/100 {
			r.Hits = append(r.Hits, Hit{
				Level: LevelWarn,
				Msg:   fmt.Sprintf("勾稽失败：%s（%s=%.0f vs %s=%.0f）", c.Msg, l.Display, l.Value, rt.Display, rt.Value),
			})
			if r.Level == LevelOK {
				r.Level = LevelWarn
			}
		}
	}
}

// ─── 快照/环比/通用小工具（sections.go 与 handler 共用） ─────────────────────

// num 宽容地把查询结果字段转成 float64（"null"/"" → 0）。
func num(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// Num 导出版本，供 handler 使用。
func Num(s string) float64 { return num(s) }

// DeltaPct 计算环比文本，如 "+3.2%"；prev 缺失返回 "—"。
// WeeklyMinBaseline 取昨日（prevDays[0]）与上周同日（prevDays[6]）中较低者，
// 作为量类指标的跌幅告警基线：周末量天然低于工作日，只比昨日会出假警。
// 两天都缺快照时返回 nil（调用方应视为无基线、不判跌幅）。
func WeeklyMinBaseline(prevDays []Snapshot, key string) *float64 {
	var base *float64
	for _, idx := range []int{0, 6} {
		if idx >= len(prevDays) || prevDays[idx] == nil {
			continue
		}
		if v, ok := prevDays[idx][key]; ok {
			if base == nil || v < *base {
				base = &v
			}
		}
	}
	return base
}

func DeltaPct(cur float64, prevDays []Snapshot, key string) string {
	if len(prevDays) == 0 || prevDays[0] == nil {
		return "—"
	}
	p, ok := prevDays[0][key]
	if !ok || p == 0 {
		return "—"
	}
	d := (cur - p) / p * 100
	return fmt.Sprintf("%+.1f%%", d)
}

// Streak 计算某指标连续满足条件的天数（从昨日往前数）。
func Streak(prevDays []Snapshot, key string, cond func(v float64) bool) int {
	n := 0
	for _, s := range prevDays {
		if s == nil {
			break
		}
		v, ok := s[key]
		if !ok || !cond(v) {
			break
		}
		n++
	}
	return n
}

// TopHits 从全部节里收集命中的事项，按级别降序排（Top3 需关注用）。
func TopHits(results []*Result, limit int) []Hit {
	var hits []Hit
	for _, r := range results {
		if r == nil {
			continue
		}
		hits = append(hits, r.Hits...)
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Level > hits[j].Level })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// OverallLevel 汇总整卡级别（Broken 视为 Warn 展示，避免一节口径失效把整卡标红）。
func OverallLevel(results []*Result) Level {
	lvl := LevelOK
	for _, r := range results {
		if r == nil {
			continue
		}
		l := r.Level
		if l == LevelBroken {
			l = LevelWarn
		}
		if l > lvl {
			lvl = l
		}
	}
	return lvl
}
