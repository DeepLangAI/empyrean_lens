package service

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/utils"
	"sort"
	"strconv"
	"strings"
)

type NginxMonthReportModel struct {
	Date          string
	Scene         string
	CoreApiName   string
	FailCount     int
	TotalCount    int
	FailRate      float64
	FailStatus    string
	FailStatus4xx int
	FailStatus5xx int
}

func NginxMonthReport(ctx context.Context) ([]NginxMonthReportModel, error) {

	monthLogs, err := aliyun.NginxReportThisMonth(ctx)
	if err != nil {
		return nil, err
	}
	monthReports := map[int]map[string]NginxMonthReportModel{}
	for _, log := range monthLogs {
		day := log.Time.Day()
		dayReports, ok := monthReports[day]
		if !ok {

			dayReports = map[string]NginxMonthReportModel{}
			monthReports[day] = dayReports
		}

		report, ok := dayReports[log.CleanUrl]
		if !ok {
			report = NginxMonthReportModel{
				Date:        log.Time.Format("2006-01-02"),
				CoreApiName: log.CleanUrl,
			}
		}
		report.TotalCount += 1
		if log.Status != "200" {
			statusCode, _ := strconv.ParseInt(log.Status, 10, 64)
			if statusCode >= 400 && statusCode < 500 {
				report.FailStatus4xx += 1
			} else if statusCode >= 500 {
				report.FailStatus5xx += 1
			}
			report.FailCount += 1
			codes := strings.Split(report.FailStatus, ",")
			if !utils.Contains(codes, log.Status) {
				codes = append(codes, log.Status)
				report.FailStatus = strings.Join(codes, ",")
				report.FailStatus = strings.TrimLeft(report.FailStatus, ",")
			}
		}
		report.FailRate = float64(report.FailCount) / float64(report.TotalCount) * 100
		monthReports[day][log.CleanUrl] = report
	}
	finalReports := []NginxMonthReportModel{}
	for _, dayReports := range monthReports {
		for _, report := range dayReports {
			report.CoreApiName = consts.CORE_APIS[report.CoreApiName]
			finalReports = append(finalReports, report)
		}
	}
	// finalReports按照Date字段降续排序
	sort.Slice(finalReports, func(i, j int) bool {
		if finalReports[i].Date == finalReports[j].Date {
			return finalReports[i].CoreApiName < finalReports[j].CoreApiName
		}
		return finalReports[i].Date > finalReports[j].Date
	})
	return finalReports, nil
}
