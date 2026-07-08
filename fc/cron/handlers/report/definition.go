// Package report 是每日巡检报表的声明式框架。
//
// 设计原则（源自 docs/daily-report-口径-第一层数据来源.md 的方法论第 0 节）：
//   - 一个 Section = 一份定义 {数据源查询, 提取规则, 阈值, 勾稽断言}，
//     业务口径（SQL/字段/阈值）全部集中在 sections.go，改埋点只动定义不动引擎。
//   - 本包不 import 任何业务/基建包（纯 stdlib），数据源能力通过 QueryFunc 依赖注入，
//     可独立编译、独立单测。
//   - 指标必须声明量纲（任务/URL/篇/处理动作），防止条数当篇数的老错误。
//   - 查询结果 0 行/全 null 时不显示 0，显示"口径疑似失效"。
package report

import (
	"context"
	"time"
)

// ─── 数据源 ──────────────────────────────────────────────────────────────────

// SourceKind 声明查询走哪个数据源，由 handler 注入对应实现。
type SourceKind string

const (
	SourceSLS       SourceKind = "sls"        // business-pod 日志库
	SourceSLSNginx  SourceKind = "sls-nginx"  // nginx-ingress 日志库
	SourceMongo     SourceKind = "mongo"      // lingowhale 等库（走 mcp-db 代理）
	SourceSpiderDB  SourceKind = "spider-db"  // wechat-spider 库（直连）
)

// Query 是一次数据源查询的声明。时间窗口由引擎按报表日注入，SQL/pipeline 里不写死日期。
type Query struct {
	Name   string     // 结果在 Extract 中按此名取用
	Source SourceKind

	// SLS 专用。SQL 为完整查询语句（关键词粗筛 + | select 精确判定）。
	SQL   string
	Limit int // 返回行数上限，0 则默认 20

	// Mongo 专用。PipelineJSON 为聚合管道的 JSON 数组文本；
	// 其中的时间占位符 {{DAY_START}} / {{DAY_END}} 由引擎替换为报表日
	// 的 UTC 边界（RFC3339），如 2026-07-06T16:00:00Z。
	DB           string
	Collection   string
	PipelineJSON string

	// VerifyAdditive 非空时，引擎会把该查询按上下半天各跑一次，将列出的
	// 数值列求和与全天结果比对；偏差 >1% 时采用分片之和并给 Section 挂
	// "SLS 扫描疑似截断" 注记。只能用于纯加法列（count 类），分位数列不适用。
	VerifyAdditive []string

	// Optional 为 true 时查询失败只降级为注记，不把整节标为口径失效。
	Optional bool
}

// Rows 是查询结果的统一形态：每行一个 string→string 映射（SLS 原生如此，Mongo 由注入层拍平）。
type Rows []map[string]string

// QueryFunc 由 handler 注入。from/to 为报表日的本地时间边界 [00:00, 24:00)。
type QueryFunc func(ctx context.Context, q Query, from, to time.Time) (Rows, error)

// ─── 指标与结果 ──────────────────────────────────────────────────────────────

// Dimension 是指标量纲。报表上必须展示，防止"条数当篇数"。
type Dimension string

const (
	DimTask    Dimension = "任务"   // 一次动作一条（轮询/请求/处理动作）
	DimURL     Dimension = "URL"    // 按 URL 去重后的篇数
	DimDoc     Dimension = "篇"     // 文档/文章数
	DimAccount Dimension = "账号"
	DimPercent Dimension = "%"
	DimSeconds Dimension = "秒"
	DimMinutes Dimension = "分钟"
	DimNone    Dimension = ""
)

// Metric 是一个巡检指标。Key 全局唯一（用于快照/环比/勾稽引用），格式 "节.名"，
// 如 "supplier.push.renminwang"。Value 用于阈值/环比/勾稽，Text 用于展示（可含单位换算）。
type Metric struct {
	Key       string
	Display   string
	Value     float64
	Text      string
	Dimension Dimension
}

// Table 是报表里的一张表格。列名必须是 ASCII 标识符（飞书卡片约束），Display 才是中文。
// Compact 为 true 时渲染层降级为 markdown 文本行——飞书卡片有 table 组件数量上限，
// 低信息密度的表让位给矩阵/漏斗等核心大表。
type Table struct {
	Title   string
	Cols    []TableCol
	Rows    []map[string]string
	Compact bool
}

type TableCol struct {
	Name    string
	Display string
	Width   string // 可选："auto"（默认）、"120px"（80~600）、"25%"
}

// Level 是节/整卡的健康级别。
type Level int

const (
	LevelOK Level = iota
	LevelWarn
	LevelCrit
	// LevelBroken 表示口径疑似失效（查询 0 行/全 null/出错），数字不可信。
	LevelBroken
)

func (l Level) Icon() string {
	switch l {
	case LevelOK:
		return "🟢"
	case LevelWarn:
		return "🟡"
	case LevelCrit:
		return "🔴"
	default:
		return "⚠️"
	}
}

// Chart 是卡片里的一张图（飞书卡片 chart 组件，SpecJSON 为 VChart 规格 JSON）。
type Chart struct {
	SpecJSON string
}

// Output 是一节的产出。
type Output struct {
	Metrics []Metric
	Tables  []Table
	Charts  []Chart
	Notes   []string // 口径注记/异常说明，渲染为引用行
}

// 报表三层结构（渲染层按此分组，与模板结构一致）。
const (
	LayerL1 = "第一层 · 数据来源层（接受了哪些数据）"
	LayerL2 = "第二层 · 处理与监控层（中间发生了什么）"
	LayerL3 = "第三层 · 有效入库层（用户视角剩下多少）"
)

// ─── 阈值与勾稽 ──────────────────────────────────────────────────────────────

// Threshold 对某个指标做健康判级。cur 为当日值；prev 为昨日快照值（无快照时为 nil）。
// 返回该规则命中的级别（未命中返回 LevelOK）。Msg 在命中时作为"需关注"事项输出。
type Threshold struct {
	MetricKey string
	Eval      func(cur float64, prev *float64) Level
	Msg       string // 支持 %v 占位当日值
}

// Check 是勾稽断言：|left − right| / max(right,1) ≤ TolerancePct。
// Left/Right 是指标 Key，可跨节引用（漏斗就是一组跨节 Check）。
// 断言失败不是报表错误，而是报表内容——渲染为"⚠️ 勾稽失败"事项。
type Check struct {
	LeftKey      string
	RightKey     string
	TolerancePct float64
	Msg          string
}

// ─── Section ────────────────────────────────────────────────────────────────

// Snapshot 是某一天全部指标的落盘形态（handler 存 Redis，key=daily-report:{date}）。
type Snapshot map[string]float64

// Section 是一节报表的完整定义。
type Section struct {
	Key   string
	Title string
	Layer string // LayerL1/L2/L3，渲染层按此分组

	Queries []Query

	// Extract 把查询结果解析为指标/表格。results 按 Query.Name 取；
	// prevDays[0] 为昨日快照、[1] 为前日……（可能为空切片或含 nil）。
	// Derived 节（无 Queries）的 Extract 收到的 results 为空，改用 all 参数。
	Extract func(day time.Time, results map[string]Rows, prevDays []Snapshot) (*Output, error)

	// DerivedExtract 非空时该节为派生节（如漏斗）：引擎最后运行，
	// all 为本次运行已产出的全部指标。与 Extract 二选一。
	DerivedExtract func(day time.Time, all map[string]Metric, prevDays []Snapshot) (*Output, error)

	Thresholds []Threshold
	Checks     []Check
}

// Result 是引擎对一节的运行结果。
type Result struct {
	Section *Section
	Output  *Output
	Level   Level
	// Hits 是命中的阈值/勾稽消息（含级别），用于"今日需关注"汇总。
	Hits []Hit
	Err  error // 节级失败（查询全挂/Extract panic），失败隔离不拖垮整报表
}

type Hit struct {
	Level Level
	Msg   string
}
