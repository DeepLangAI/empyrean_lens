package tasks

import (
	"context"

	"empyrean_lens/fc/conf"
	"empyrean_lens/fc/utils"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type SaveLinkTraceEvent struct {
	EntryType int    `json:"entry_type"`
	EntryID   string `json:"entry_id"`
}

func HandleSaveLinkTrace(ctx context.Context, msgBody []byte) error {
	event := &SaveLinkTraceEvent{}
	err := sonic.Unmarshal(msgBody, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "unmarshal msg body:%s, error:%v", string(msgBody), err)
		return nil
	}

	hlog.CtxInfof(ctx, "save link trace begin, event:%v", event)

	err = doSave(ctx, event)
	if err != nil {
		hlog.CtxErrorf(ctx, "save link trace error:%v, event:%v", err, event)
		return err
	}

	hlog.CtxInfof(ctx, "save link trace success, event:%v", event)
	return nil
}

func doSave(ctx context.Context, event *SaveLinkTraceEvent) error {
	req := map[string]interface{}{
		"entry_type": event.EntryType,
		"entry_id":   event.EntryID,
	}
	_, err := utils.HttpPost(ctx, conf.GetConfig().Api.SaveLinkTrace, map[string]string{"Channel": "local", "env": "fc-20241124"}, req)
	if err != nil {
		hlog.CtxErrorf(ctx, "do save error:%v", err)
	}
	return err
}
