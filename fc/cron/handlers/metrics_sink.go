package handlers

// 巡检指标落飞书多维表格（数据平台知识库「数据表」）。
// 设计约束（详见 docs/metrics-tiering.md）：
//   - 指标日表是窄表：一行一个指标值，日报改版不动表结构；
//   - 全量指标都写（核心/诊断分档在字典表，展示层用视图筛）；
//   - 写入失败只记日志，绝不影响日报发送（sink 是旁路，不是主链路）。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"empyrean_lens/conf"
	"empyrean_lens/fc/cron/handlers/report"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"github.com/avast/retry-go"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	consts "github.com/cloudwego/hertz/pkg/protocol/consts"
)

const feishuOpenBase = "https://open.feishu.cn/open-apis"

// tenantToken 获取 tenant_access_token（有效期 2h，FC 单次执行内不缓存也够用）。
// 网络类错误重试 3 次（DNS 抖动实测出现过）。
func tenantToken(ctx context.Context, appID, appSecret string) (string, error) {
	body, _ := sonic.Marshal(map[string]string{"app_id": appID, "app_secret": appSecret})
	var resp struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Token string `json:"tenant_access_token"`
	}
	err := retry.Do(func() error {
		_, err := httplib.Do(ctx, feishuOpenBase+"/auth/v3/tenant_access_token/internal",
			map[string]string{consts.HeaderContentType: consts.MIMEApplicationJSON}, body, &resp)
		return err
	}, retry.Attempts(3), retry.Delay(time.Second), retry.Context(ctx))
	if err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("tenant token code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.Token, nil
}

// 层级展示名：与字典表选项一致。gen 节单独归"生成服务"档。
func layerDisplay(sectionKey, layer string) string {
	if sectionKey == "gen" {
		return "生成服务"
	}
	switch layer {
	case report.LayerL1:
		return "来源层"
	case report.LayerL2:
		return "处理层"
	case report.LayerL3:
		return "入库层"
	}
	return ""
}

// 维度：从指标 Key 的已知后缀推导（Phase 1 用映射表，Phase 2 让 Metric 自带 Dims）。
var dimSuffix = map[string]string{
	"12": "供应商=人民网", "14": "供应商=清博", "11": "渠道=自采集", "15": "渠道=小宇宙",
	"SubRSS": "渠道=自有RSS", "SubWeb": "渠道=自有网站",
	"Renminwang": "渠道=人民网", "Qingbo": "渠道=清博", "FromMonitoring": "渠道=自采集",
	"weixin": "类型=公众号", "web": "类型=网站", "pdf": "类型=PDF",
}

func metricDim(key string) string {
	parts := strings.Split(key, ".")
	for i := len(parts) - 1; i > 0; i-- {
		if d, ok := dimSuffix[parts[i]]; ok {
			return d
		}
	}
	return ""
}


// sinkMetrics 把当日全量指标批量写入指标日表。失败只记日志。
func sinkMetrics(ctx context.Context, day time.Time, results []*report.Result) {
	cfg := conf.GetConfig().MetricsSink
	if cfg.BitableAppToken == "" || cfg.MetricsTableID == "" {
		return
	}
	token, err := tenantToken(ctx, cfg.FeishuAppID, cfg.FeishuAppSecret)
	if err != nil {
		hlog.CtxErrorf(ctx, "[metrics-sink] token: %v", err)
		return
	}

	dayMs := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, cstLoc).UnixMilli()
	var records []map[string]any
	for _, r := range results {
		if r == nil || r.Output == nil {
			continue
		}
		// 指标 → 命中状态（无命中即 🟢）
		hitLevel := map[string]report.Level{}
		for _, h := range r.Hits {
			if h.MetricKey != "" && h.Level > hitLevel[h.MetricKey] {
				hitLevel[h.MetricKey] = h.Level
			}
		}
		for _, m := range r.Output.Metrics {
			status := "🟢"
			switch hitLevel[m.Key] {
			case report.LevelWarn:
				status = "🟡"
			case report.LevelCrit:
				status = "🔴"
			}
			fields := map[string]any{
				"日期":   dayMs,
				"指标Key": m.Key,
				"指标名":  m.Display,
				"层级":   layerDisplay(r.Section.Key, r.Section.Layer),
				"数值":   m.Value,
				"展示值":  m.Text,
				"状态":   status,
				"是否异常": hitLevel[m.Key] > report.LevelOK,
			}
			if d := metricDim(m.Key); d != "" {
				fields["维度"] = d
			}
			if u := string(m.Dimension); u != "" {
				fields["单位"] = u // Dimension 值本身就是单位名，与表单选项对齐
			}
			records = append(records, map[string]any{"fields": fields})
		}
	}
	if len(records) == 0 {
		return
	}

	// batch_create 单次上限 500，当前 130+/天，一批即可；留分批防未来膨胀
	for i := 0; i < len(records); i += 400 {
		j := min(i+400, len(records))
		body, _ := sonic.Marshal(map[string]any{"records": records[i:j]})
		var resp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		url := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/batch_create",
			feishuOpenBase, cfg.BitableAppToken, cfg.MetricsTableID)
		_, err := httplib.Do(ctx, url, map[string]string{
			consts.HeaderContentType: consts.MIMEApplicationJSON,
			"Authorization":          "Bearer " + token,
		}, body, &resp)
		if err != nil || resp.Code != 0 {
			hlog.CtxErrorf(ctx, "[metrics-sink] batch_create [%d:%d]: err=%v code=%d msg=%s", i, j, err, resp.Code, resp.Msg)
			return
		}
	}
	hlog.CtxInfof(ctx, "[metrics-sink] %d metrics written for %s", len(records), day.Format("2006-01-02"))
}
