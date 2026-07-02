package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	notice_webhook "empyrean_lens/biz/model/empyrean_lens/notice_webhook"
	"empyrean_lens/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAlertName = "生成服务日志告警"

func mockWebhookServer(t *testing.T) (*httptest.Server, *[]byte, *int32) {
	t.Helper()
	var body []byte
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &body, &count
}

func callService(t *testing.T, rawBody string) {
	t.Helper()
	svc := service.NewSlsNoticeWebhookService()
	resp, bizErr := svc.SlsNoticeWebhook(context.Background(), &notice_webhook.SlsNoticeWebhookReq{
		RawBody: rawBody,
	})
	require.Nil(t, bizErr)
	require.NotNil(t, resp)
	assert.Equal(t, int32(0), resp.Code)
}

// TestSlsNoticeWebhook_FireResults 验证: firing 告警发送红色卡片，表格包含 fire_results 数据
func TestSlsNoticeWebhook_FireResults(t *testing.T) {
	srv, body, count := mockWebhookServer(t)

	rawBody := fmt.Sprintf(`[{
		"alert_id": "alert-test-001",
		"alert_name": %q,
		"alert_time": 1782975476,
		"fire_time": 1782969584,
		"project": "k8s-log-test",
		"region": "cn-zhangjiakou",
		"severity": 10,
		"status": "firing",
		"fire_results_count": 2,
		"labels": {"x-report-to-feishu-webhook": %q},
		"annotations": {},
		"fire_results": [
			{"service_type": "single_abstract", "total_calls": "97", "error_ops": "0", "success_pct": "100.0"},
			{"service_type": "single_outline",  "total_calls": "4",  "error_ops": "0", "success_pct": "100.0"}
		],
		"results": []
	}]`, testAlertName, srv.URL)

	callService(t, rawBody)

	assert.EqualValues(t, 1, atomic.LoadInt32(count), "feishu webhook should be called exactly once")
	require.NotEmpty(t, *body)

	var msg map[string]interface{}
	require.NoError(t, json.Unmarshal(*body, &msg))

	assert.Equal(t, "interactive", msg["msg_type"])

	card := msg["card"].(map[string]interface{})
	assert.Equal(t, "2.0", card["schema"])

	header := card["header"].(map[string]interface{})
	assert.Equal(t, testAlertName, header["title"].(map[string]interface{})["content"])
	assert.Equal(t, "red", header["template"], "firing should use red header")

	elements := card["body"].(map[string]interface{})["elements"].([]interface{})
	require.GreaterOrEqual(t, len(elements), 2)

	tableElem := elements[1].(map[string]interface{})
	assert.Equal(t, "table", tableElem["tag"])

	rows := tableElem["rows"].([]interface{})
	assert.Len(t, rows, 2)

	// columns in original JSON key order: service_type, total_calls, error_ops, success_pct
	cols := tableElem["columns"].([]interface{})
	firstCol := cols[0].(map[string]interface{})
	assert.Equal(t, "service_type", firstCol["name"])
}

// TestSlsNoticeWebhook_Resolved 验证: resolved 告警发送绿色卡片
func TestSlsNoticeWebhook_Resolved(t *testing.T) {
	srv, body, _ := mockWebhookServer(t)

	rawBody := fmt.Sprintf(`[{
		"alert_id": "alert-resolved",
		"alert_name": "服务恢复通知",
		"status": "resolved",
		"labels": {"x-report-to-feishu-webhook": %q},
		"annotations": {},
		"fire_results": []
	}]`, srv.URL)

	callService(t, rawBody)

	require.NotEmpty(t, *body)
	var msg map[string]interface{}
	require.NoError(t, json.Unmarshal(*body, &msg))

	header := msg["card"].(map[string]interface{})["header"].(map[string]interface{})
	assert.Equal(t, "green", header["template"], "resolved should use green header")
}

// TestSlsNoticeWebhook_NoWebhookURL 验证: 没有 webhook URL 的告警被跳过，服务正常返回
func TestSlsNoticeWebhook_NoWebhookURL(t *testing.T) {
	rawBody := `[{
		"alert_id": "alert-no-webhook",
		"alert_name": "Test Alert",
		"status": "firing",
		"labels": {},
		"annotations": {},
		"fire_results": []
	}]`

	callService(t, rawBody)
}

// TestSlsNoticeWebhook_WebhookURLFromAnnotations 验证: URL 从 annotations 回退读取
func TestSlsNoticeWebhook_WebhookURLFromAnnotations(t *testing.T) {
	srv, _, count := mockWebhookServer(t)

	rawBody := fmt.Sprintf(`[{
		"alert_id": "alert-anno-fallback",
		"alert_name": "Annotation Fallback",
		"status": "firing",
		"labels": {},
		"annotations": {"x-report-to-feishu-webhook": %q},
		"fire_results": []
	}]`, srv.URL)

	callService(t, rawBody)
	assert.EqualValues(t, 1, atomic.LoadInt32(count), "should call webhook via annotation URL")
}

// TestSlsNoticeWebhook_InvalidBody 验证: 非法 JSON 不崩溃，正常返回 success
func TestSlsNoticeWebhook_InvalidBody(t *testing.T) {
	callService(t, `not valid json`)
}
