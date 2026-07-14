package report

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

var testDay = time.Date(2026, 7, 7, 0, 0, 0, 0, time.Local)

// fakeQF 按 Query.Name 返回预置结果；支持按时间窗返回不同数据（拆半自验用）。
func fakeQF(data map[string]Rows, byWindow map[string]map[string]Rows) QueryFunc {
	return func(ctx context.Context, q Query, from, to time.Time) (Rows, error) {
		if byWindow != nil {
			key := fmt.Sprintf("%s~%s", from.Format("15:04"), to.Format("15:04"))
			if m, ok := byWindow[key]; ok {
				if rows, ok := m[q.Name]; ok {
					return rows, nil
				}
			}
		}
		rows, ok := data[q.Name]
		if !ok {
			return nil, errors.New("no such query: " + q.Name)
		}
		return rows, nil
	}
}

func simpleSection(key string, metricVal float64) *Section {
	return &Section{
		Key: key, Title: key,
		Queries: []Query{{Name: "q", Source: SourceSLS, SQL: "x"}},
		Extract: func(day time.Time, results map[string]Rows, prev []Snapshot) (*Output, error) {
			return &Output{Metrics: []Metric{{Key: key + ".v", Display: key, Value: metricVal, Text: fmt.Sprintf("%.0f", metricVal)}}}, nil
		},
	}
}

func TestRun_ThresholdAndSnapshot(t *testing.T) {
	sec := simpleSection("s1", 100)
	sec.Thresholds = []Threshold{{
		MetricKey: "s1.v",
		Eval: func(cur float64, prev *float64) Level {
			if prev != nil && cur < *prev*0.7 {
				return LevelCrit
			}
			if cur < 150 {
				return LevelWarn
			}
			return LevelOK
		},
		Msg: "s1 偏低：%s",
	}}

	qf := fakeQF(map[string]Rows{"q": {{"v": "100"}}}, nil)

	// 无昨日快照：只触发 Warn
	results, snap := Run(context.Background(), testDay, []*Section{sec}, qf, nil)
	if results[0].Level != LevelWarn {
		t.Fatalf("want Warn, got %v", results[0].Level)
	}
	if snap["s1.v"] != 100 {
		t.Fatalf("snapshot missing metric: %v", snap)
	}

	// 有昨日快照且暴跌：升级 Crit
	prev := Snapshot{"s1.v": 200}
	results, _ = Run(context.Background(), testDay, []*Section{sec}, qf, []Snapshot{prev})
	if results[0].Level != LevelCrit {
		t.Fatalf("want Crit with prev snapshot, got %v", results[0].Level)
	}
}

func TestRun_FailureIsolationAndBroken(t *testing.T) {
	bad := &Section{
		Key: "bad", Title: "bad",
		Queries: []Query{{Name: "missing", Source: SourceSLS, SQL: "x"}},
		Extract: func(day time.Time, results map[string]Rows, prev []Snapshot) (*Output, error) {
			return &Output{}, nil
		},
	}
	good := simpleSection("good", 1)

	qf := fakeQF(map[string]Rows{"q": {{"v": "1"}}}, nil)
	results, _ := Run(context.Background(), testDay, []*Section{bad, good}, qf, nil)

	if results[0].Err == nil || results[0].Level != LevelBroken {
		t.Fatalf("bad section should be Broken with err, got %+v", results[0])
	}
	if results[1].Err != nil || len(results[1].Output.Metrics) != 1 {
		t.Fatalf("good section should be isolated from bad one: %+v", results[1])
	}
}

func TestRun_EmptyMetricsMeansBroken(t *testing.T) {
	sec := &Section{
		Key: "s", Title: "s",
		Queries: []Query{{Name: "q", Source: SourceSLS, SQL: "x"}},
		Extract: func(day time.Time, results map[string]Rows, prev []Snapshot) (*Output, error) {
			return &Output{}, nil // 有响应但解不出指标 → 口径失效
		},
	}
	qf := fakeQF(map[string]Rows{"q": {}}, nil)
	results, _ := Run(context.Background(), testDay, []*Section{sec}, qf, nil)
	if results[0].Level != LevelBroken {
		t.Fatalf("empty metrics should mark section Broken, got %v", results[0].Level)
	}
}

func TestRun_CrossSectionCheckAndDerived(t *testing.T) {
	a := simpleSection("a", 100)
	b := simpleSection("b", 90)
	// 派生节引用 a/b 指标并声明勾稽：a.v ≈ b.v（容差 5% → 100 vs 90 失败）
	funnel := &Section{
		Key: "funnel", Title: "funnel",
		DerivedExtract: func(day time.Time, all map[string]Metric, prev []Snapshot) (*Output, error) {
			diff := all["a.v"].Value - all["b.v"].Value
			return &Output{Metrics: []Metric{{Key: "funnel.gap", Display: "缺口", Value: diff, Text: fmt.Sprintf("%.0f", diff)}}}, nil
		},
		Checks: []Check{{LeftKey: "a.v", RightKey: "b.v", TolerancePct: 5, Msg: "a 应≈b"}},
	}
	qf := fakeQF(map[string]Rows{"q": {{"v": "1"}}}, nil)
	results, snap := Run(context.Background(), testDay, []*Section{a, b, funnel}, qf, nil)

	if snap["funnel.gap"] != 10 {
		t.Fatalf("derived metric missing: %v", snap)
	}
	last := results[2]
	if len(last.Hits) != 1 || last.Level != LevelWarn {
		t.Fatalf("check should fail with Warn: %+v", last)
	}
}

func TestExecQuery_SplitVerifyFixesTruncation(t *testing.T) {
	// 全天返回被"截断"的 217905，两个半天相加 221547 → 应采用分片和并出注记
	sec := &Section{
		Key: "s", Title: "s",
		Queries: []Query{{Name: "q", Source: SourceSLS, SQL: "x", VerifyAdditive: []string{"req"}}},
		Extract: func(day time.Time, results map[string]Rows, prev []Snapshot) (*Output, error) {
			v := num(results["q"][0]["req"])
			return &Output{Metrics: []Metric{{Key: "s.req", Display: "req", Value: v, Text: fmt.Sprintf("%.0f", v)}}}, nil
		},
	}
	qf := fakeQF(
		map[string]Rows{"q": {{"req": "217905"}}},
		map[string]map[string]Rows{
			"00:00~12:00": {"q": Rows{{"req": "66295"}}},
			"12:00~00:00": {"q": Rows{{"req": "155252"}}},
		},
	)
	results, snap := Run(context.Background(), testDay, []*Section{sec}, qf, nil)
	if snap["s.req"] != 221547 {
		t.Fatalf("want fixed value 221547, got %v", snap["s.req"])
	}
	foundNote := false
	for _, n := range results[0].Output.Notes {
		if len(n) > 0 {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("truncation note missing: %+v", results[0].Output.Notes)
	}
}

func TestStreakAndDelta(t *testing.T) {
	prevs := []Snapshot{
		{"d.fail": 400}, // 昨日
		{"d.fail": 350},
		{"d.fail": 100}, // 断
		{"d.fail": 500},
	}
	n := Streak(prevs, "d.fail", func(v float64) bool { return v > 300 })
	if n != 2 {
		t.Fatalf("want streak 2, got %d", n)
	}
	if got := DeltaPct(110, prevs, "d.fail"); got != "-72.5%" {
		t.Fatalf("delta got %s", got)
	}
	if got := DeltaPct(110, nil, "d.fail"); got != "—" {
		t.Fatalf("delta without prev got %s", got)
	}
}

func TestWeeklyMinBaseline(t *testing.T) {
	f := func(v float64) Snapshot { return Snapshot{"k": v} }
	cases := []struct {
		name string
		days []Snapshot
		want *float64
	}{
		{"无快照", nil, nil},
		{"仅昨日", []Snapshot{f(100)}, ptr(100.0)},
		{"昨日高上周低取上周", []Snapshot{f(100), nil, nil, nil, nil, nil, f(60)}, ptr(60.0)},
		{"昨日低上周高取昨日", []Snapshot{f(50), nil, nil, nil, nil, nil, f(80)}, ptr(50.0)},
		{"昨日缺退上周", []Snapshot{nil, nil, nil, nil, nil, nil, f(70)}, ptr(70.0)},
		{"指标缺失", []Snapshot{{"other": 1}}, nil},
	}
	for _, c := range cases {
		got := WeeklyMinBaseline(c.days, "k")
		switch {
		case c.want == nil && got != nil:
			t.Errorf("%s: want nil, got %v", c.name, *got)
		case c.want != nil && (got == nil || *got != *c.want):
			t.Errorf("%s: want %v, got %v", c.name, *c.want, got)
		}
	}
}

func ptr(v float64) *float64 { return &v }
