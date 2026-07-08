package main

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/fc/cron/handlers"
	"fmt"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/logger"
	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"github.com/bytedance/sonic"
)

func FcInit(ctx context.Context) {
	conf.InitConfig()
	logger.Init(conf.GetConfig().Logger)
}

// Define the timer trigger event struct
type StructEvent struct {
	TriggerTime string
	TriggerName string
	Payload     string
}

const (
	TimerTriggerName_LingowhaleStability   = "lingowhale-stability"
	TimerTriggerName_LingowhaleDailyReport = "lingowhale-daily-report"
)

func FcHandler(ctx context.Context, event StructEvent) (string, error) {
	marshalString, _ := sonic.MarshalString(event)
	fmt.Println(marshalString)

	switch event.TriggerName {
	case TimerTriggerName_LingowhaleStability:
		return "", handlers.NewLingowhaleStability().Handle(ctx, event.Payload)
	case TimerTriggerName_LingowhaleDailyReport:
		// Payload 可传 "2026-07-07" 指定报表日（重跑/补发），空则默认昨天
		return "", handlers.NewLingowhaleDailyReport().Handle(ctx, event.Payload)
	}
	return "hello world", nil
}

func main() {
	fc.RegisterInitializerFunction(FcInit)
	fc.Start(FcHandler)
}
