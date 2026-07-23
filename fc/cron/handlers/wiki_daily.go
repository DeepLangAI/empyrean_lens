package handlers

// 日报归档到飞书知识库（数据平台知识库「每日日报」节点）。
// 复用渲染层产物 []fcMsg 转 docx 块，布局与群卡片完全一致：
//   每日日报/<YYYY-MM>/<MM-DD 状态> 一天一篇，标题带状态灯——目录即异常时间轴。
// 与 metrics_sink 一样是旁路：失败只记日志，不影响发报。

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"empyrean_lens/conf"

	"github.com/avast/retry-go"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	consts "github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ─── 飞书开放平台基础调用 ────────────────────────────────────────────────────

type feishuResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data sonic.NoCopyRawMessage `json:"data"`
}

func feishuCall(ctx context.Context, method, url, token string, payload any) (*feishuResp, error) {
	var body []byte
	if payload != nil {
		body, _ = sonic.Marshal(payload)
	}
	resp := &feishuResp{}
	// 网络类错误重试 3 次（DNS 抖动实测出现过）；业务错误码不重试
	err := retry.Do(func() error {
		_, err := httplib.Do(ctx, url, map[string]string{
			consts.HeaderContentType: consts.MIMEApplicationJSON,
			"Authorization":          "Bearer " + token,
		}, body, resp, httplib.WithHttpMethod(method))
		return err
	}, retry.Attempts(3), retry.Delay(time.Second), retry.Context(ctx))
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("%s %s: code=%d msg=%s", method, url, resp.Code, resp.Msg)
	}
	return resp, nil
}

// ─── wiki 节点 ───────────────────────────────────────────────────────────────

type wikiNode struct {
	NodeToken string `json:"node_token"`
	ObjToken  string `json:"obj_token"`
	Title     string `json:"title"`
}

func wikiChildren(ctx context.Context, token, spaceID, parent string) ([]wikiNode, error) {
	url := fmt.Sprintf("%s/wiki/v2/spaces/%s/nodes?page_size=50&parent_node_token=%s", feishuOpenBase, spaceID, parent)
	resp, err := feishuCall(ctx, "GET", url, token, nil)
	if err != nil {
		return nil, err
	}
	var data struct {
		Items []wikiNode `json:"items"`
	}
	_ = sonic.Unmarshal(resp.Data, &data)
	return data.Items, nil
}

func wikiCreateNode(ctx context.Context, token, spaceID, parent, title string) (*wikiNode, error) {
	url := fmt.Sprintf("%s/wiki/v2/spaces/%s/nodes", feishuOpenBase, spaceID)
	resp, err := feishuCall(ctx, "POST", url, token, map[string]any{
		"obj_type": "docx", "parent_node_token": parent, "node_type": "origin", "title": title,
	})
	if err != nil {
		return nil, err
	}
	var data struct {
		Node wikiNode `json:"node"`
	}
	_ = sonic.Unmarshal(resp.Data, &data)
	return &data.Node, nil
}

// ─── fcMsg → docx 块 ─────────────────────────────────────────────────────────

type wkBlock = map[string]any

func txtEls(s string) []map[string]any {
	return []map[string]any{{"text_run": map[string]any{"content": s}}}
}

// mdLinkRe 匹配 markdown 链接 [text](url)。卡片 markdown 原生支持；docx 需转 text_run link。
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s)]+)\)`)

// txtElsMd 同 txtEls，但把 markdown 链接转为 docx 超链接元素（link.url 需整体 URL 编码）。
func txtElsMd(s string) []map[string]any {
	ms := mdLinkRe.FindAllStringSubmatchIndex(s, -1)
	if len(ms) == 0 {
		return txtEls(s)
	}
	var els []map[string]any
	last := 0
	for _, m := range ms {
		if m[0] > last {
			els = append(els, map[string]any{"text_run": map[string]any{"content": s[last:m[0]]}})
		}
		els = append(els, map[string]any{"text_run": map[string]any{
			"content":            s[m[2]:m[3]],
			"text_element_style": map[string]any{"link": map[string]any{"url": url.QueryEscape(s[m[4]:m[5]])}},
		}})
		last = m[1]
	}
	if last < len(s) {
		els = append(els, map[string]any{"text_run": map[string]any{"content": s[last:]}})
	}
	return els
}

// blockBatch 一批要追加到文档根的块（含嵌套后代）。
type blockBatch struct {
	seq      int
	Children []string  // 顶层块 id
	Blocks   []wkBlock // 全部块（含单元格等后代）
}

func (b *blockBatch) add(blockType int, key string, payload map[string]any, children []string, top bool) string {
	b.seq++
	id := fmt.Sprintf("blk%d", b.seq)
	blk := wkBlock{"block_id": id, "block_type": blockType, key: payload}
	if len(children) > 0 {
		blk["children"] = children
	}
	b.Blocks = append(b.Blocks, blk)
	if top {
		b.Children = append(b.Children, id)
	}
	return id
}

var mdCleaner = strings.NewReplacer("**", "", "~~", "", "<u>", "", "</u>", "")

var reSectionNo = regexp.MustCompile(`^\d+\.\d+\s`)

// mdToBlocks 把卡片 markdown 文本转 docx 块（按行拆）。
// 层标题（第X层/四、告警/健康总览）升 H2、小节标题（1.1 这类）升 H3——
// 文档目录树由此立起来，一眼可跳转。
func (b *blockBatch) mdToBlocks(content string) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(mdCleaner.Replace(line), " ")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		noIcon := strings.TrimSpace(strings.TrimLeft(trimmed, "🟢🟡🔴⚪⚠️❌✅"))
		switch {
		case strings.HasPrefix(line, "> "):
			b.add(15, "quote", map[string]any{"elements": txtEls(strings.TrimPrefix(line, "> "))}, nil, true)
		case strings.HasPrefix(line, "- "):
			b.add(12, "bullet", map[string]any{"elements": txtEls(strings.TrimPrefix(line, "- "))}, nil, true)
		case strings.HasPrefix(noIcon, "第一层") || strings.HasPrefix(noIcon, "第二层") ||
			strings.HasPrefix(noIcon, "第三层") || strings.HasPrefix(noIcon, "四、"):
			b.add(3, "heading1", map[string]any{"elements": txtEls(trimmed)}, nil, true)
		case reSectionNo.MatchString(noIcon):
			b.add(4, "heading2", map[string]any{"elements": txtEls(trimmed)}, nil, true)
		default:
			b.add(2, "text", map[string]any{"elements": txtElsMd(line)}, nil, true)
		}
	}
}

func (b *blockBatch) tableToBlocks(t fcTable) {
	rows, cols := len(t.Rows)+1, len(t.Columns)
	if cols == 0 {
		return
	}
	var cellIDs []string
	addCell := func(content string) {
		textID := b.add(2, "text", map[string]any{"elements": txtElsMd(content)}, nil, false)
		cellIDs = append(cellIDs, b.add(32, "table_cell", map[string]any{}, []string{textID}, false))
	}
	for _, c := range t.Columns {
		addCell(c.DisplayName)
	}
	for _, row := range t.Rows {
		for _, c := range t.Columns {
			addCell(row[c.Name])
		}
	}
	// 列宽：沿用卡片的百分比提示换算成像素（docx 正文可用宽度约 720px），
	// auto/未标列均分剩余。不带列宽时 docx 全列等宽，长文字列会挤成多行。
	const tblWidth = 720
	widths := make([]int, cols)
	remain, autos := tblWidth, 0
	for i, c := range t.Columns {
		if n, err := strconv.Atoi(strings.TrimSuffix(c.Width, "%")); err == nil && strings.HasSuffix(c.Width, "%") && n > 0 {
			w := max(tblWidth*n/100, 70)
			widths[i] = w
			remain -= w
			continue
		}
		autos++
	}
	if autos > 0 {
		each := max(remain/autos, 90)
		for i := range widths {
			if widths[i] == 0 {
				widths[i] = each
			}
		}
	}
	b.add(31, "table", map[string]any{
		"property": map[string]any{"row_size": rows, "column_size": cols, "header_row": true, "column_width": widths},
	}, cellIDs, true)
}

func appendBatch(ctx context.Context, token, docID string, b *blockBatch) error {
	if len(b.Children) == 0 {
		return nil
	}
	url := fmt.Sprintf("%s/docx/v1/documents/%s/blocks/%s/descendant", feishuOpenBase, docID, docID)
	_, err := feishuCall(ctx, "POST", url, token, map[string]any{
		"children_id": b.Children, "descendants": b.Blocks,
	})
	return err
}

// ─── 主流程 ──────────────────────────────────────────────────────────────────

// docTitle 形如 "07-13 🟢" / "07-13 🔴2·🟡1"，从卡片头部状态标签推导。
func docTitle(day time.Time, msgs []fcMsg) string {
	suffix := "🟢"
	if len(msgs) > 0 {
		var parts []string
		for _, tag := range msgs[0].Card.Header.TextTagList {
			c := tag.Text.Content
			switch {
			case strings.Contains(c, "严重"):
				parts = append(parts, "🔴"+strings.TrimSuffix(strings.TrimSpace(strings.Split(c, "项")[0]), " "))
			case strings.Contains(c, "关注"):
				parts = append(parts, "🟡"+strings.TrimSuffix(strings.TrimSpace(strings.Split(c, "项")[0]), " "))
			}
		}
		if len(parts) > 0 {
			suffix = strings.Join(parts, "·")
		}
	}
	return day.Format("01-02") + " " + suffix
}

// sinkWikiDaily 把日报写成知识库云文档，返回文档链接（失败返回空串并记日志）。
func sinkWikiDaily(ctx context.Context, day time.Time, msgs []fcMsg) string {
	cfg := conf.GetConfig().MetricsSink
	if cfg.WikiSpaceID == "" || cfg.DailyNodeToken == "" {
		return ""
	}
	token, err := tenantToken(ctx, cfg.FeishuAppID, cfg.FeishuAppSecret)
	if err != nil {
		hlog.CtxErrorf(ctx, "[wiki-daily] token: %v", err)
		return ""
	}

	// 月份节点（2026-07），无则建
	monthTitle := day.Format("2006-01")
	var month *wikiNode
	nodes, err := wikiChildren(ctx, token, cfg.WikiSpaceID, cfg.DailyNodeToken)
	if err != nil {
		hlog.CtxErrorf(ctx, "[wiki-daily] list month nodes: %v", err)
		return ""
	}
	for i := range nodes {
		if nodes[i].Title == monthTitle {
			month = &nodes[i]
			break
		}
	}
	if month == nil {
		if month, err = wikiCreateNode(ctx, token, cfg.WikiSpaceID, cfg.DailyNodeToken, monthTitle); err != nil {
			hlog.CtxErrorf(ctx, "[wiki-daily] create month node: %v", err)
			return ""
		}
	}

	// 幂等：先建"⏳"临时标题文档，内容全部写完最后一步才改正式标题。
	// 防重复只认正式标题（MM-DD 开头）——写一半被中断的残缺品会一直顶着 ⏳，
	// 下次重跑将其改名废弃并重建，不会把半成品当成品跳过。
	dayPrefix := day.Format("01-02")
	days, err := wikiChildren(ctx, token, cfg.WikiSpaceID, month.NodeToken)
	if err != nil {
		hlog.CtxErrorf(ctx, "[wiki-daily] list day nodes: %v", err)
		return ""
	}
	for _, n := range days {
		if strings.HasPrefix(n.Title, dayPrefix) {
			hlog.CtxInfof(ctx, "[wiki-daily] doc exists for %s, skip", dayPrefix)
			return "https://deeplang.feishu.cn/wiki/" + n.NodeToken
		}
		if strings.HasPrefix(n.Title, "⏳"+dayPrefix) {
			_ = wikiUpdateTitle(ctx, token, cfg.WikiSpaceID, n.NodeToken, "❌残缺-"+dayPrefix)
			hlog.CtxInfof(ctx, "[wiki-daily] stale partial doc renamed for %s", dayPrefix)
		}
	}
	node, err := wikiCreateNode(ctx, token, cfg.WikiSpaceID, month.NodeToken, "⏳"+dayPrefix)
	if err != nil {
		hlog.CtxErrorf(ctx, "[wiki-daily] create day doc: %v", err)
		return ""
	}

	// 逐卡片写入：卡片标题不落文档（"两张卡"是群投递的限制，文档里层就是一级标题）；
	// 首卡副标题（统计周期）作引用置顶。每个元素单独一批（表格含单元格块多，合批易超限）。
	for i, m := range msgs {
		if i == 0 && m.Card.Header.Subtitle != nil {
			head := &blockBatch{}
			head.add(15, "quote", map[string]any{"elements": txtEls(m.Card.Header.Subtitle.Content)}, nil, true)
			if err := appendBatch(ctx, token, node.ObjToken, head); err != nil {
				hlog.CtxErrorf(ctx, "[wiki-daily] append head: %v", err)
				return ""
			}
		}
		for _, el := range m.Card.Body.Elements {
			b := &blockBatch{}
			switch v := el.(type) {
			case fcMd:
				b.mdToBlocks(v.Content)
			case fcTable:
				b.tableToBlocks(v)
			case fcHr:
				b.add(22, "divider", map[string]any{}, nil, true)
			}
			if err := appendBatch(ctx, token, node.ObjToken, b); err != nil {
				hlog.CtxErrorf(ctx, "[wiki-daily] append element: %v", err)
				return ""
			}
		}
	}
	// 内容全部写入成功，改成正式标题（幂等锚点）
	if err := wikiUpdateTitle(ctx, token, cfg.WikiSpaceID, node.NodeToken, docTitle(day, msgs)); err != nil {
		hlog.CtxErrorf(ctx, "[wiki-daily] finalize title: %v", err)
		return ""
	}
	url := "https://deeplang.feishu.cn/wiki/" + node.NodeToken
	hlog.CtxInfof(ctx, "[wiki-daily] archived %s -> %s", day.Format("2006-01-02"), url)
	return url
}

func wikiUpdateTitle(ctx context.Context, token, spaceID, nodeToken, title string) error {
	url := fmt.Sprintf("%s/wiki/v2/spaces/%s/nodes/%s/update_title", feishuOpenBase, spaceID, nodeToken)
	_, err := feishuCall(ctx, "POST", url, token, map[string]any{"title": title})
	return err
}
