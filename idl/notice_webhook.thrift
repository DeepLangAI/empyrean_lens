namespace go empyrean_lens.notice_webhook

struct BaseResp {
    1: i32 code
    2: string msg
}

struct SlsNoticeWebhookReq {
    1: required string report_to_feishu_webhook (api.header="x-report-to-feishu-webhook")
}


service NoticeWebhookService {
    BaseResp SlsNoticeWebhook(1: SlsNoticeWebhookReq req)(
        api.post="/api/v1/notice_webhook/sls_notice"
    )
}