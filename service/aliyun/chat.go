package aliyun

import (
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/utils"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"strings"
)

func BuildLogAnlzPrompt(ctx context.Context, traceId, date string) string {
	logs, err := aliyun.EndToEndLogsQuery(ctx, traceId, date)
	if err != nil {
		hlog.CtxErrorf(ctx, "err: %s", err)
		return ""
	}
	prompt := `你现在是一个日志分析专家，请根据以下日志，分析为什么出现了异常，涉及的整个链路是怎样调用的（先后来到哪些服务（主机名等）、调用了哪些函数（函数名等）），以及异常在链路的什么位置，并给出详细的分析和解决建议。

先是Nginx日志：
%v

然后是错误日志：
%v

你根据以上Nginx日志和错误日志(尤其是message、exc_info字段)，来分析一下，链路调用过程是怎样的，以及异常发生在哪个环节，可能是什么原因。如果没有异常，则分析一下耗时，看看耗时最长的链路是哪里、全链路耗时(出现的最早和最晚的时间戳之差)等。
`
	nginxLogs := []string{}
	errorLogs := []string{}
	for _, log := range logs {
		if log.LogStoreName == "nginx-ingress" || log.LogStoreName == "model-nginx-ingress" {
			if !strings.Contains(log.Host, "verbose") {
				nginxLogs = append(nginxLogs, utils.JSONMarshal(log))
			}
		} else {
			if strings.ToUpper(log.OriginLog["level"]) == "ERROR" {
				errorLogs = append(errorLogs, utils.JSONMarshal(log))
			}
		}
	}
	prompt = fmt.Sprintf(prompt, strings.Join(nginxLogs, "\n"), strings.Join(errorLogs, "\n"))
	return prompt
}
