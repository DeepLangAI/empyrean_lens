package link_trace

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/tools"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func GetWcdOssLogDetail(ctx context.Context, req empyrean_lens.WcdOssDetalReq) ([]*empyrean_lens.WcdOssDetalRespData, *consts.BizCode) {
	ossList := []aliyun.WcdOssZipModel{}

	if req.OssBucket != "" && req.OssKey != "" {
		// 有指定oss bucket和key，直接下载
		ossList = append(ossList, aliyun.WcdOssZipModel{
			Bucket: req.OssBucket,
			Key:    req.OssKey,
		})
	} else {
		// 没有oss相关信息, 1. 获取文章详情
		webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, req.EntryID)
		if err != nil || webReaderInfo == nil {
			hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		start := webReaderInfo.CreateTime.Add(-1 * time.Hour)
		end := webReaderInfo.CreateTime.Add(24 * time.Hour)

		// 2. 获取oss列表
		ossList_, err := aliyun.WcdOsskeyQuery(ctx, req.TraceID, start, end)
		if err != nil {
			hlog.CtxErrorf(ctx, "get oss list failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		ossList = ossList_
	}
	ossOp := tools.GetOssOperator(ctx)
	result := make([]*empyrean_lens.WcdOssDetalRespData, 0)
	for _, oss := range ossList {
		file, err := ossOp.DownloadWcdOssFile(oss.Bucket, oss.Key)
		if err != nil {
			hlog.CtxErrorf(ctx, "download oss file failed, err: %v", err)
			continue
		}
		result = append(result, &empyrean_lens.WcdOssDetalRespData{
			RawHTML:          file.RawHtml,
			ParsedHTML:       file.ParsedHtml,
			TextParserLabels: file.TextParserLabels,
			Conclusion:       file.Conclusion,
			ModelInput:       file.ModelInput,
			ParsedText:       file.ParsedText,
		})
	}
	return result, nil
}

func queryInResult(query string, log aliyun.WcdWorthlessModel) bool {
	if strings.Contains(log.Url, query) {
		return true
	}
	if strings.Contains(log.Title, query) {
		return true
	}
	if strings.Contains(log.TraceId, query) {
		return true
	}
	if strings.Contains(log.WcdRequestId, query) {
		return true
	}
	return false
}

func GetWcdWorthlessLogs(ctx context.Context, req empyrean_lens.WcdWorthlessReq) ([]*empyrean_lens.WcdWorthlessRespData, error) {
	timeBegin, err := time.ParseInLocation(consts.DateHourMinSecTemplate, req.TimeBegin, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse time begin failed, err: %v", err)
		return nil, err
	}
	timeEnd, err := time.ParseInLocation(consts.DateHourMinSecTemplate, req.TimeEnd, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse time end failed, err: %v", err)
		return nil, err
	}
	logs, err := aliyun.WcdWorthlessQuery(ctx, timeBegin, timeEnd)
	if err != nil {
		hlog.CtxErrorf(ctx, "get oss list failed, err: %v", err)
		return nil, err
	}
	result := make([]*empyrean_lens.WcdWorthlessRespData, 0)
	for _, log := range logs {
		if req.Query != "" && !queryInResult(req.Query, log) {
			continue
		}
		result = append(result, &empyrean_lens.WcdWorthlessRespData{
			TraceID:      log.TraceId,
			WcdRequestID: log.WcdRequestId,
			URL:          log.Url,
			Host:         log.Host,
			Title:        log.Title,
			Time:         log.Time.Format(consts.DateHourMinSecTemplate),
			OssDlCmd:     log.OssDlCmd,
			OssBucket:    log.OssBucket,
			OssKey:       log.OssKey,
		})
	}
	return result, nil
}
