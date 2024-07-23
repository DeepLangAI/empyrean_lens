package probe

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/url"
	"path/filepath"
	"sync"
	"time"
)

type Agent struct {
	Headers map[string]string
	Url     string
	UrlId   string
	FileId  string
}

type RestResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type PassportResp struct {
	RestResp
	Data struct {
		Uid         string `json:"uid"`
		Bid         string `json:"b_id"`
		AuthToken   string `json:"auth_token"`
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

type UrlUploadResp struct {
	RestResp
	Data struct {
		UrlId string `json:"url_id"`
	} `json:"data"`
}

var agent = Agent{}
var agentOnce sync.Once

func NewAgent() *Agent {
	agentOnce.Do(func() {
		agent = Agent{}
		agent.Headers = map[string]string{
			"access-token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJCdWZmZXJUaW1lIjo2MDQ4MDAsImlzcyI6ImFjY2Vzc190b2tlbiIsImV4cCI6MTcyMDUwOTkxOCwibmJmIjoxNzIwNTAyNzE4LCJVaWQiOiI0ZmQwNWVkMWJmYWE0YWQwOTU5OWM3ZjQwYWQ3MTg3NiIsIlBpZCI6IjYzZTA3MTM5MzBjMzNhMTY3Zjc5ZDVkOCIsIkdpZCI6NywiUGhvbmUiOiIiLCJVc2VyTmFtZSI6IueUqOaIt18xNzE3NDg5MjM3NTA0In0.CtXjo45KDiN21xQVkl6Qtuh3dmsLIZ1oG9vcSm4qb6Q",
			"anonymous-id": "1903979de9e1823-0035cc72e6c949d-1b525637-1484784-1903979de9f3a8",
			"auth-token":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJCdWZmZXJUaW1lIjo2MDQ4MDAsImlzcyI6ImF1dGhfdG9rZW4iLCJleHAiOjE3MjE3MTIzMTgsIm5iZiI6MTcyMDUwMjcxOCwiVWlkIjoiNGZkMDVlZDFiZmFhNGFkMDk1OTljN2Y0MGFkNzE4NzYiLCJQaWQiOiI2M2UwNzEzOTMwYzMzYTE2N2Y3OWQ1ZDgiLCJHaWQiOjcsIlBob25lIjoiIiwiVXNlck5hbWUiOiLnlKjmiLdfMTcxNzQ4OTIzNzUwNCJ9.jFWMUwY1kWbMq5g98cN2wKWVYdonTgxgog0RYkiLcC4",
			"b-id":         "86fdbdaad86e4d21907e3991a5e7b1b4",
			"u-id":         "63e0713930c33a167f79d5d8",
			"user-agent":   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		}
	})
	return &agent
}

func (self *Agent) Login(ctx context.Context) bool {
	jsonData := struct {
		Account   string `json:"account"`
		Pwd       string `json:"pwd"`
		Source    int    `json:"source"`
		SubSource int    `json:"sub_source"`
	}{}
	jsonData.Account = "KunSJhJOmkFyOKANvYh4jg=="
	jsonData.Pwd = "qvAxiRdPv5ag88o7h2FPAA=="
	jsonData.Source = 7
	jsonData.SubSource = 703

	resp := PassportResp{}
	url := consts.PASSPORT_HOST + "/api/login/account"
	if apiFailed(utils.DoPost(ctx, url, self.Headers, jsonData, &resp), &resp.RestResp) {
		return false
	}
	self.Headers["access-token"] = resp.Data.AccessToken
	self.Headers["auth-token"] = resp.Data.AuthToken
	self.Headers["u-id"] = resp.Data.Uid
	self.Headers["b-id"] = resp.Data.Bid

	return true
}

func (self *Agent) Logout(ctx context.Context) bool {
	url := consts.PASSPORT_HOST + "/api/user/logout"
	resp := RestResp{}
	return !apiFailed(utils.DoPost(ctx, url, self.Headers, nil, &resp), &resp)
}

func (self *Agent) searchFile(ctx context.Context) bool {
	jsonData := struct {
		FileNames []string `json:"filenames"`
	}{}
	resp := RestResp{}
	jsonData.FileNames = append(jsonData.FileNames, filepath.Base(consts.PDF_TO_UPLOAD))
	url := consts.LINGO_HOST + "/api/plugin/file/batch/name/search"
	return !apiFailed(utils.DoPost(ctx, url, self.Headers, nil, &resp), &resp)
}

func (self *Agent) uploadFile(ctx context.Context) bool {
	return false
}

func (self *Agent) UploadPDF(ctx context.Context) bool {
	return self.searchFile(ctx) && self.uploadFile(ctx)
}
func (self *Agent) PdfParse(ctx context.Context) bool {
	return false
}
func (self *Agent) PdfAbstract(ctx context.Context) bool {
	return false
}
func (self *Agent) PdfViewpoint(ctx context.Context) bool {
	return false
}
func (self *Agent) PdfOutline(ctx context.Context) bool {
	return false
}

func (self *Agent) Root(ctx context.Context) bool {
	return true
}

func (self *Agent) uploadURL_check(ctx context.Context) bool {
	jsonData := map[string]string{
		"url": self.Url,
	}
	resp := RestResp{}
	return !apiFailed(utils.DoPost(ctx, consts.LINGO_HOST+"/api/readers/url/check", self.Headers, jsonData, &resp), &resp)
}

func (self *Agent) uploadURL_upload(ctx context.Context) bool {
	jsonData := struct {
		Url         string `json:"url"`
		ChannelType int    `json:"channel_type"`
	}{
		Url:         self.Url,
		ChannelType: 70,
	}
	resp := UrlUploadResp{}
	if apiFailed(utils.DoPost(ctx, consts.LINGO_HOST+"/api/readers/url/upload", self.Headers, jsonData, &resp), &resp.RestResp) {
		return false
	}
	self.UrlId = resp.Data.UrlId
	return true
}

func (self *Agent) uploadURL_waitStatus(ctx context.Context) bool {
	type Data struct {
		Status int `json:"status"`
	}
	type Resp struct {
		RestResp
		Data Data `json:"data"`
	}

	for i := 0; i < 60*5; i++ {
		jsonData := map[string]string{
			"url_id": self.UrlId,
		}
		resp := Resp{}
		resp.Data.Status = -1
		if apiFailed(utils.DoPost(ctx, consts.LINGO_HOST+"/api/readers/url/status", self.Headers, jsonData, &resp), &resp.RestResp) {
			return false
		}
		if resp.Data.Status == 0 {
			return true
		}
		hlog.CtxInfof(ctx, "[client] waiting for url to be uploaded. current: %v", i+1)
		time.Sleep(1 * time.Second)
	}
	return false
}

func (self *Agent) uploadURL_getDetail(ctx context.Context) bool {
	params := url.Values{
		"entry_id":   {self.UrlId},
		"entry_type": {"7"},
	}

	resp := RestResp{}
	return !apiFailed(utils.DoGet(ctx, consts.LINGO_HOST+"/api/entry/detail", params, self.Headers, &resp), &resp)
}

func (self *Agent) UploadURL(ctx context.Context) bool {
	url := fmt.Sprintf("http://www.news.cn/20240712/a0f88cded3bb48d29772e8b7bb797695/c.html?sign=%s", primitive.NewObjectID().Hex())
	self.Url = url
	return self.uploadURL_check(ctx) &&
		self.uploadURL_upload(ctx)
}

func (self *Agent) UrlDldParse(ctx context.Context) bool {
	return self.uploadURL_waitStatus(ctx) &&
		self.uploadURL_getDetail(ctx)

}

func (self *Agent) UrlAbstract(ctx context.Context) bool {
	jsonData := struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		Title         string `json:"title"`
		Author        string `json:"author"`
		OutlineType   int    `json:"outline_type"`
		Content       string `json:"content"`
		ContentLength int    `json:"contentLength"`
		URLId         string `json:"url_id"`
		EntryType     int    `json:"entry_type"`
		GenerateType  int    `json:"generate_type"`
		PairID        string `json:"pair_id"`
		ChannelType   int    `json:"channel_type"`
		ModeStage     int    `json:"mode_stage"`
	}{
		ID:            self.UrlId,
		URL:           "",
		Title:         "",
		Author:        "",
		OutlineType:   1,
		Content:       "",
		ContentLength: 2284,
		URLId:         self.UrlId,
		EntryType:     5,
		GenerateType:  0,
		PairID:        "Cqz98jiT84t9KAQNBLiuu",
		ChannelType:   20,
		ModeStage:     3,
	}
	resp := RestResp{}
	return !apiFailed(utils.DoPost(ctx, consts.LINGO_HOST+"/api/plugin/articles/summary", self.Headers, jsonData, &resp), &resp)
}

func (self *Agent) UrlViewpoint(ctx context.Context) bool {

	jsonData := struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		Title         string `json:"title"`
		Author        string `json:"author"`
		OutlineType   int    `json:"outline_type"`
		Content       string `json:"content"`
		ContentLength int    `json:"contentLength"`
		URLId         string `json:"url_id"`
		EntryType     int    `json:"entry_type"`
		GenerateType  int    `json:"generate_type"`
		PairID        string `json:"pair_id"`
		ChannelType   int    `json:"channel_type"`
		ModeStage     int    `json:"mode_stage"`
		Version       string `json:"version"`
	}{
		ID:            self.UrlId,
		URL:           "",
		Title:         "",
		Author:        "",
		OutlineType:   1,
		Content:       "",
		ContentLength: 2204,
		URLId:         self.UrlId,
		EntryType:     11,
		GenerateType:  3,
		PairID:        "R_XBMTpkr9hT0hVijyhW6",
		ChannelType:   20,
		ModeStage:     3,
		Version:       "v1.0.6",
	}
	lines, err := utils.DoStreamPost(ctx, consts.LINGO_HOST+"/api/plugin/articles/summary", self.Headers, jsonData)
	if err != nil {
		return false
	}
	return len(lines) >= 3
}

func (self *Agent) urlOutline(ctx context.Context, outlineType int) bool {
	jsonData := struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		Title         string `json:"title"`
		Author        string `json:"author"`
		OutlineType   int    `json:"outline_type"`
		Content       string `json:"content"`
		ContentLength int    `json:"contentLength"`
		UrlId         string `json:"url_id"`
		EntryType     int    `json:"entry_type"`
		GenerateType  int    `json:"generate_type"`
		PairID        string `json:"pair_id"`
		ChannelType   int    `json:"channel_type"`
		ModeStage     int    `json:"mode_stage"`
		Version       string `json:"version"`
	}{
		ID:            self.UrlId,
		URL:           "",
		Title:         "",
		Author:        "",
		OutlineType:   outlineType,
		Content:       "",
		ContentLength: 2204,
		UrlId:         self.UrlId,
		EntryType:     6,
		GenerateType:  1,
		PairID:        "1DzK5ijyi6JjEbOWy7SJH",
		ChannelType:   20,
		ModeStage:     3,
		Version:       "v1.0.6",
	}
	lines, err := utils.DoStreamPost(ctx, consts.LINGO_HOST+"/api/plugin/articles/summary", self.Headers, jsonData)
	if err != nil {
		return false
	}
	return len(lines) >= 3
}

func (self *Agent) UrlOutline(ctx context.Context) bool {
	OUTLINE_SIMPLE := 1
	OUTLINE_COMPLEX := 2
	return self.urlOutline(ctx, OUTLINE_SIMPLE) && self.urlOutline(ctx, OUTLINE_COMPLEX)
}
