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

// 渠道/内容类型：从指标 Key 的已知后缀推导（Phase 1 用映射表，Phase 2 让 Metric 自带 Dims）。
// 两类切片正交、各占一列（渠道=从哪条路进来，类型=内容长什么样，一篇公众号文章
// 可能来自任一渠道）；混在一列会让筛选下拉渠道和类型掺在一起，2026-07-15 拆开。
var channelSuffix = map[string]string{
	"12": "人民网", "14": "清博", "11": "自采集", "15": "小宇宙",
	"SubRSS": "自有RSS", "SubWeb": "自有网站",
	"Renminwang": "人民网", "Qingbo": "清博", "FromMonitoring": "自采集",
}

var contentTypeSuffix = map[string]string{
	"weixin": "公众号", "web": "网站", "pdf": "PDF",
}

func metricDims(key string) (channel, contentType string) {
	parts := strings.Split(key, ".")
	for i := len(parts) - 1; i > 0; i-- {
		if c, ok := channelSuffix[parts[i]]; ok && channel == "" {
			channel = c
		}
		if t, ok := contentTypeSuffix[parts[i]]; ok && contentType == "" {
			contentType = t
		}
	}
	return
}


// deleteDayRecords 删除指标日表中指定日期的全部记录（写入前清场，保证幂等）。
func deleteDayRecords(ctx context.Context, token string, cfg conf.MetricsSink, dayMs int64) error {
	searchURL := fmt.Sprintf("%s/bitable/v1/apps/%s/tables/%s/records/search?page_size=500",
		feishuOpenBase, cfg.BitableAppToken, cfg.MetricsTableID)
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
			feishuOpenBase, cfg.BitableAppToken, cfg.MetricsTableID)
		if _, err := feishuCall(ctx, "POST", delURL, token, map[string]any{"records": ids}); err != nil {
			return err
		}
		if !data.HasMore {
			return nil
		}
	}
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
	// 幂等：先删同日旧记录再写（重跑/回填不产生重复行）
	if err := deleteDayRecords(ctx, token, cfg, dayMs); err != nil {
		hlog.CtxErrorf(ctx, "[metrics-sink] delete old records: %v", err)
		return
	}
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
			if ch, ct := metricDims(m.Key); true {
				if ch != "" {
					fields["渠道"] = ch
				}
				if ct != "" {
					fields["内容类型"] = ct
				}
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
