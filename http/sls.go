package http

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"time"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type GetLogsResp struct {
	Code  int                 `json:"code"`
	Msg   string              `json:"msg"`
	Logs  []map[string]string `json:"logs"`
	Total int                 `json:"total"`
}

type LogStore string

const (
	LogStore_BusinessPod LogStore = "business-pod"
	LogStore_NginxIngress LogStore = "nginx-ingress"
)

func SlsQuery(ctx context.Context, logStore LogStore, query string, startTime, endTime time.Time, limit int) (*GetLogsResp, error) {
	body := map[string]any{
		"log_store": logStore,
		"query":     query,
		"time_from": startTime.Local().Format(time.DateTime),
		"time_to":   endTime.Local().Format(time.DateTime),
		"limit":     limit,
	}
	bodyBytes, _ := sonic.Marshal(body)
	receiver := &GetLogsResp{}
	slsSecret := conf.GetConfig().ExternalSecret.DeeplangSlsFcSecret
	_, err := httplib.Do(ctx, fmt.Sprintf("%s/get_logs", slsSecret.BaseUrl), map[string]string{
		consts.HeaderContentType:   consts.MIMEApplicationJSON,
		consts.HeaderAuthorization: "Bearer " + slsSecret.Token,
	}, bodyBytes, receiver)
	if err != nil {
		return nil, err
	}
	if receiver.Code != 0 {
		return nil, fmt.Errorf("sls query error, code: %v, msg: %s", receiver.Code, receiver.Msg)
	}
	return receiver, nil
}
