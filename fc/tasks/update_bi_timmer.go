package tasks

import (
	"context"
	"time"

	"empyrean_lens/fc/conf"
	"empyrean_lens/fc/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type BatchUpdateLinkTraceEventEvent struct {
	BeginAt int64 `json:"begin_at"`
	EndAt   int64 `json:"end_at"`
}

func HandleBatchUpdateLinkTraceEvent(ctx context.Context) error {
	hlog.CtxInfof(ctx, "batch update link trace begin")

	startDay := startDay(time.Now())
	endAt := startDay.Unix()
	beginAt := endAt - 24*60*60
	event := &BatchUpdateLinkTraceEventEvent{
		BeginAt: beginAt,
		EndAt:   endAt,
	}

	err := doBatchUpdate(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch update link trace error:%v, event:%v", err, event)
		return err
	}

	hlog.CtxInfof(ctx, "batch update link trace success, event:%v", event)
	return nil
}

func startDay(t time.Time) time.Time {
	return t.Add(8 * time.Hour).Truncate(24 * time.Hour).Add(-8 * time.Hour)
}

func doBatchUpdate(ctx context.Context, event *BatchUpdateLinkTraceEventEvent) error {
	req := map[string]interface{}{
		"begin_at": event.BeginAt,
		"end_at":   event.EndAt,
	}
	hlog.CtxInfof(ctx, "batch update link trace begin, req:%v", req)
	_, err := utils.HttpPost(ctx, conf.GetConfig().Api.BatchUpdateLinkTrace, map[string]string{"Channel": "local"}, req)
	if err != nil {
		hlog.CtxErrorf(ctx, "do batch update error:%v", err)
	}
	return err
}
