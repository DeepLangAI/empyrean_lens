package handlers

// 公众号文章入 Feed 核验（1.2 缺失口径）。
// 完整移植自语鲸 fc_v2/topic-monitor 的监控9（monitor_9_wechat_article_in_feed），
// 与其发的「自建抓取公众号文章入库语鲸监控」卡片同口径：
//   抓取文章 = spider 库 create_time 落在当日（北京墙钟自然日）的文章，按账号+标题去重，
//              且账号能解析出 fakeid 并在 author_map 里有 channel_id（可核验数 = 进入 + 缺失）；
//   进入语鲸 = 按账号调订阅 Feed iapi 翻页，标题能在 Feed 中查到；
//   缺失     = Feed 中查不到标题；缺失率 = 缺失 ÷ 可核验。
// 在监控9口径基础上剔除作者删文（2026-07 核对定性：缺失主因）：缺失文章按 URL 回查
// 资源库 content 头部，命中微信删文模板（与失败明细分类同款标记）视为正常淘汰，
// 从缺失与可核验中同时剔除、单列计数。回查失败或查无资源时保守计缺失。
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
	// 1. 当日落库文章（spider 库为 naive 北京墙钟，MongoNaiveCST 处理窗口）。
	// 代理单次响应上限 512KB（约 4500 篇标题即超），必须分页拉取。
	const articlePage = 2000
	byOid := map[string]map[string]string{} // 账号 ObjectId → 去重标题集（与监控9一致），值为文章 URL（删文回查用）
	for skip := 0; skip < 50000; skip += articlePage {
		articles, err := queryFunc(ctx, report.Query{
			Source: report.SourceMongo, DB: "wechat-spider", Collection: "article", MongoNaiveCST: true,
			PipelineJSON: fmt.Sprintf(`[
			  {"$match": {"create_time": {"$gte": {"$date": "{{DAY_START}}"}, "$lt": {"$date": "{{DAY_END}}"}}}},
			  {"$sort": {"_id": 1}}, {"$skip": %d}, {"$limit": %d},
			  {"$project": {"_id": 0, "title": 1, "url": 1, "acct": {"$toString": "$target_account"}}}]`, skip, articlePage),
		}, from, to)
		if err != nil {
			return nil, fmt.Errorf("spider article (skip=%d): %w", skip, err)
		}
		for _, row := range articles {
			t, acct := firstLine(row["title"]), row["acct"]
			if t == "" || acct == "" {
				continue
			}
			if byOid[acct] == nil {
				byOid[acct] = map[string]string{}
			}
			byOid[acct][t] = row["url"]
		}
		if len(articles) < articlePage {
			break
		}
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
	byFakeid := map[string]map[string]string{}
	var unresolvedTitles int
	for oid, titles := range byOid {
		fk := oidToFakeid[oid]
		if fk == "" {
			unresolvedTitles += len(titles)
			continue
		}
		if byFakeid[fk] == nil {
			byFakeid[fk] = map[string]string{}
		}
		for t, u := range titles {
			byFakeid[fk][t] = u
		}
	}

	// 4. 逐账号 Feed 核验（命中即停翻页），并发限流
	type acctJob struct {
		channelID string
		titles    map[string]string // 标题 → URL
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

	// 熔断探测：先拨一个账号，连不通（重试一次仍失败）直接放弃整个核验。
	// 环境不可达时避免上千次无谓拨号——实测拨号风暴会塞满进程的 DNS 解析队列，
	// 殃及同进程后续所有网络调用（本机复现两次，均死在紧随其后的查询上）。
	if len(jobs) > 0 {
		if _, err := fetchFeedHits(ctx, jobs[0].channelID, jobs[0].titles); err != nil {
			if _, err2 := fetchFeedHits(ctx, jobs[0].channelID, jobs[0].titles); err2 != nil {
				return nil, fmt.Errorf("feed 接口不可达（探测失败）: %w", err2)
			}
		}
	}

	var mu sync.Mutex
	var hit, missing, feedErrs int
	var missingURLs []string // 每篇缺失一项（URL 可能为空，空的无法回查删文）
	sem := make(chan struct{}, feedConcurrency)
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(job acctJob) {
			defer func() { wg.Done(); <-sem }()
			found, err := fetchFeedHits(ctx, job.channelID, job.titles)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				// 单账号接口失败按全缺失计会误报，按错误单独计数并整体降级
				feedErrs++
				hlog.CtxErrorf(ctx, "[feedcheck] channel %s: %v", job.channelID, err)
				return
			}
			hit += len(found)
			for t, u := range job.titles {
				if !found[t] {
					missing++
					missingURLs = append(missingURLs, u)
				}
			}
		}(job)
	}
	wg.Wait()
	if feedErrs > 0 && feedErrs*10 > len(jobs) {
		// 超过 10% 的账号核验失败，数字不可信
		return nil, fmt.Errorf("feed 核验失败账号过多: %d/%d", feedErrs, len(jobs))
	}

	// 5. 缺失文章删文回查：作者删文是正常淘汰，从缺失与可核验中同时剔除
	deletedSet, foundSet := lookupDeletedByURL(ctx, missingURLs, from, to)
	var deleted, hasRes, noRes int
	var sample []string
	for _, u := range missingURLs {
		switch {
		case u != "" && deletedSet[u]:
			deleted++
		case u != "" && foundSet[u]:
			hasRes++ // 有资源但正文非删文模板：真缺失，URL 留日志供排查
			if len(sample) < 10 {
				sample = append(sample, u)
			}
		default:
			noRes++ // 查无资源（未进处理链路或 URL 形式不一致）
			if len(sample) < 10 {
				sample = append(sample, u)
			}
		}
	}
	hlog.CtxInfof(ctx, "[feedcheck] missing=%d → 删文 %d / 有资源非删文 %d / 查无资源 %d，样例: %v",
		missing, deleted, hasRes, noRes, sample)

	return report.Rows{{
		"checked":  fmt.Sprintf("%d", checked-deleted),
		"hit":      fmt.Sprintf("%d", hit),
		"missing":  fmt.Sprintf("%d", missing-deleted),
		"deleted":  fmt.Sprintf("%d", deleted),
		"unmapped": fmt.Sprintf("%d", unmappedAccounts),
		"errs":     fmt.Sprintf("%d", feedErrs),
	}}, nil
}

// firstLine 取标题的第一行非空文本做匹配键。
// 段子体/朋友圈体短文没有独立标题，spider 从微信列表页拿到的 title 是带换行的全文，
// 而语鲸解析只取首行做标题——两边整串精确匹配永远失败，曾把在库在 Feed 的文章
// 记成缺失（2026-07-27 定性：当日 23 篇"缺失"主体即此），归一到首行后再比对。
func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}

// isDeletedContentHead 判断资源正文头部是否为微信删文/违规不可见提示模板
// （与失败明细分类「文章已删除(作者删文)」同款标记，见 ops_sink.go classifyFailReason）。
// "发送失败无法查看" = 平台判违规下架（"此内容因涉嫌违反相关法律法规和政策发送失败"），
// 与作者删文同性质：原文对用户已不可见，属正常淘汰。
func isDeletedContentHead(head string) bool {
	return strings.Contains(head, "content has been deleted") ||
		strings.Contains(head, "已被发布者删除") ||
		strings.Contains(head, "内容因违规") ||
		strings.Contains(head, "发送失败无法查看")
}

// lookupDeletedByURL 按 orig_url 回查资源库正文头部，返回命中删文模板的 URL 集合
// 和查到资源的 URL 集合（诊断分桶用）。回查失败只打日志（缺失不剔除，保守方向）。
func lookupDeletedByURL(ctx context.Context, urls []string, from, to time.Time) (map[string]bool, map[string]bool) {
	uniq := map[string]bool{}
	for _, u := range urls {
		if u != "" {
			uniq[u] = true
		}
	}
	deleted, found := map[string]bool{}, map[string]bool{}
	if len(uniq) == 0 {
		return deleted, found
	}
	batch := make([]string, 0, len(uniq))
	for u := range uniq {
		batch = append(batch, u)
	}
	for i := 0; i < len(batch); i += 100 {
		j := min(i+100, len(batch))
		inJSON, _ := sonic.MarshalString(batch[i:j])
		rows, err := queryFunc(ctx, report.Query{
			Source: report.SourceMongo, DB: "lingowhale_plugin", Collection: "resource",
			PipelineJSON: `[{"$match": {"orig_url": {"$in": ` + inJSON + `}}},
			  {"$project": {"_id": 0, "u": "$orig_url", "head": {"$substrCP": [{"$ifNull": ["$content", ""]}, 0, 60]}}}]`,
		}, from, to)
		if err != nil {
			hlog.CtxErrorf(ctx, "[feedcheck] deleted lookup batch %d: %v", i, err)
			continue
		}
		for _, r := range rows {
			found[r["u"]] = true
			if isDeletedContentHead(r["head"]) {
				deleted[r["u"]] = true
			}
		}
	}
	return deleted, found
}

// fetchFeedHits 翻页拉取单频道 Feed，返回 want 中命中的标题集。命中全部即提前停。
func fetchFeedHits(ctx context.Context, channelID string, want map[string]string) (map[string]bool, error) {
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
			return nil, err
		}
		if resp.Code != 0 {
			return nil, fmt.Errorf("feed api code=%d msg=%s", resp.Code, resp.Msg)
		}
		for _, item := range resp.Data.FeedList {
			t := firstLine(item.Title)
			if t != "" {
				if _, ok := want[t]; ok {
					found[t] = true
				}
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
	return found, nil
}
