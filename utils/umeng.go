package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"io"
	"net/http"
)

func UmengDoPost(ctx context.Context, url string, data map[string]interface{}, cookie string) (string, error) {
	// 建立链接
	client := &http.Client{}
	reqData, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("cookie", cookie)
	// 发起请求
	rep, err := client.Do(req)
	if err != nil {
		hlog.CtxErrorf(ctx, "post fail, data:%v, err:%v", reqData, err)
		return "", err
	}
	if rep.StatusCode != 200 {
		hlog.Errorf("must check fail, data:%v, code:%d", data, rep.StatusCode)
		return "", errors.New("must check fail")
	}
	// 返回结果
	body, _ := io.ReadAll(rep.Body)
	return string(body), nil
}
