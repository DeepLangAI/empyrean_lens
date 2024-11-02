package link_trace

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/tools"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

func GetWcdOssLogDetail(ctx context.Context, req empyrean_lens.WcdOssDetalReq) ([]*empyrean_lens.WcdOssDetalRespData, *consts.BizCode) {
	// 获取文章详情
	webReaderInfo, err := plugin.NewWebReaderDao().FindWebReaderById(ctx, req.EntryID)
	if err != nil || webReaderInfo == nil {
		hlog.CtxErrorf(ctx, "get web reader info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	start := webReaderInfo.CreateTime.Add(-1 * time.Hour)
	end := webReaderInfo.CreateTime.Add(24 * time.Hour)

	ossList, err := aliyun.WcdOsskeyQuery(ctx, req.TraceID, start, end)
	if err != nil {
		hlog.CtxErrorf(ctx, "get oss list failed, err: %v", err)
		return nil, &consts.QueryRecordError
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
		})
	}
	return result, nil
}
