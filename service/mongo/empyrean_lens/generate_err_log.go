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

func SaveOnceGenerateErrlogByDate() {
	// 起始日期和结束日期
	startDate := "2024-11-01"
	endDate := "2024-11-28"

	// 将字符串转换为 time.Time 类型
	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		hlog.Errorf("Invalid start date format: %v", err)
		return
	}
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		hlog.Errorf("Invalid start date format: %v", err)
		return
	}

	// 循环从 startDate 到 endDate 每天调用一次 SaveGenerateErrlogByDate
	for startTime.Before(endTime) || startTime.Equal(endTime) {
		dateStr := startTime.Format("2006-01-02")
		SaveGenerateErrlogByDate(context.Background(), dateStr)
		startTime = startTime.AddDate(0, 0, 1) // 增加一天
	}

}
