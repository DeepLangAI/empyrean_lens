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

// 示例数据作为注释，勿删
var _ = `
告警：https://sls.console.aliyun.com/lognext/project/k8s-log-cefd7f8df3eab44a4a8184343c914af94/alert/alert-1721877944-703277
回调请求体：
[{
	"alert_id": "alert-1721877944-703277",
	"alert_instance_id": "fa6c5b74540cf85f-655ac8c7e1013-2257966",
	"alert_name": "生成服务日志告警",
	"alert_time": 1783049440,
	"aliuid": "1092547886793620",
	"annotations": {
		"__count__": "1",
		"desc": "生成服务日志告警告警触发",
		"error_ops": "920",
		"service_type": "single_abstract",
		"success_pct": "81.42",
		"title": "生成服务日志告警告警触发",
		"total_calls": "4951"
	},
	"fingerprint": "341e6a6a57b54572",
	"fire_results": [{
		"error_ops": "920",
		"service_type": "single_abstract",
		"success_pct": "81.42",
		"total_calls": "4951"
	}],
	"fire_results_count": 1,
	"fire_time": 1783049440,
	"labels": {
		"x-report-to-feishu-webhook": "https://open.feishu.cn/open-apis/bot/v2/hook/00000000-0000-0000-0000-000000000000"
	},
	"project": "k8s-log-cefd7f8df3eab44a4a8184343c914af94",
	"region": "cn-zhangjiakou",
	"resolve_time": 0,
	"results": [{
		"dashboard_url": "",
		"end_time": 1783049400,
		"fire_result": {
			"error_ops": "920",
			"service_type": "single_abstract",
			"success_pct": "81.42",
			"total_calls": "4951"
		},
		"project": "k8s-log-cefd7f8df3eab44a4a8184343c914af94",
		"query": "__tag__:_container_name_: lingowhale-repeater-go-prod and __tag__:_namespace_: repeater and (OutRequest or level: error)\n| select\n    service_type,\n    sum(total)  as total_calls,\n    sum(errors) as error_ops,\n    round(100.0 * (1.0 - cast(sum(errors) as double) / nullif(cast(sum(total) as double), 0)), 2) as success_pct\nfrom (\n    -- 分母：模型调用总量\n    select\n        regexp_extract(message, 'OutRequest (\\S+)', 1) as service_type,\n        1 as total,\n        0 as errors\n    from log\n    where message like 'OutRequest %'\n      and regexp_extract(message, 'OutRequest (\\S+)', 1)\n          not in ('upload_oss', 'add_voice', 'model_daily_voice')\n\n    union all\n\n    -- 分子：真实失败数（operation_id 去重，排除用户输入类噪音）\n    select service_type, 0 as total, 1 as errors\n    from (\n        select\n            operation_id,\n            case\n                when message like 'StreamErrResp%'\n                 and message not like '%code:21001%'          then 'single_abstract'\n                when message like '解析模型返回数据失败%'\n                  or message like '调用下游服务失败%'          then regexp_extract(message, 'model:(\\w+)', 1)\n                when message like 'StreamErr %'               then regexp_extract(message, 'StreamErr (\\w+)', 1)\n                when message like '多文档大纲模型生成失败%'    then 'multi_summary'\n                when message like 'EditAudio error%'\n                  or message like 'HSText2Voice%'             then 'hs_tts'\n            end as service_type\n        from log\n        where level = 'error'\n          and message not like '%origContent is empty%'\n          and message not like '%内容过少，不支持生成%'\n          and message not like '%AsyncExec error success%'\n          and message not like 'FindOneByParseIDAndDataType%'\n          and message not like 'SingAnalyze Error%'\n          and message not like 'ErrorResponse%'\n          and message not like 'server panic%'\n        group by operation_id, service_type\n        having service_type is not null\n    )\n) t\ngroup by service_type\nhaving service_type is not null and service_type != ''\nand success_pct \u003c 90\nand total_calls \u003e= 30\norder by total_calls desc",
		"query_url": "https://sls.console.aliyun.com/lognext/project/k8s-log-cefd7f8df3eab44a4a8184343c914af94/logsearch/business-pod?encode=base64\u0026endTime=1783049400\u0026queryString=X190YWdfXzpfY29udGFpbmVyX25hbWVfOiBsaW5nb3doYWxlLXJlcGVhdGVyLWdvLXByb2QgYW5kIF9fdGFnX186X25hbWVzcGFjZV86IHJlcGVhdGVyIGFuZCAoT3V0UmVxdWVzdCBvciBsZXZlbDogZXJyb3IpCnwgc2VsZWN0CiAgICBzZXJ2aWNlX3R5cGUsCiAgICBzdW0odG90YWwpICBhcyB0b3RhbF9jYWxscywKICAgIHN1bShlcnJvcnMpIGFzIGVycm9yX29wcywKICAgIHJvdW5kKDEwMC4wICogKDEuMCAtIGNhc3Qoc3VtKGVycm9ycykgYXMgZG91YmxlKSAvIG51bGxpZihjYXN0KHN1bSh0b3RhbCkgYXMgZG91YmxlKSwgMCkpLCAyKSBhcyBzdWNjZXNzX3BjdApmcm9tICgKICAgIC0tIOWIhuavje%2B8muaooeWei%2Biwg%2BeUqOaAu%2BmHjwogICAgc2VsZWN0CiAgICAgICAgcmVnZXhwX2V4dHJhY3QobWVzc2FnZSwgJ091dFJlcXVlc3QgKFxTKyknLCAxKSBhcyBzZXJ2aWNlX3R5cGUsCiAgICAgICAgMSBhcyB0b3RhbCwKICAgICAgICAwIGFzIGVycm9ycwogICAgZnJvbSBsb2cKICAgIHdoZXJlIG1lc3NhZ2UgbGlrZSAnT3V0UmVxdWVzdCAlJwogICAgICBhbmQgcmVnZXhwX2V4dHJhY3QobWVzc2FnZSwgJ091dFJlcXVlc3QgKFxTKyknLCAxKQogICAgICAgICAgbm90IGluICgndXBsb2FkX29zcycsICdhZGRfdm9pY2UnLCAnbW9kZWxfZGFpbHlfdm9pY2UnKQoKICAgIHVuaW9uIGFsbAoKICAgIC0tIOWIhuWtkO%2B8muecn%2BWunuWksei0peaVsO%2B8iG9wZXJhdGlvbl9pZCDljrvph43vvIzmjpLpmaTnlKjmiLfovpPlhaXnsbvlmarpn7PvvIkKICAgIHNlbGVjdCBzZXJ2aWNlX3R5cGUsIDAgYXMgdG90YWwsIDEgYXMgZXJyb3JzCiAgICBmcm9tICgKICAgICAgICBzZWxlY3QKICAgICAgICAgICAgb3BlcmF0aW9uX2lkLAogICAgICAgICAgICBjYXNlCiAgICAgICAgICAgICAgICB3aGVuIG1lc3NhZ2UgbGlrZSAnU3RyZWFtRXJyUmVzcCUnCiAgICAgICAgICAgICAgICAgYW5kIG1lc3NhZ2Ugbm90IGxpa2UgJyVjb2RlOjIxMDAxJScgICAgICAgICAgdGhlbiAnc2luZ2xlX2Fic3RyYWN0JwogICAgICAgICAgICAgICAgd2hlbiBtZXNzYWdlIGxpa2UgJ%2Bino%2BaekOaooeWei%2Bi%2FlOWbnuaVsOaNruWksei0pSUnCiAgICAgICAgICAgICAgICAgIG9yIG1lc3NhZ2UgbGlrZSAn6LCD55So5LiL5ri45pyN5Yqh5aSx6LSlJScgICAgICAgICAgdGhlbiByZWdleHBfZXh0cmFjdChtZXNzYWdlLCAnbW9kZWw6KFx3KyknLCAxKQogICAgICAgICAgICAgICAgd2hlbiBtZXNzYWdlIGxpa2UgJ1N0cmVhbUVyciAlJyAgICAgICAgICAgICAgIHRoZW4gcmVnZXhwX2V4dHJhY3QobWVzc2FnZSwgJ1N0cmVhbUVyciAoXHcrKScsIDEpCiAgICAgICAgICAgICAgICB3aGVuIG1lc3NhZ2UgbGlrZSAn5aSa5paH5qGj5aSn57qy5qih5Z6L55Sf5oiQ5aSx6LSlJScgICAgdGhlbiAnbXVsdGlfc3VtbWFyeScKICAgICAgICAgICAgICAgIHdoZW4gbWVzc2FnZSBsaWtlICdFZGl0QXVkaW8gZXJyb3IlJwogICAgICAgICAgICAgICAgICBvciBtZXNzYWdlIGxpa2UgJ0hTVGV4dDJWb2ljZSUnICAgICAgICAgICAgIHRoZW4gJ2hzX3R0cycKICAgICAgICAgICAgZW5kIGFzIHNlcnZpY2VfdHlwZQogICAgICAgIGZyb20gbG9nCiAgICAgICAgd2hlcmUgbGV2ZWwgPSAnZXJyb3InCiAgICAgICAgICBhbmQgbWVzc2FnZSBub3QgbGlrZSAnJW9yaWdDb250ZW50IGlzIGVtcHR5JScKICAgICAgICAgIGFuZCBtZXNzYWdlIG5vdCBsaWtlICcl5YaF5a656L%2BH5bCR77yM5LiN5pSv5oyB55Sf5oiQJScKICAgICAgICAgIGFuZCBtZXNzYWdlIG5vdCBsaWtlICclQXN5bmNFeGVjIGVycm9yIHN1Y2Nlc3MlJwogICAgICAgICAgYW5kIG1lc3NhZ2Ugbm90IGxpa2UgJ0ZpbmRPbmVCeVBhcnNlSURBbmREYXRhVHlwZSUnCiAgICAgICAgICBhbmQgbWVzc2FnZSBub3QgbGlrZSAnU2luZ0FuYWx5emUgRXJyb3IlJwogICAgICAgICAgYW5kIG1lc3NhZ2Ugbm90IGxpa2UgJ0Vycm9yUmVzcG9uc2UlJwogICAgICAgICAgYW5kIG1lc3NhZ2Ugbm90IGxpa2UgJ3NlcnZlciBwYW5pYyUnCiAgICAgICAgZ3JvdXAgYnkgb3BlcmF0aW9uX2lkLCBzZXJ2aWNlX3R5cGUKICAgICAgICBoYXZpbmcgc2VydmljZV90eXBlIGlzIG5vdCBudWxsCiAgICApCikgdApncm91cCBieSBzZXJ2aWNlX3R5cGUKaGF2aW5nIHNlcnZpY2VfdHlwZSBpcyBub3QgbnVsbCBhbmQgc2VydmljZV90eXBlICE9ICcnCmFuZCBzdWNjZXNzX3BjdCA8IDkwCmFuZCB0b3RhbF9jYWxscyA%2BPSAzMApvcmRlciBieSB0b3RhbF9jYWxscyBkZXNj\u0026queryTimeType=99\u0026startTime=1783048500",
		"raw_results": [{
			"error_ops": "920",
			"service_type": "single_abstract",
			"success_pct": "81.42",
			"total_calls": "4951"
		}],
		"raw_results_count": 1,
		"region": "cn-zhangjiakou",
		"role_arn": "",
		"start_time": 1783048500,
		"store": "business-pod",
		"store_type": "log"
	}],
	"severity": 10,
	"status": "firing"
}]
`

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
