package service

import (
	"context"
	"empyrean_lens/aliyun"
	"empyrean_lens/dal/mongo"
	"time"
)

func ApiFailResult(ctx context.Context, timeBegin, timeEnd time.Time) ([]aliyun.NginxTimeSpanReportModel, error) {
	models, err := mongo.NewApifailureModelDao().FindTimespanFailure(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	reports := []aliyun.NginxTimeSpanReportModel{}
	for _, model := range models {
		date := model.Date.Format("2006-01-02")
		reports = append(reports, aliyun.NginxTimeSpanReportModel{
			Date:          date,
			HostName:      model.HostName,
			CoreApiName:   model.ApiName,
			FailCount:     int(model.FailCnt),
			TotalCount:    int(model.TotalCnt),
			FailRate:      float64(model.FailCnt) / float64(model.TotalCnt),
			FailStatus:    "",
			FailStatus3xx: int(model.ErrCode3xxCnt),
			FailStatus4xx: int(model.ErrCode4xxCnt),
			FailStatus5xx: int(model.ErrCode5xxCnt),
		})
	}

	return reports, nil
}
