package empyrean_lens

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	empyrean_lens2 "empyrean_lens/dal/mongo/empyrean_lens"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

func UpsertOnlineOperations(ctx context.Context, req empyrean_lens.UploadOnlineOperationReq) (*empyrean_lens.UploadOnlineOperationRespData, error) {
	dao := empyrean_lens2.NewOnlineOperationModelDao()
	models := []empyrean_lens2.OnlineOperationModel{}
	data := &empyrean_lens.UploadOnlineOperationRespData{
		SuccessCnt: int32(len(req.Data)),
		FailMsgs:   make([]string, 0),
	}
	for i, op := range req.Data {
		t, err := time.ParseInLocation(consts.DateHourMinSecTemplate, op.Time, time.Local)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse time error: %v", err)
			data.SuccessCnt -= 1
			data.FailMsgs = append(data.FailMsgs, fmt.Sprintf("处理第%v条数据，时间解析失败：%v", i+1, err.Error()))
			continue
		}
		model := empyrean_lens2.OnlineOperationModel{
			Time:          t,
			AppName:       op.AppName,
			Builder:       op.Builder,
			BranchName:    op.BranchName,
			CommitMessage: op.CommitMessage,
			DomainName:    op.DomainName,
			RemoteUrl:     op.RemoteURL,
		}
		models = append(models, model)
	}
	err := dao.UpsertMany(ctx, models)
	if err != nil {
		hlog.CtxErrorf(ctx, "upsert online operation error: %v", err)
		return nil, err
	}
	return data, nil
}
