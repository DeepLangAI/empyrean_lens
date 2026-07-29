package handlers

// 整站故障告警的病因分流（2026-07-28 拍板）：
//   对端自身死亡（连不上/DNS 失败/5xx 服务故障）→ 我方无可行动作 → 不告警，只留文档注记；
//   对端活着但抓取全灭（2xx/3xx/401/403/429）→ 疑似反爬/封 IP，我方可换代理处置 → 保留告警。
// 判定手段：报表 FC 从自己的网络对故障站做一次独立探活（与爬虫不同出口，能区分
// "对全世界死" vs "只对爬虫死"）。探测失败跑不出结论时保守保留告警（宁误报不漏报）。

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"empyrean_lens/fc/cron/handlers/report"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

var outageEntryRe = regexp.MustCompile(`([a-z0-9.-]+\.[a-z]{2,})\([^)]*\)`)

// probeSite 返回 true=站点对外活着（我方被拦，可告警）；false=对端死亡/服务故障。
// 5xx 视为对端服务故障（zhenglian 案例：nginx 活着但后端 502，同属"对端的问题"）。
func probeSite(ctx context.Context, domain string) (alive bool, detail string) {
	cli := &http.Client{Timeout: 8 * time.Second}
	for _, scheme := range []string{"https", "http"} {
		req, err := http.NewRequestWithContext(ctx, "GET", scheme+"://"+domain+"/", nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
		resp, err := cli.Do(req)
		if err != nil {
			detail = "连接失败"
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			return false, fmt.Sprintf("HTTP %d(服务故障)", resp.StatusCode)
		}
		return true, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return false, detail
}

// classifyOutages 对 sub.outage.sites 命中的站点做探活分流，原地改写该节的 Hits 与 Notes。
func classifyOutages(ctx context.Context, results []*report.Result) {
	for _, r := range results {
		if r == nil || r.Output == nil || r.Section.Key != "sub" {
			continue
		}
		// 找到整站故障命中（可能不存在）
		hitIdx := -1
		for i, h := range r.Hits {
			if h.MetricKey == "sub.outage.sites" {
				hitIdx = i
				break
			}
		}
		if hitIdx < 0 {
			return
		}
		// 从指标 Text 里取站点清单，如 "rsshub.x.me(68个源 98.9%)、pubs.acs.org(59个源 100.0%)"
		var siteText string
		for _, m := range r.Output.Metrics {
			if m.Key == "sub.outage.sites" {
				siteText = m.Text
			}
		}
		entries := outageEntryRe.FindAllStringSubmatch(siteText, -1)
		if len(entries) == 0 {
			return
		}
		var blocked, dead []string
		for _, e := range entries {
			full, domain := e[0], e[1]
			alive, detail := probeSite(ctx, domain)
			hlog.CtxInfof(ctx, "[outage-probe] %s alive=%v (%s)", domain, alive, detail)
			if alive {
				blocked = append(blocked, fmt.Sprintf("%s[探活%s]", full, detail))
			} else {
				dead = append(dead, fmt.Sprintf("%s[%s]", full, detail))
			}
		}
		// 重写命中：只有"站点活着但抓取全灭"才保留告警
		newHits := append(r.Hits[:hitIdx:hitIdx], r.Hits[hitIdx+1:]...)
		if len(blocked) > 0 {
			lvl := report.LevelWarn
			if len(blocked) >= 3 {
				lvl = report.LevelCrit
			}
			newHits = append(newHits, report.Hit{
				Level: lvl, MetricKey: "sub.outage.sites",
				Msg: fmt.Sprintf("订阅源整站抓取失败但站点对外可达：%s——疑似我方出口被拦", strings.Join(blocked, "、")),
			})
		}
		if len(dead) > 0 {
			r.Output.Notes = append(r.Output.Notes,
				fmt.Sprintf("对端自身故障的失败站点（我方无可行动作，不计告警）：%s", strings.Join(dead, "、")))
		}
		r.Hits = newHits
		// 重算该节级别（去掉/降级命中后可能回绿）
		lvl := report.LevelOK
		for _, h := range r.Hits {
			if h.Level > lvl {
				lvl = h.Level
			}
		}
		r.Level = lvl
		return
	}
}
