package service

import (
	"context"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/utils"
	"sort"
	"strconv"
	"strings"
)

type NginxMonthReportModel struct {
	Date          string
	HostName      string
	CoreApiName   string
	FailCount     int
	TotalCount    int
	FailRate      float64
	FailStatus    string
	FailStatus3xx int
	FailStatus4xx int
	FailStatus5xx int
}

func NginxMonthReport(ctx context.Context) ([]NginxMonthReportModel, error) {
	//monthLogs := []aliyun.NginxLog{}
	//nlogs, _ := aliyun.NginxIngressLogQuery(ctx, 0)
	//mlogs, _ := aliyun.ModelNginxIngressLogQuery(ctx, 0)
	//monthLogs = append(monthLogs, nlogs...)
	//monthLogs = append(monthLogs, mlogs...)

	monthLogs, err := aliyun.NginxReportThisMonth(ctx)
	if err != nil {
		return nil, err
	}

	// day-host-api
	monthReports := map[int]map[string]map[string]NginxMonthReportModel{}
	for _, log := range monthLogs {
		day := log.Time.Day()
		dayReports, ok := monthReports[day]
		if !ok {
			dayReports = map[string]map[string]NginxMonthReportModel{}
			monthReports[day] = dayReports
		}

		hostReports, ok := dayReports[log.Host]
		if !ok {
			hostReports = map[string]NginxMonthReportModel{}
			dayReports[log.Host] = hostReports
		}

		apiReport, ok := hostReports[log.CleanUrl]
		if !ok {
			apiReport = NginxMonthReportModel{
				Date:        log.Time.Format("2006-01-02"),
				HostName:    log.Host,
				CoreApiName: log.CleanUrl,
			}
			hostReports[log.CleanUrl] = apiReport
		}

		apiReport.TotalCount += 1
		if log.Status != "200" {
			statusCode, _ := strconv.ParseInt(log.Status, 10, 64)
			if statusCode >= 300 && statusCode < 400 {
				apiReport.FailStatus3xx += 1
			} else if statusCode >= 400 && statusCode < 500 {
				apiReport.FailStatus4xx += 1
			} else if statusCode >= 500 {
				apiReport.FailStatus5xx += 1
			}
			apiReport.FailCount += 1
			codes := strings.Split(apiReport.FailStatus, ",")
			if !utils.Contains(codes, log.Status) {
				codes = append(codes, log.Status)
				apiReport.FailStatus = strings.Join(codes, ",")
				apiReport.FailStatus = strings.TrimLeft(apiReport.FailStatus, ",")
			}
		}
		apiReport.FailRate = float64(apiReport.FailCount) / float64(apiReport.TotalCount) * 100
		monthReports[day][log.Host][log.CleanUrl] = apiReport
	}
	finalReports := []NginxMonthReportModel{}
	daySumReports := map[string]NginxMonthReportModel{}
	for _, dayReports := range monthReports {
		for _, hostReports := range dayReports {
			for _, report := range hostReports {
				//fmt.Println("==========", host, url)
				report.CoreApiName = utils.GetApiAlias(report.HostName, report.CoreApiName)
				finalReports = append(finalReports, report)

				daySumReport, ok := daySumReports[report.Date]
				if !ok {
					daySumReport = NginxMonthReportModel{}
				}
				daySumReport.Date = report.Date
				daySumReport.CoreApiName = "当日总览"
				daySumReport.FailCount += report.FailCount
				daySumReport.TotalCount += report.TotalCount
				daySumReport.FailRate = float64(daySumReport.FailCount) / float64(daySumReport.TotalCount) * 100
				daySumReport.FailStatus3xx += report.FailStatus3xx
				daySumReport.FailStatus3xx += report.FailStatus4xx
				daySumReport.FailStatus5xx += report.FailStatus5xx
				daySumReports[report.Date] = daySumReport
			}
		}
		// finalReports按照Date字段降续排序
	}
	finalReports = append(finalReports, utils.ValuesOfMap(daySumReports)...)
	sort.Slice(finalReports, func(i, j int) bool {
		if finalReports[i].Date == finalReports[j].Date {
			if finalReports[i].CoreApiName == "当日总览" {
				return true
			} else if finalReports[j].CoreApiName == "当日总览" {
				return false
			}
			if finalReports[i].HostName == finalReports[j].HostName {
				return finalReports[i].CoreApiName < finalReports[j].CoreApiName
			} else {
				return finalReports[i].HostName < finalReports[j].HostName
			}
		}
		return finalReports[i].Date > finalReports[j].Date
	})
	return finalReports, nil
}
