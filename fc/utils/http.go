package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type CommonResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func HttpPost(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error) {
	reqBody, _ := sonic.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(reqBody)))
	if err != nil {
		hlog.CtxErrorf(ctx, "post new req, url:%s error:%v", url, err)
		return nil, err
	}

	if env, ok := ctx.Value("env").(string); ok && env != "" {
		req.Header.Set("env", env)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	hlog.CtxInfof(ctx, "header", req.Header)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		hlog.CtxErrorf(ctx, "error, url:%s, err:%v", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		hlog.CtxErrorf(ctx, "resp status code, url:%s, code:%v", url, resp.StatusCode)
		return nil, fmt.Errorf("resp status code: %v", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		hlog.CtxErrorf(ctx, "read resp error, url:%s, err:%v", url, err)
		return nil, err
	}

	commonResp := &CommonResponse{}
	_ = sonic.Unmarshal(respBody, commonResp)
	if commonResp.Code != 0 {
		return nil, errors.New(commonResp.Msg)
	}

	return respBody, nil
}
