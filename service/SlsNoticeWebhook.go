package service

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens/notice_webhook"
	"empyrean_lens/consts"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/middleware"
)

type SlsNoticeWebhookService struct {
}

func NewSlsNoticeWebhookService() *SlsNoticeWebhookService {
	return &SlsNoticeWebhookService{}
}

func (s *SlsNoticeWebhookService) SlsNoticeWebhook(ctx context.Context, req *notice_webhook.SlsNoticeWebhookReq) (resp *middleware.BaseResp, bizErr *consts.BizCode) {
	return &middleware.BaseResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
