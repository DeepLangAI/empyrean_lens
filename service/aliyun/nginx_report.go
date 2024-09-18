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
	"sync"
	"time"
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

	timespanReports := map[string]NginxTimeSpanReportModel{}
	for _, log := range nginxLogs {
		day := log.Time.Format(consts.DateTemplate)
		key := fmt.Sprintf("%v\t%v\t%v", day, log.Host, log.CleanUrl)
		apiReport, ok := timespanReports[key]
		if !ok {
			apiReport = NginxTimeSpanReportModel{
				Date:        log.Time.Format(consts.DateTemplate),
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
	}
	finalReports := []NginxTimeSpanReportModel{}
	daySumReports := map[string]NginxTimeSpanReportModel{}
	for _, report := range timespanReports {
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
	// finalReports按照Date字段降续排序
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
			Time:     log.Time.Format(consts.DateHourMinSecTemplate),
			APIName:  log.CleanUrl,
			Host:     log.Host,
			Path:     log.CleanUrl,
			HTTPCode: log.Status,
			UserID:   log.UserId,
			TraceID:  log.TraceId,
			ClientIP: log.ClientIp,
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

func EndToEndUserTraceLogs(ctx context.Context, req empyrean_lens.EndToEndUserTraceReq) ([]*empyrean_lens.EndToEndUserTraceRespData, error) {
	logs, err := aliyun.EndToEndUserLogsQuery(ctx, req.UserID, req.TimeBegin, req.TimeEnd)
	if err != nil {
		return nil, err
	}
	parsedLogs := []*empyrean_lens.EndToEndTraceRespData{}
	for _, log := range logs {
		parsedLogs = append(parsedLogs, &empyrean_lens.EndToEndTraceRespData{
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
	data := []*empyrean_lens.EndToEndUserTraceRespData{}

	groupedPaths := map[string][]string{
		"数据处理": {
			"/crawl",
			"/wcd-raw",
			"/edu_parse",
			"/api/plugin/file/add",
			"/api/readers/url/upload",
			"/api/readers/url/content/upload",
		},
		"摘录": {
			"/api/plugin/extract/detail",
		},
		"单文档": {
			"/api/repeater/abstract",
			"/api/repeater/outline",
			"/api/repeater/viewpoint",
			"/api/plugin/articles/summary",
		},
		"问答": {
			"/qa/main",
			"/qa/query_recommend",
		},
		"多文档": {
			"/doc/single/analyze",
			"/doc/multi/analyze",
			"/doc/multi/outline",
			"/multi-doc/single-doc-analysis",
			"/multi-doc/doc-merge",
			"/multi-doc/doc-summary",
		},
		"订阅": {
			"/api/feed/v1/subscription/upsert",
		},
	}
	groupedLogs := map[string][]*empyrean_lens.EndToEndTraceRespData{}
	for _, log := range parsedLogs {
		key := ""
		for _key, paths := range groupedPaths {
			if utils.Contains(paths, log.APIPath) {
				key = _key
				break
			}
		}
		if key == "" {
			if strings.Contains(log.APIPath, "safety") {
				key = "安全"
			}
		}
		if key == "" {
			key = "其他"
		}
		if groupedLogs[key] == nil {
			groupedLogs[key] = []*empyrean_lens.EndToEndTraceRespData{}
		}
		groupedLogs[key] = append(groupedLogs[key], log)
	}
	keys := utils.KeysOfMap(groupedLogs)
	orderedKeys := []string{"单文档", "多文档", "问答", "摘录", "数据处理", "订阅", "安全", "其他"}
	sort.Slice(keys, func(i, j int) bool {
		idxI := utils.Index(orderedKeys, keys[i])
		idxJ := utils.Index(orderedKeys, keys[j])
		if idxI != -1 && idxJ != -1 {
			return idxI < idxJ
		}
		if idxI == -1 && idxJ == -1 {
			return keys[i] < keys[j]
		}
		return idxI == -1
	})
	for _, key := range keys {
		sort.Slice(groupedLogs[key], func(i, j int) bool {
			return groupedLogs[key][i].Time >= groupedLogs[key][j].Time
		})
		data = append(data, &empyrean_lens.EndToEndUserTraceRespData{
			Scene: key,
			Logs:  groupedLogs[key],
		})
	}

	return data, nil
}

func RequestTrend(ctx context.Context, req empyrean_lens.RequestTrendReq) (*empyrean_lens.RequestTrendRespData, error) {
	date, err := time.Parse(consts.DateTemplate, req.Date)
	if err != nil {
		return nil, err
	}

	data := &empyrean_lens.RequestTrendRespData{}
	//daysSince := int(date.Sub(time.Now()).Hours() / 24)
	daysSince := int(time.Now().Sub(date).Hours() / 24)
	wg := sync.WaitGroup{}
	for _, daysLookback := range []int{0, 1, 7} {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			logs, err := aliyun.NginxLogsDaysAgo(ctx, daysSince+daysLookback)
			if err != nil {
				return
			}
			trend := map[string]int{}
			for _, log := range logs {
				timestamp := log.Time.Format(consts.DateHourTemplate)
				//timestamp := log.Time.Format(consts.DateHourMinuteTemplate)
				timestamp = fmt.Sprintf("%v:%02d:00", timestamp[:len(timestamp)-6], log.Time.Minute()-(log.Time.Minute()%15))
				//log.Time.Minute() % 10
				trend[timestamp] += 1
			}
			trendItems := []*empyrean_lens.RequestTrendRespDataItem{}
			for timestamp, cnt := range trend {
				trendItems = append(trendItems, &empyrean_lens.RequestTrendRespDataItem{
					Time:  timestamp,
					Count: int32(cnt),
				})
			}
			sort.Slice(trendItems, func(i, j int) bool {
				return trendItems[i].Time < trendItems[j].Time
			})
			if daysLookback == 0 {
				data.Data0 = trendItems
			} else if daysLookback == 1 {
				data.Data1 = trendItems
			} else if daysLookback == 7 {
				data.Data7 = trendItems
			}
		}(daysLookback)
	}
	wg.Wait()
	return data, err
}
