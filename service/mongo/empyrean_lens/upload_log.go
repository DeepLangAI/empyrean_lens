package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/http"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

func SaveUploadLogByDate(ctx context.Context, dateStr string) error {
	uploadInfos, err := http.ShenceDal.GetUploadInfoByTime(ctx, dateStr)
	if err != nil {
		hlog.Errorf("get upload log from shence failed.err %v", err)
		return err
	}
	if len(uploadInfos) == 0 {
		hlog.Debugf("no upload log need to save. date %s", dateStr)
		return nil
	}

	uploadLogModel := empyrean_lens.NewUploadLogModelDao()
	models := make([]empyrean_lens.UploadLogModel, 0)
	for _, uploadInfo := range uploadInfos {
		models = append(models, empyrean_lens.UploadLogModel{
			Date:          uploadInfo.Date,
			FailureReason: uploadInfo.FailureReason,
			FileName:      uploadInfo.FileName,
			FileSize:      uploadInfo.FileSize,
			FileType:      uploadInfo.FileType,
			Ip:            uploadInfo.Ip,
			IpRegion:      utils.GetIPLocation(uploadInfo.Ip),
			Time:          uploadInfo.Time,
			TraceId:       uploadInfo.TraceId,
			UserId:        uploadInfo.UserId,
			UpdateTime:    time.Now().UTC(),
		})
	}
	return uploadLogModel.SaveBatch(ctx, models)
}

func SaveOnceUploadLogByDate() {
	// 起始日期和结束日期
	startDate := "2024-11-01"
	endDate := "2024-11-28"

	// 将字符串转换为 time.Time 类型
	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		hlog.CtxErrorf(context.Background(), "Invalid start date format: %v", err)
		return
	}
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		hlog.CtxErrorf(context.Background(), "Invalid start date format: %v", err)
		return
	}

	// 循环从 startDate 到 endDate 每天调用一次 SaveGenerateErrlogByDate
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		dateStr := startTime.Format("2006-01-02")
		SaveUploadLogByDate(context.Background(), dateStr)
		startTime = startTime.AddDate(0, 0, 1) // 增加一天
	}
}
