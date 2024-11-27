package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/http"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

func SaveGenerateErrlogByDate(ctx context.Context, dateStr string) error {
	generateErrInfos, err := http.ShenceDal.GetGenerateErrLogByTime(ctx, dateStr)
	if err != nil {
		hlog.Errorf("get generate error log from shence failed.err %v", err)
		return err
	}
	if len(generateErrInfos) == 0 {
		hlog.Debugf("no generate error log need to save. date %s", dateStr)
		return nil
	}

	generateErrLogDal := empyrean_lens.NewGeneratorErrlogModelDao()
	models := make([]empyrean_lens.GenerateErrLogModel, 0)
	for _, generateErrInfo := range generateErrInfos {
		models = append(models, empyrean_lens.GenerateErrLogModel{
			Date:          dateStr,
			EntryId:       generateErrInfo.EntryId,
			FailureReason: generateErrInfo.FailureReason,
			Ip:            generateErrInfo.Ip,
			IpRegion:      utils.GetIPLocation(generateErrInfo.Ip),
			Time:          generateErrInfo.Time,
			TraceId:       generateErrInfo.TraceId,
			UpdateTime:    time.Now().UTC(),
			UserId:        generateErrInfo.UserId,
		})
	}
	return generateErrLogDal.SaveBatch(ctx, models)
}
