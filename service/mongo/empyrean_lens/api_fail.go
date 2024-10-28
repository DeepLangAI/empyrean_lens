package empyrean_lens

import (
	"context"
	el "empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/service/aliyun"
	"sort"
	"time"
)

func ApiFailResult(ctx context.Context, timeBegin, timeEnd time.Time) ([]aliyun.NginxTimeSpanReportModel, error) {
	models, err := el.NewApifailureModelDao().FindTimespanFailure(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	reports := []aliyun.NginxTimeSpanReportModel{}
	for _, model := range models {
		date := model.Date.Format("2006-01-02 15:04:05")
		reports = append(reports, aliyun.NginxTimeSpanReportModel{
			Date:             date,
			HostName:         model.HostName,
			CoreApiName:      model.ApiName,
			CoreApiPath:      model.ApiPath,
			FailCount:        int(model.FailCnt),
			TotalCount:       int(model.TotalCnt),
			FailRate:         float64(model.FailCnt+model.BizFailCnt) / float64(model.TotalCnt) * 100,
			FailStatus:       "",
			FailStatus3xx:    int(model.ErrCode3xxCnt),
			FailStatus4xx:    int(model.ErrCode4xxCnt),
			FailStatus5xx:    int(model.ErrCode5xxCnt),
			BizCodeFailCount: int(model.BizFailCnt),
		})
	}
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Date == reports[j].Date {
			if reports[i].CoreApiName == "当日总览" {
				return true
			} else if reports[j].CoreApiName == "当日总览" {
				return false
			}
			return reports[i].CoreApiName < reports[j].CoreApiName
			//if reports[i].HostName == reports[j].HostName {
			//	return reports[i].CoreApiName < reports[j].CoreApiName
			//} else {
			//	return reports[i].HostName < reports[j].HostName
			//}
		}
		return reports[i].Date > reports[j].Date
	})

	return reports, nil
}
