package handlers

// 公众号文章入 Feed 核验（1.2 缺失口径）。
// 完整移植自语鲸 fc_v2/topic-monitor 的监控9（monitor_9_wechat_article_in_feed），
// 与其发的「自建抓取公众号文章入库语鲸监控」卡片同口径：
//   抓取文章 = spider 库 create_time 落在当日（北京墙钟自然日）的文章，按账号+标题去重，
//              且账号能解析出 fakeid 并在 author_map 里有 channel_id（可核验数 = 进入 + 缺失）；
//   进入语鲸 = 按账号调订阅 Feed iapi 翻页，标题能在 Feed 中查到；
//   缺失     = Feed 中查不到标题；缺失率 = 缺失 ÷ 可核验。
// author_map.json 与监控9同一份静态文件（fakeid→channel_id，由 export_account_channel_mapping 导出），
// 两边要同步更新，否则数字会漂。

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"sync"
	"time"

	"empyrean_lens/fc/cron/handlers/report"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

//go:embed author_map.json
var authorMapJSON []byte

const (
	feedInnerAPIBase = "http://api-inner.lingowhale.com"
	feedSubPath      = "/iapi/lingowhale/v1/feed/subscription"
	feedPageLimit    = 50
	feedMaxPages     = 40
	feedConcurrency  = 8
)

type feedResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		FeedList []struct {
			Title string `json:"title"`
		} `json:"feed_list"`
		Cursor  string `json:"cursor"`
		HasMore bool   `json:"has_more"`
	} `json:"data"`
}

// wechatFeedCheck 返回单行：checked / hit / missing / unmapped_accounts / unresolved_titles。
func wechatFeedCheck(ctx context.Context, from, to time.Time) (report.Rows, error) {
	// 1. 当日落库文章（spider 库为 naive 北京墙钟，MongoNaiveCST 处理窗口）
	articles, err := queryFunc(ctx, report.Query{
		Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", MongoNaiveCST: true,
		PipelineJSON: `[
		  {"$match": {"create_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}},
		  {"$project": {"_id": 0, "title": 1, "acct": {"$toString": "$target_account"}}},
		  {"$limit": 30000}]`,
	}, from, to)
	if err != nil {
		return nil, fmt.Errorf("spider article: %w", err)
	}
	// 按账号 ObjectId 分组、标题去重（与监控9一致）
	byOid := map[string]map[string]bool{}
	for _, row := range articles {
		t, acct := strings.TrimSpace(row["title"]), row["acct"]
		if t == "" || acct == "" {
			continue
		}
		if byOid[acct] == nil {
			byOid[acct] = map[string]bool{}
		}
		byOid[acct][t] = true
	}

	// 2. ObjectId → fakeid（$toString 匹配，避免依赖代理的 $oid 扩展语法）
	oidToFakeid := map[string]string{}
	oids := make([]string, 0, len(byOid))
	for oid := range byOid {
		oids = append(oids, oid)
	}
	for i := 0; i < len(oids); i += 200 {
		j := min(i+200, len(oids))
		quoted := make([]string, 0, j-i)
		for _, o := range oids[i:j] {
			b, _ := sonic.MarshalString(o)
			quoted = append(quoted, b)
		}
		rows, err := queryFunc(ctx, report.Query{
			Source: report.SourceMongo, DB: "wechat-spider", Collection: "target_account", MongoNaiveCST: true,
			PipelineJSON: `[{"$match": {"$expr": {"$in": [{"$toString": "$_id"}, [` + strings.Join(quoted, ",") + `]]}}},
			  {"$project": {"_id": 0, "id": {"$toString": "$_id"}, "fakeid": 1}}]`,
		}, from, to)
		if err != nil {
			return nil, fmt.Errorf("target_account batch: %w", err)
		}
		for _, r := range rows {
			if r["fakeid"] != "" {
				oidToFakeid[r["id"]] = r["fakeid"]
			}
		}
	}

	// 3. fakeid → channel_id（与监控9同一份静态映射）
	authorMap := map[string]string{}
	if err := sonic.Unmarshal(authorMapJSON, &authorMap); err != nil {
		return nil, fmt.Errorf("author_map.json: %w", err)
	}
	// 按 fakeid 合并（同 fakeid 多个 ObjectId）
	byFakeid := map[string]map[string]bool{}
	var unresolvedTitles int
	for oid, titles := range byOid {
		fk := oidToFakeid[oid]
		if fk == "" {
			unresolvedTitles += len(titles)
			continue
		}
		if byFakeid[fk] == nil {
			byFakeid[fk] = map[string]bool{}
		}
		for t := range titles {
			byFakeid[fk][t] = true
		}
	}

	// 4. 逐账号 Feed 核验（命中即停翻页），并发限流
	type acctJob struct {
		channelID string
		titles    map[string]bool
	}
	var jobs []acctJob
	var unmappedAccounts, checked int
	for fk, titles := range byFakeid {
		cid := authorMap[fk]
		if cid == "" {
			unmappedAccounts++
			continue
		}
		checked += len(titles)
		jobs = append(jobs, acctJob{channelID: cid, titles: titles})
	}

	var mu sync.Mutex
	var hit, missing, feedErrs int
	sem := make(chan struct{}, feedConcurrency)
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(job acctJob) {
			defer func() { wg.Done(); <-sem }()
			h, err := fetchFeedHits(ctx, job.channelID, job.titles)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				// 单账号接口失败按全缺失计会误报，按错误单独计数并整体降级
				feedErrs++
				hlog.CtxErrorf(ctx, "[feedcheck] channel %s: %v", job.channelID, err)
				return
			}
			hit += h
			missing += len(job.titles) - h
		}(job)
	}
	wg.Wait()
	if feedErrs > 0 && feedErrs*10 > len(jobs) {
		// 超过 10% 的账号核验失败，数字不可信
		return nil, fmt.Errorf("feed 核验失败账号过多: %d/%d", feedErrs, len(jobs))
	}

	return report.Rows{{
		"checked":  fmt.Sprintf("%d", checked),
		"hit":      fmt.Sprintf("%d", hit),
		"missing":  fmt.Sprintf("%d", missing),
		"unmapped": fmt.Sprintf("%d", unmappedAccounts),
		"errs":     fmt.Sprintf("%d", feedErrs),
	}}, nil
}

// fetchFeedHits 翻页拉取单频道 Feed，返回 want 中命中的标题数。命中全部即提前停。
func fetchFeedHits(ctx context.Context, channelID string, want map[string]bool) (int, error) {
	found := map[string]bool{}
	cursor := ""
	for page := 0; page < feedMaxPages; page++ {
		body, _ := sonic.Marshal(map[string]any{
			"cursor": cursor, "channel_ids": []string{channelID}, "limit": feedPageLimit,
		})
		resp := &feedResp{}
		_, err := httplib.Do(ctx, feedInnerAPIBase+feedSubPath, map[string]string{
			consts.HeaderContentType: consts.MIMEApplicationJSON,
		}, body, resp)
		if err != nil {
			return 0, err
		}
		if resp.Code != 0 {
			return 0, fmt.Errorf("feed api code=%d msg=%s", resp.Code, resp.Msg)
		}
		for _, item := range resp.Data.FeedList {
			t := strings.TrimSpace(item.Title)
			if t != "" && want[t] {
				found[t] = true
			}
		}
		if len(found) >= len(want) {
			break
		}
		cursor = resp.Data.Cursor
		if !resp.Data.HasMore || cursor == "" {
			break
		}
	}
	return len(found), nil
}
