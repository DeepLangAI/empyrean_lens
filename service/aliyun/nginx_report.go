package aliyun

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	"empyrean_lens/utils"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type NginxTimeSpanReportModel struct {
	Date          string
	HostName      string
	CoreApiName   string
	CoreApiPath   string
	FailCount     int
	TotalCount    int
	FailRate      float64
	FailStatus    string
	FailStatus3xx int
	FailStatus4xx int
	FailStatus5xx int
}

func NginxTimespanReport(ctx context.Context, timespan int) ([]NginxTimeSpanReportModel, error) {
	var nginxLogs []aliyun.NginxLog

	if timespan == consts.TIMESPAN_LONGTIME {
		_nginxLogs, err := aliyun.NginxReportLongTime(ctx)
		if err != nil {
			return nil, err
		}
		nginxLogs = _nginxLogs
	} else if timespan == consts.TIMESPAN_WEEK {
		_nginxLogs, err := aliyun.NginxReportOneWeek(ctx)
		if err != nil {
			return nil, err
		}
		nginxLogs = _nginxLogs
	} else if timespan == consts.TIMESPAN_TODAY {
		_nginxLogs, err := aliyun.NginxLogsToday(ctx)
		if err != nil {
			return nil, err
		}
		nginxLogs = _nginxLogs
	}
	if len(nginxLogs) == 0 {
		return nil, nil
	}

	// day-host-api
	//timespanReports := map[string]map[string]map[string]NginxTimeSpanReportModel{}
	timespanReports := map[string]NginxTimeSpanReportModel{}
	for _, log := range nginxLogs {
		day := log.Time.Format("2006-01-02")
		key := fmt.Sprintf("%v\t%v\t%v", day, log.Host, log.CleanUrl)
		if strings.HasPrefix(log.Host, "pre") {
			fmt.Println(key)
		}
		apiReport, ok := timespanReports[key]
		if !ok {
			apiReport = NginxTimeSpanReportModel{
				Date:        log.Time.Format("2006-01-02"),
				HostName:    log.Host,
				CoreApiName: log.CleanUrl,
				CoreApiPath: log.CleanUrl,
			}
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
		timespanReports[key] = apiReport
		//timespanReports[day][log.Host][log.CleanUrl] = apiReport
	}
	finalReports := []NginxTimeSpanReportModel{}
	daySumReports := map[string]NginxTimeSpanReportModel{}
	//for _, dayReports := range timespanReports {
	//	for _, hostReports := range dayReports {
	//		for _, report := range hostReports {
	for _, report := range timespanReports {
		//fmt.Println("==========", host, url)
		if report.HostName == "pre-api.lingoreader.cn" {
			fmt.Println(report)
		}
		if report.HostName == "api.lingoreader.cn" {
			fmt.Println(report)
		}
		report.CoreApiName = utils.GetApiAlias(report.HostName, report.CoreApiName)
		finalReports = append(finalReports, report)

		daySumReport, ok := daySumReports[report.Date]
		if !ok {
			daySumReport = NginxTimeSpanReportModel{}
		}
		daySumReport.Date = report.Date
		daySumReport.CoreApiName = "当日总览"
		daySumReport.FailCount += report.FailCount
		daySumReport.TotalCount += report.TotalCount
		daySumReport.FailRate = float64(daySumReport.FailCount) / float64(daySumReport.TotalCount) * 100
		daySumReport.FailStatus3xx += report.FailStatus3xx
		daySumReport.FailStatus4xx += report.FailStatus4xx
		daySumReport.FailStatus5xx += report.FailStatus5xx
		daySumReports[report.Date] = daySumReport
	}
	//	}
	//	// finalReports按照Date字段降续排序
	//}
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

func NginxApiFailureDetail(ctx context.Context, req empyrean_lens.DailyApiFailureDetailReq) ([]*empyrean_lens.ApiFailureDetailRespData, error) {
	api, err := aliyun.NginxErrorLogsOfAPI(ctx, req.Host, req.Path, req.DateBegin)
	if err != nil {
		return nil, err
	}
	data := []*empyrean_lens.ApiFailureDetailRespData{}
	for _, log := range api {
		data = append(data, &empyrean_lens.ApiFailureDetailRespData{
			Time:     log.Time.Format("2006-01-02"),
			APIName:  log.CleanUrl,
			Host:     log.Host,
			Path:     log.CleanUrl,
			HTTPCode: log.Status,
			UserID:   log.UserId,
			TraceID:  log.TraceId,
		})
	}
	return data, nil
}

func EndToEndTraceLogs(ctx context.Context, req empyrean_lens.EndToEndTraceReq) ([]*empyrean_lens.EndToEndTraceRespData, error) {
	logs, err := aliyun.EndToEndLogsQuery(ctx, req.TraceID, req.DateBegin)
	if err != nil {
		return nil, err
	}
	data := []*empyrean_lens.EndToEndTraceRespData{}
	for _, log := range logs {
		data = append(data, &empyrean_lens.EndToEndTraceRespData{
			LogStoreName: log.LogStoreName,
			TraceID:      log.TraceId,
			UserID:       log.UserId,
			Time:         log.Time,
			Msg:          log.Message,
			Host:         log.Host,
			APIPath:      log.ApiPath,
			Cost:         float64(log.Cost),
			ClientIP:     log.ClientIp,
			Ua:           log.UA,
			Channel:      log.Channel,
			OriginLog:    log.OriginLog,
		})
	}
	return data, nil
}
