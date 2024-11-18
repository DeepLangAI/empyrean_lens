package main

import (
	"context"
	"encoding/base64"
	"os"

	"empyrean_lens/fc/conf"
	"empyrean_lens/fc/tasks"
	"empyrean_lens/fc/utils"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/logger"
	"github.com/aliyun/fc-runtime-go-sdk/events"
	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func Init(ctx context.Context) {
	conf.InitConfig()
	logger.Init(conf.GetConfig().Logger)
	hlog.Info("Init success")
}

func main() {
	fc.RegisterInitializerFunction(Init)
	if os.Getenv("event_type") == "timer" {
		Init(context.Background())
		fc.Start(HandleMnsTimerRequest)
	} else {
		Init(context.Background())
		fc.Start(HandleMnsQueueRequest)
	}
}

func HandleMnsTimerRequest(ctx context.Context, event events.TimerEvent) error {
	switch *event.Payload {
	case conf.GetConfig().MNS.BatchSaveLinkTrace.Name:
		return tasks.HandleBatchSaveLinkTraceEvent(ctx)
	}
	return nil
}

func HandleMnsQueueRequest(ctx context.Context, event events.MnsQueueEvent) error {
	if event.Data.MessageBody == nil {
		hlog.CtxErrorf(ctx, "msg body is nil")
		return nil
	}
	bodyBytes, err := base64.StdEncoding.DecodeString(*event.Data.MessageBody)
	if err != nil {
		hlog.CtxErrorf(ctx, "decode msg body:%v, error:%v", *event.Data.MessageBody, err)
		return err
	}
	switch utils.GetQueueNameFromSubject(*event.Subject) {
	case conf.GetConfig().MNS.SaveLinkTrace.Name:
		return tasks.HandleSaveLinkTrace(ctx, bodyBytes)
	}
	return nil
}
