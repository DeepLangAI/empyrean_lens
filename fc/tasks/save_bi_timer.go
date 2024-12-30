package tasks

import (
	"context"
	"time"

	"empyrean_lens/fc/conf"
	"empyrean_lens/fc/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type BatchSaveLinkTraceEventEvent struct {
	EntryType int   `json:"entry_type"`
	BeginAt   int64 `json:"begin_at"`
	EndAt     int64 `json:"end_at"`
}

func HandleBatchSaveLinkTraceEvent(ctx context.Context) error {
	hlog.CtxInfof(ctx, "batch save link trace begin")

	beginAt := time.Now().Add(-15 * time.Minute).Unix()
	endAt := beginAt + 5*time.Minute.Milliseconds()/1000
	event := &BatchSaveLinkTraceEventEvent{
		BeginAt: beginAt,
		EndAt:   endAt,
	}

	// 单文档，web
	event.EntryType = 7
	err := doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 单文档，pdf
	event.EntryType = 10
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 多文档
	event.EntryType = 12
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 概述重新生成
	event.EntryType = 5
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 大纲重新生成
	event.EntryType = 6
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 关键观点重新生成
	event.EntryType = 11
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 多文档总结重新生成
	event.EntryType = 126
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 订阅网页
	event.EntryType = 71
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 订阅pdf
	event.EntryType = 101
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	// 订阅多文档
	event.EntryType = 121
	err = doBatchSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "batch save link trace error:%v, event:%v", err, event)
		return err
	}

	hlog.CtxInfof(ctx, "batch save link trace success, event:%v", event)
	return nil
}

func doBatchSave(ctx context.Context, event *BatchSaveLinkTraceEventEvent) error {
	req := map[string]interface{}{
		"entry_type": event.EntryType,
		"begin_at":   event.BeginAt,
		"end_at":     event.EndAt,
	}
	hlog.CtxInfof(ctx, "batch save link trace begin, req:%v", req)
	_, err := utils.HttpPost(ctx, conf.GetConfig().Api.BatchSaveLinkTrace, map[string]string{"Channel": "local"}, req)
	if err != nil {
		hlog.CtxErrorf(ctx, "do batch save error:%v", err)
	}
	return err
}
