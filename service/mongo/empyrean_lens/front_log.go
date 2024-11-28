package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"sort"
)

type userError struct{}

var UserErrorService *userError

func (u *userError) GeneralErrorList(ctx context.Context, req empyrean_lens2.UserErrorListReq) ([]*empyrean_lens2.UserErrorListRespData, error) {
	resMap := map[string]*empyrean_lens2.UserErrorListRespData{}
	uploadLogDal := empyrean_lens.NewUploadLogModelDao()
	uploadLogs, err := uploadLogDal.CountLogsByTimeRange(ctx, consts.UserErrorStartDate, "")
	if err != nil {
		hlog.CtxErrorf(ctx, "CountLogsByTimeRange err: %v", err)
		return nil, err
	}

	generateErrlogDal := empyrean_lens.NewGeneratorErrlogModelDao()
	generateLogs, err := generateErrlogDal.CountGenerateErrLogsByTimeRange(ctx, consts.UserErrorStartDate, "")
	if err != nil {
		hlog.CtxErrorf(ctx, "CountGenerateErrLogsByTimeRange err: %v", err)
		return nil, err
	}
	for date, errCnt := range uploadLogs {
		if resMap[date] == nil {
			resMap[date] = &empyrean_lens2.UserErrorListRespData{}
		}
		resMap[date].NumErrorUpload = errCnt
		resMap[date].Date = date
	}
	for date, errCnt := range generateLogs {
		if resMap[date] == nil {
			resMap[date] = &empyrean_lens2.UserErrorListRespData{}
		}
		resMap[date].NumErrorGenerate = errCnt
		resMap[date].Date = date
	}
	res := []*empyrean_lens2.UserErrorListRespData{}
	res = append(res, utils.ValuesOfMap(resMap)...)
	sort.Slice(res, func(i, j int) bool {
		return res[i].Date > res[j].Date
	})
	return res, nil
}

func filesizeToReadable(size int64) string {
	//	size, 单位B
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2fKB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2fMB", float64(size)/1024/1024)
	} else {
		return fmt.Sprintf("%.2fGB", float64(size)/1024/1024/1024)
	}
}

func (u *userError) UploadErrorList(ctx context.Context, req empyrean_lens2.UserErrorUploadReq) ([]*empyrean_lens2.UploadErrorLog, error) {
	dao := empyrean_lens.NewUploadLogModelDao()
	logs, err := dao.GetUploadInfoByTime(ctx, req.Date)
	if err != nil {
		hlog.CtxErrorf(ctx, "GetUploadInfoByTime err: %v", err)
		return nil, err
	}
	result := []*empyrean_lens2.UploadErrorLog{}
	for _, log := range logs {
		result = append(result, &empyrean_lens2.UploadErrorLog{
			Date:          log.Date,
			UserID:        log.UserId,
			Time:          log.Time.Format(consts.DateHourMinSecTemplate),
			FailureReason: log.FailureReason,
			FileName:      log.FileName,
			FileSize:      filesizeToReadable(log.FileSize),
			FileType:      log.FileType,
			IP:            log.Ip,
			IPRegion:      log.IpRegion,
			TraceID:       log.TraceId,
		})
	}
	return result, nil
}

func (u *userError) GenerateErrorList(ctx context.Context, req empyrean_lens2.UserErrorGenerateReq) ([]*empyrean_lens2.GenerateErrorLog, error) {
	dao := empyrean_lens.NewGeneratorErrlogModelDao()
	logs, err := dao.GetGenerateErrlogByTime(ctx, req.Date)
	if err != nil {
		hlog.CtxErrorf(ctx, "GetGenerateErrlogByTime err: %v", err)
		return nil, err
	}
	result := []*empyrean_lens2.GenerateErrorLog{}
	for _, log := range logs {
		result = append(result, &empyrean_lens2.GenerateErrorLog{
			Time:          log.Time.Format(consts.DateHourMinSecTemplate),
			UserID:        log.UserId,
			Date:          log.Date,
			EntryID:       log.EntryId,
			FailureReason: log.FailureReason,
			IP:            log.Ip,
			IPRegion:      log.IpRegion,
			TraceID:       log.TraceId,
		})
	}
	return result, nil
}
