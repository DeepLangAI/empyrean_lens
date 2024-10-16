package plugin

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/plugin"
	"empyrean_lens/utils"
	"sync"
	"time"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func GetUserAction(ctx context.Context, req empyrean_lens.GetUserActionReq) ([]*empyrean_lens.GetUserActionRespData, *consts.BizCode) {
	t1, _ := time.Parse(consts.DateHourMinuteTemplate, req.StartTime)
	t2, _ := time.Parse(consts.DateHourMinuteTemplate, req.EndTime)
	if req.UID == "" && (t1.IsZero() || t2.IsZero()) {
		return nil, &consts.RetParamError
	}

	res := make([]*empyrean_lens.GetUserActionRespData, 0)

	funcList, mu := []utillib.AsyncFunc{}, &sync.Mutex{}
	funcList = append(funcList, func() error {
		fileActions, err := plugin.NewFileDao().FindFileByUserIdAndCreateTime(ctx, req.UID, t1, t2)
		if err != nil {
			hlog.CtxErrorf(ctx, "GetUserAction FindFileByUserIdAndCreateTime err:%+v", err)
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		for i := range fileActions {
			res = append(res, &empyrean_lens.GetUserActionRespData{
				UID:     fileActions[i].UserID,
				Time:    fileActions[i].CreateTime.Format(consts.DateHourMinuteTemplate),
				Action:  consts.PDF,
				Title:   fileActions[i].Name,
				Success: fileActions[i].IsGenerated,
				ID:      fileActions[i].ID.String(),
			})
		}

		return nil
	})
	funcList = append(funcList, func() error {
		var err error
		webReaderActions, err := plugin.NewWebReaderDao().FindWebReaderByUserIdAndCreateTime(ctx, req.UID, t1, t2)
		if err != nil {
			hlog.CtxErrorf(ctx, "GetUserAction FindWebReaderByUserIdAndCreateTime err:%+v", err)
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		for i := range webReaderActions {
			res = append(res, &empyrean_lens.GetUserActionRespData{
				UID:     webReaderActions[i].UserID,
				Time:    webReaderActions[i].CreateTime.Format(consts.DateHourMinuteTemplate),
				Action:  consts.URL,
				Title:   webReaderActions[i].Title,
				Success: webReaderActions[i].IsGenerated,
				ID:      webReaderActions[i].ID.String(),
			})
		}

		return nil
	})
	funcList = append(funcList, func() error {
		var err error
		multiActions, err := plugin.NewMultiDao().FindMultiByUserIdAndCreateTime(ctx, req.UID, t1, t2)
		if err != nil {
			hlog.CtxErrorf(ctx, "GetUserAction FindMultiModlByUserIdAndCreateTime err:%+v", err)
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		for i := range multiActions {
			res = append(res, &empyrean_lens.GetUserActionRespData{
				UID:    multiActions[i].UserID,
				Time:   multiActions[i].CreateTime.Format(consts.DateHourMinuteTemplate),
				Action: consts.MULTI,
				Title:  multiActions[i].Title,
				// todo  Success: multiActions[i].IsGeneratedaa,
				ID: multiActions[i].Id.String(),
			})
		}

		return nil
	})

	err := utillib.ParallelExec(ctx, funcList, len(funcList))
	if len(err) > 0 {
		hlog.CtxErrorf(ctx, "GetUserAction err:%+v", utils.JSONMarshal(err))
		return nil, &consts.SystemErr
	}
	return res, nil
}
