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
