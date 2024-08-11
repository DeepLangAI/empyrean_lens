package empyrean_lens

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	empyrean_lens2 "empyrean_lens/dal/mongo/empyrean_lens"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

func SystemTracebackQuery(ctx context.Context, req empyrean_lens.TracebackReq) ([]*empyrean_lens.TracebackRespData, error) {
	t1, e := time.Parse("2006-01-02", req.DateBegin)
	if e != nil {
		hlog.CtxErrorf(ctx, "time parse err:%v", e)
		return nil, e
	}
	t2 := t1.AddDate(0, 0, 1)
	dao := empyrean_lens2.NewTracebackLogModelDao()
	logs, e := dao.FindTimespanTracebackLog(ctx, t1, t2)
	if e != nil {
		hlog.CtxErrorf(ctx, "FindTimespanTracebackLog err:%v", e)
		return nil, e
	}
	data := []*empyrean_lens.TracebackRespData{}
	for _, log := range logs {
		data = append(data, &empyrean_lens.TracebackRespData{
			ExcInfo:   log.Msg,
			Msg:       log.Msg,
			TraceID:   log.TraceId,
			UserID:    log.UserId,
			Time:      log.Time.Format("2006-01-02 15:04:05"),
			OriginLog: log.OriginLog,
		})
	}
	return data, nil
}
