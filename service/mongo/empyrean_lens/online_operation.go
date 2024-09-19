package empyrean_lens

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	empyrean_lens2 "empyrean_lens/dal/mongo/empyrean_lens"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"sort"
	"strings"
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

func FindOnlineOperations(ctx context.Context, req empyrean_lens.OnlineOperationReq) ([]*empyrean_lens.OnlineOperationRespData, error) {
	dao := empyrean_lens2.NewOnlineOperationModelDao()
	timeBegin, err := time.ParseInLocation(consts.DateHourMinSecTemplate, req.TimeBegin, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse time error: %v", err)
		return nil, err
	}
	timeEnd, err := time.ParseInLocation(consts.DateHourMinSecTemplate, req.TimeEnd, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse time error: %v", err)
		return nil, err
	}
	models, err := dao.FindTimespanModels(ctx, timeBegin, timeEnd)
	if err != nil {
		hlog.CtxErrorf(ctx, "find online operation error: %v", err)
		return nil, err
	}
	modelsGrouped := make(map[string][]*empyrean_lens.UploadOnlineOperationReqData)
	for _, model := range models {
		if req.AppName != "" && strings.Contains(model.AppName, req.AppName) {
			continue
		}
		if _, ok := modelsGrouped[model.AppName]; !ok {
			modelsGrouped[model.AppName] = []*empyrean_lens.UploadOnlineOperationReqData{}
		}
		modelsGrouped[model.AppName] = append(modelsGrouped[model.AppName], &empyrean_lens.UploadOnlineOperationReqData{
			Builder:       model.Builder,
			BranchName:    model.BranchName,
			CommitMessage: model.CommitMessage,
			DomainName:    model.DomainName,
			Time:          model.Time.Format(consts.DateHourMinuteTemplate),
			AppName:       model.AppName,
			RemoteURL:     model.RemoteUrl,
		})
	}
	result := make([]*empyrean_lens.OnlineOperationRespData, 0)
	for appName, groupModels := range modelsGrouped {
		sort.Slice(groupModels, func(i, j int) bool {
			// 降序
			return groupModels[i].Time >= groupModels[j].Time
		})
		result = append(result, &empyrean_lens.OnlineOperationRespData{
			AppName: appName,
			Detail:  groupModels,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Detail[0].Time >= result[j].Detail[0].Time
	})
	return result, nil
}
