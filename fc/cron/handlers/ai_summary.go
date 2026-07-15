package handlers

// AI 日报解读：把近 7 天核心指标序列 + 当日告警喂给内部 new-api 网关
// （Anthropic 兼容 Messages 接口，与 resource 服务同一套），产出 3-4 句人话解读，
// 放在群摘要卡顶部。旁路：网关不可用/超时/未配置时返回空串，卡片自动省略该段。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"empyrean_lens/conf"
	"empyrean_lens/fc/cron/handlers/report"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	consts "github.com/cloudwego/hertz/pkg/protocol/consts"
)

// 喂给模型的核心指标序列（挑趋势最有信息量的，别把 140 个全塞进去）
var aiContextKeys = []struct{ key, name string }{
	{"eff.total", "有效入库总量(篇)"},
	{"funnel.top", "一层接收合计(篇)"},
	{"self.missing.rate", "自采缺失率(%)"},
	{"self.regular", "自采按发布日采集量(篇)"},
	{"m22.clean_rate", "处理成功率剔除去重(%)"},
	{"gen.text_rate", "内容生成成功率(%)"},
	{"sub.task_rate", "自有RSS/网站抓取成功率(%)"},
	{"supplier.arrive.12", "人民网推送量(篇)"},
	{"supplier.arrive.14", "清博推送量(篇)"},
	{"sub.outage.sites", "疑似整站故障站点数"},
	{"eff.e2e.weixin.le30m", "公众号端到端≤30min占比(%)"},
}

const aiSummarySystem = `你是数据平台的值班分析师，负责给团队写日报的一句话解读。输入是近几天的核心指标序列（最后一列是今天）和今天的告警列表。要求：
1. 3-4 句话、120 字以内，先说整体结论（平稳/好转/恶化），再点出最值得关注的 1-2 件事
2. 结合序列说趋势（如"连续三天下降""恢复到上周水平"），不要逐条复述数字
3. 告警项要判断是新出现的还是持续存在的
4. 讲人话，不用黑话不用表格，不要客套开场白，直接输出正文`

// aiDailySummary 生成 AI 解读文本；任何失败返回空串（旁路降级）。
func aiDailySummary(ctx context.Context, day time.Time, results []*report.Result, prevDays []report.Snapshot) string {
	cfg := conf.GetConfig().NewApi
	if cfg.Host == "" || cfg.Model == "" {
		return ""
	}

	// 组装上下文：每个核心指标一行，近 7 天 + 今天的值序列
	all := map[string]report.Metric{}
	var hits []report.Hit
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.Output != nil {
			for _, m := range r.Output.Metrics {
				all[m.Key] = m
			}
		}
		hits = append(hits, r.Hits...)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "报表日：%s（%s）\n\n核心指标（历史→今天）：\n", day.Format("2006-01-02"), weekdayCN(day))
	for _, k := range aiContextKeys {
		vals := make([]string, 0, len(prevDays)+1)
		for i := len(prevDays) - 1; i >= 0; i-- { // prevDays[0]=昨天，倒序成时间正序
			if prevDays[i] == nil {
				vals = append(vals, "-")
			} else if v, ok := prevDays[i][k.key]; ok {
				vals = append(vals, fmtI(v))
			} else {
				vals = append(vals, "-")
			}
		}
		if m, ok := all[k.key]; ok {
			vals = append(vals, m.Text)
		} else {
			vals = append(vals, "-")
		}
		fmt.Fprintf(&b, "%s: %s\n", k.name, strings.Join(vals, " → "))
	}
	b.WriteString("\n今天的告警：\n")
	if len(hits) == 0 {
		b.WriteString("（无）\n")
	}
	for _, h := range hits {
		fmt.Fprintf(&b, "- %s %s\n", h.Level.Icon(), h.Msg)
	}

	text, err := newApiMessages(ctx, cfg, aiSummarySystem, b.String(), 500)
	if err != nil {
		hlog.CtxErrorf(ctx, "[ai-summary] %v", err)
		return ""
	}
	return strings.TrimSpace(text)
}

func weekdayCN(t time.Time) string {
	return [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[t.Weekday()]
}

func fmtI(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.1f", v)
}

// newApiMessages 调 new-api 网关的 Anthropic 兼容 Messages 接口（口径同 resource/http/new_api.go）。
func newApiMessages(ctx context.Context, cfg conf.NewApi, system, userContent string, maxTokens int) (string, error) {
	reqBody, _ := sonic.Marshal(map[string]any{
		"model":      cfg.Model,
		"max_tokens": maxTokens,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": userContent}},
		// 短任务统一关思考，降延迟（resource 侧实测 qwen 系列 55s→0.6s）
		"thinking": map[string]string{"type": "disabled"},
	})
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_, err := httplib.Do(ctx, "https://"+cfg.Host+"/v1/messages", map[string]string{
		consts.HeaderContentType: consts.MIMEApplicationJSON,
		"x-api-key":              cfg.Key,
		"anthropic-version":      "2023-06-01",
	}, reqBody, &resp)
	if err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("new-api: %s", resp.Error.Message)
	}
	for _, blk := range resp.Content {
		if blk.Type == "text" && blk.Text != "" {
			return blk.Text, nil
		}
	}
	return "", fmt.Errorf("new-api: empty content")
}
