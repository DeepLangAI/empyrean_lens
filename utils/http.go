package utils

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func DoPost(
	ctx context.Context,
	url string,
	headers map[string]string,
	data any,
	res any,
) error {
	start := time.Now()
	defer func() {
		hlog.CtxInfof(ctx, "out request, method:%s, cost:%v", url, TimeSub(start))
	}()
	// 建立链接
	client := &http.Client{}
	reqData := ""
	if data != nil {
		reqData = JSONMarshal(data)
	}
	hlog.CtxInfof(ctx, "out request, method:%s, req:%s, ", url, reqData)
	req, _ := http.NewRequest("POST", url, strings.NewReader(reqData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Trace-Id", utillib.GetCtxTraceId(ctx))
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Post] post fail, err:%v, data:%v", err, reqData)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		hlog.CtxErrorf(ctx, "[Http Post] status code not 200, url:%s  code:%d, data:%v", url, resp.StatusCode, JSONMarshal(reqData))
		return errors.New("status code not 200")
	}
	// 解析校验结果
	body, _ := io.ReadAll(resp.Body)
	err = sonic.Unmarshal(body, &res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Post] json unmarshal fail, err:%v", err)
		return err
	}
	return nil
}

func DoStreamPost(
	ctx context.Context,
	url string,
	headers map[string]string,
	data any,
) ([]string, error) {
	start := time.Now()
	defer func() {
		hlog.CtxInfof(ctx, "out request, method:%s, cost:%v", url, TimeSub(start))
	}()
	// 建立链接
	client := &http.Client{}
	reqData := ""
	if data != nil {
		reqData = JSONMarshal(data)
	}
	hlog.CtxInfof(ctx, "out request, method:%s, req:%s, ", url, reqData)
	req, _ := http.NewRequest("POST", url, strings.NewReader(reqData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Trace-Id", utillib.GetCtxTraceId(ctx))
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Post] post fail, err:%v, data:%v", err, reqData)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		hlog.CtxErrorf(ctx, "[Http Post] status code not 200, url:%s  code:%d, data:%v", url, resp.StatusCode, JSONMarshal(reqData))
		return nil, errors.New("status code not 200")
	}

	reader := bufio.NewReader(resp.Body)
	lines := []string{}
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			hlog.CtxErrorf(ctx, "[Http Post] read stream fail, err:%v", err)
			return nil, err
		}
		if len(line) == 0 {
			continue
		}
		lines = append(lines, line)
		hlog.CtxDebugf(ctx, "out request, method:%s, line:%s", url, line)
	}

	//// 解析校验结果
	//body, _ := io.ReadAll(resp.Body)
	//err = sonic.Unmarshal(body, &res)
	//if err != nil {
	//	hlog.CtxErrorf(ctx, "[Http Post] json unmarshal fail, err:%v", err)
	//	return nil, err
	//}
	return lines, nil
}

func DoGet(
	ctx context.Context,
	baseUrl string,
	params url.Values,
	headers map[string]string,
	res any,
) error {
	return DoGetWithAuth(ctx, baseUrl, params, headers, res, "", "")
}

func DoGetWithAuth(
	ctx context.Context,
	baseUrl string,
	params url.Values,
	headers map[string]string,
	res any,
	authKey, authValue string,
) error {
	start := time.Now()
	uri := baseUrl
	if params != nil {
		uri = fmt.Sprintf("%s?%s", baseUrl, params.Encode())
	}
	defer func() {
		hlog.CtxInfof(ctx, "out request, method:%s, cost:%v", baseUrl, TimeSub(start))
	}()
	// 构建请求对象
	client := &http.Client{}
	client.Timeout = time.Second * 10
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Get] new request fail, err:%v, params:%v", err, params)
		return err
	}
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	if authKey != "" && authValue != "" {
		req.SetBasicAuth(authKey, authValue)
	}

	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Get] get fail, err:%v, params:%v", err, params)
		return err
	}
	defer resp.Body.Close()
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		hlog.CtxErrorf(ctx, "[Http Get] status code not 200,  code:%d, params:%v", resp.StatusCode, params)
		return errors.New("status code not 200")
	}
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Get] read body fail, err:%v, params:%v", err, params)
		return err
	}
	// 解析响应
	err = sonic.Unmarshal(body, &res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Get] json unmarshal fail, err:%v, params:%v", err, params)
		return err
	}
	return nil
}

func IsInnerIp(ip string) bool {
	if ip == "" {
		return false
	}
	if Contains([]string{"127.0.0.1", "::1"}, ip) {
		return true
	}
	return false
}
