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

经整理，整个可能的调用链路如下：
|--api-chat.lingoreader.cn
|   |--qa-recommend.shenyandayi.com
|   |__qa-main.shenyandayi.com
|__api.lingoreader.cn
   |--wcd-v2.deeplang.net
   |   |__text-parse.shenyandayi.com
   |--pdfparser.shenyandayi.com
   |   |__text-parse.shenyandayi.com
   |--outlinecata.shenyandayi.com
   |--key-opinion.shenyandayi.com
   |--crawler.shenyandayi.com
   |   |__text-parse.shenyandayi.com
   |--summary.shenyandayi.com
   |--api-edu-arch.shenyandayi.com
   |   |__text-parse.shenyandayi.com
   |__api-repeater.lingoreader.cn
      |__ai-infra-service.shenyandayi.com
以上整理的调用链路可能并不会全部出现。你根据以上Nginx日志和错误日志(尤其是message、exc_info字段)，来分析一下，链路调用过程是怎样的，以及异常发生在哪个环节，可能是什么原因。如果没有异常，则分析一下耗时，看看耗时最长的链路是哪里、全链路耗时(出现的最早和最晚的时间戳之差)等。
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
