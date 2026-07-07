namespace go empyrean_lens.notice_webhook

struct BaseResp {
    1: i32 code
    2: string msg
}

struct SlsResult {
    1: string dashboard_url
    2: i64 end_time
    3: map<string, string> fire_result
    4: string project
    5: string query
    6: string query_url
    7: list<map<string, string>> raw_results
    8: i64 raw_results_count
    9: string region
    10: string role_arn
    11: i64 start_time
    12: string store
    13: string store_type
}

struct SlsAlert {
    1: string alert_id
    2: string alert_instance_id
    3: string alert_name
    4: i64 alert_time
    5: string aliuid
    6: map<string, string> annotations
    7: string fingerprint
    8: list<map<string, string>> fire_results
    9: i64 fire_results_count
    10: i64 fire_time
    11: map<string, string> labels
    12: string project
    13: string region
    14: i64 resolve_time
    15: list<SlsResult> results
    16: i64 severity
    17: string status
}

struct SlsNoticeWebhookReq {
    1: string raw_body (api.raw_body="")
}


service NoticeWebhookService {
    BaseResp SlsNoticeWebhook(1: SlsNoticeWebhookReq req)(
        api.post="/api/v1/notice_webhook/sls_notice"
    )
}
