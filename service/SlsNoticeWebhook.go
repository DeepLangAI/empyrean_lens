package service

import (
	"context"
	notice_webhook "empyrean_lens/biz/model/empyrean_lens/notice_webhook"
	"empyrean_lens/consts"
	"fmt"
	"time"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/middleware"
	retry "github.com/avast/retry-go"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/tidwall/gjson"
)

type SlsNoticeWebhookService struct{}

func NewSlsNoticeWebhookService() *SlsNoticeWebhookService {
	return &SlsNoticeWebhookService{}
}

func (s *SlsNoticeWebhookService) SlsNoticeWebhook(ctx context.Context, req *notice_webhook.SlsNoticeWebhookReq) (resp *middleware.BaseResp, bizErr *consts.BizCode) {
	var alerts []*notice_webhook.SlsAlert
	if err := sonic.UnmarshalString(req.RawBody, &alerts); err != nil {
		hlog.CtxErrorf(ctx, "parse sls alerts failed: %v", err)
		return &middleware.BaseResp{Code: 0, Msg: "success"}, nil
	}

	for i, alert := range alerts {
		webhookURL := alert.Labels["x-report-to-feishu-webhook"]
		if webhookURL == "" {
			webhookURL = alert.Annotations["x-report-to-feishu-webhook"]
		}
		if webhookURL == "" {
			hlog.CtxWarnf(ctx, "sls alert %s has no feishu webhook URL, skip", alert.AlertID)
			continue
		}

		colNames := fireResultColNames(req.RawBody, i)
		card := buildSlsAlertCard(alert, colNames)
		body, err := sonic.Marshal(card)
		if err != nil {
			hlog.CtxErrorf(ctx, "marshal feishu card failed for alert %s: %v", alert.AlertID, err)
			continue
		}

		err = retry.Do(
			func() error {
				receiver := &middleware.BaseResp{}
				_, err := httplib.Do(ctx, webhookURL,
					map[string]string{"Content-Type": "application/json"},
					body,
					receiver,
				)
				if err != nil {
					hlog.CtxErrorf(ctx, "send feishu webhook failed for alert %s, err: %v", alert.AlertID, err)
					return err
				}
				if receiver.Code != 0 {
					err = fmt.Errorf("send feishu webhook failed for alert %s, code: %v, msg: %v", alert.AlertID, receiver.Code, receiver.Msg)
					hlog.CtxErrorf(ctx, err.Error())
					return err
				}
				return nil
			},
			retry.Attempts(3),
			retry.Delay(500*time.Millisecond),
			retry.DelayType(retry.BackOffDelay),
			retry.LastErrorOnly(true),
		)
	}

	return &middleware.BaseResp{Code: 0, Msg: "success"}, nil
}

// fireResultColNames 从原始 JSON 中按照原始顺序提取第 alertIdx 个告警的 fire_results 列名。
func fireResultColNames(rawBody string, alertIdx int) []string {
	first := gjson.Get(rawBody, fmt.Sprintf("%d.fire_results.0", alertIdx))
	var names []string
	first.ForEach(func(key, _ gjson.Result) bool {
		names = append(names, key.String())
		return true
	})
	return names
}
