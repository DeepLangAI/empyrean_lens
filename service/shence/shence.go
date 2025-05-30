package shence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"empyrean_lens/dal/mongo/empyrean_lens"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

const (
	shenceAPIURL = "https://shenyankeji.cloud.sensorsdata.cn/api/v3/analytics/v1/model/sql/query"
	apiKey       = "#K-EkE7btXHodiCAwtJqyD3PPzDhrk0jYV5"
	project      = "production"
)

// parseShenceTime 解析神策时间格式
func parseShenceTime(timeStr string) (time.Time, error) {
	// 尝试解析带毫秒的时间格式
	t, err := time.Parse("2006-01-02 15:04:05.000", timeStr)
	if err == nil {
		return t, nil
	}
	// 尝试解析不带毫秒的时间格式
	t, err = time.Parse("2006-01-02 15:04:05", timeStr)
	if err == nil {
		return t, nil
	}
	return time.Time{}, err
}

// getTimeInterval 根据总耗时确定时间区间
func getTimeInterval(totalDuration float64) string {
	switch {
	case totalDuration >= 0 && totalDuration < 2000:
		return "0-2000"
	case totalDuration >= 2000 && totalDuration < 3000:
		return "2000-3000"
	case totalDuration >= 3000 && totalDuration < 5000:
		return "3000-5000"
	case totalDuration >= 5000:
		return ">5000"
	default:
		return "0-2000" // 处理负数等异常情况
	}
}

// SyncApiPerformanceStats 从神策同步API性能统计数据到MongoDB
func SyncApiPerformanceStats(ctx context.Context, startTime, endTime time.Time) error {
	// 确保集合存在
	dao := empyrean_lens.NewApiPerformanceStatsDao()
	if err := dao.EnsureCollection(ctx); err != nil {
		hlog.CtxErrorf(ctx, "ensure collection error: %v", err)
		return fmt.Errorf("ensure collection error: %v", err)
	}

	// 获取iOS和Android最新一条记录的时间
	iosLatest, iosErr := dao.GetLatestCreatedAtBySystem(ctx, "iOS")
	androidLatest, androidErr := dao.GetLatestCreatedAtBySystem(ctx, "Android")

	// 取最早的那个
	var minLatest time.Time
	if iosErr != nil && androidErr != nil {
		// 如果两个系统都没有记录，使用传入的开始时间
		minLatest = startTime
	} else if iosErr != nil {
		minLatest = androidLatest
	} else if androidErr != nil {
		minLatest = iosLatest
	} else if iosLatest.Before(androidLatest) {
		minLatest = iosLatest
	} else {
		minLatest = androidLatest
	}

	// 取最早的date的0点
	if !minLatest.IsZero() {
		minLatestZero := time.Date(minLatest.Year(), minLatest.Month(), minLatest.Day(), 0, 0, 0, 0, minLatest.Location()).AddDate(0, 0, -1)
		if minLatestZero.Before(startTime) {
			startTime = minLatestZero
		}
	}

	// 构建SQL查询
	sql := fmt.Sprintf(`
		SELECT
			date,
			$os,
			$os_version,
			AVG(perf_article_title_shown) OVER (PARTITION BY date, $os) AS avg_duration,
			COUNT(perf_article_title_shown) OVER (PARTITION BY date, $os) AS total_requests,
			perf_article_title_shown,
			time
		FROM
			events
		WHERE
			event = 'Performance_Metrics_PerformanceTime'
			AND date BETWEEN '%s' AND '%s'
		    AND $os IN ('iOS', 'Android')
		ORDER BY
		    date DESC, $os ASC, time ASC
	`, startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))

	hlog.CtxInfof(ctx, "神策查询SQL: %s", sql)

	// 构建请求体
	reqBody := map[string]interface{}{
		"sql":   sql,
		"limit": 10000000,
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		hlog.CtxErrorf(ctx, "marshal request body error: %v", err)
		return fmt.Errorf("marshal request body error: %v", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", shenceAPIURL, nil)
	if err != nil {
		hlog.CtxErrorf(ctx, "create request error: %v", err)
		return fmt.Errorf("create request error: %v", err)
	}
	httpReq.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", apiKey)
	httpReq.Header.Set("sensorsdata-project", project)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		hlog.CtxErrorf(ctx, "send request error: %v", err)
		return fmt.Errorf("send request error: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		hlog.CtxErrorf(ctx, "read response error: %v", err)
		return fmt.Errorf("read response error: %v", err)
	}

	// 打印响应内容以便调试
	hlog.CtxInfof(ctx, "神策API响应内容: %s", string(respBody))

	// 按行分割响应内容
	lines := bytes.Split(respBody, []byte("\n"))

	// 按系统和版本分组处理数据
	statsMap := make(map[string]*empyrean_lens.ApiPerformanceStats)

	// 处理每一行数据
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// 解析单行JSON响应
		var result struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
			Data      struct {
				Data    []interface{} `json:"data"`
				Columns []string      `json:"columns"`
			} `json:"data"`
		}
		if err := json.Unmarshal(line, &result); err != nil {
			hlog.CtxErrorf(ctx, "unmarshal response line error: %v, line: %s", err, string(line))
			continue
		}

		// 检查API响应状态
		if result.Code != "SUCCESS" {
			hlog.CtxErrorf(ctx, "神策API返回错误: code=%s, request_id=%s", result.Code, result.RequestID)
			continue
		}

		// 打印数据长度
		hlog.CtxInfof(ctx, "神策返回数据长度: %d", len(result.Data.Data))

		// 解析数据
		data := result.Data.Data

		// 验证数据长度
		if len(data) < 7 {
			hlog.CtxErrorf(ctx, "数据长度不足，期望至少7个元素，实际长度: %d", len(data))
			continue
		}

		// 辅助函数：将interface{}转换为float64
		toFloat64 := func(v interface{}) float64 {
			switch val := v.(type) {
			case string:
				f, _ := strconv.ParseFloat(val, 64)
				return f
			case float64:
				return val
			case int:
				return float64(val)
			default:
				return 0
			}
		}

		// 辅助函数：将interface{}转换为string
		toString := func(v interface{}) string {
			switch val := v.(type) {
			case string:
				return val
			case float64:
				return strconv.FormatFloat(val, 'f', -1, 64)
			case int:
				return strconv.Itoa(val)
			default:
				return ""
			}
		}

		// 根据新的响应格式解析数据
		// date := toString(data[0]) // 已不再使用
		avgDuration := toFloat64(data[1])
		version := toString(data[2])
		system := toString(data[3])
		totalRequests := int64(toFloat64(data[4]))
		duration := toFloat64(data[5])
		// timeStr := toString(data[6]) // 已不再使用

		// 使用date、system和version作为唯一标识
		key := fmt.Sprintf("%s_%s_%s", toString(data[0]), system, version)
		stats, exists := statsMap[key]
		if !exists {
			// 解析时间
			_, err := parseShenceTime(toString(data[6]))
			if err != nil {
				hlog.CtxErrorf(ctx, "parse time error: %v, time string: %s", err, toString(data[6]))
				continue
			}

			// 将时间转换为当天的 UTC 时间
			// 从日期字符串中提取日期部分（去掉时间部分）
			dateParts := strings.Split(toString(data[0]), " ")
			if len(dateParts) == 0 {
				hlog.CtxErrorf(ctx, "invalid date format: %s", toString(data[0]))
				continue
			}
			dateTime, err := time.Parse("2006-01-02", dateParts[0])
			if err != nil {
				hlog.CtxErrorf(ctx, "parse date error: %v, date: %s", err, toString(data[0]))
				continue
			}
			utcTime := time.Date(dateTime.Year(), dateTime.Month(), dateTime.Day(), 0, 0, 0, 0, time.UTC)

			stats = &empyrean_lens.ApiPerformanceStats{
				System:    system,
				Version:   version,
				Date:      toString(data[0]),
				CreatedAt: utcTime,
				UpdatedAt: time.Now(),
				Summary: struct {
					TotalDuration float64 `bson:"total_duration"`
					TotalRequests int64   `bson:"total_requests"`
					AvgDuration   float64 `bson:"avg_duration"`
				}{
					TotalDuration: avgDuration * float64(totalRequests),
					TotalRequests: totalRequests,
					AvgDuration:   avgDuration,
				},
				Buckets: make([]struct {
					TimeInterval string `bson:"time_interval"`
					RequestCount int64  `bson:"request_count"`
				}, 0),
			}
			statsMap[key] = stats
		} else {
			// 更新最新时间
			itemTime, err := parseShenceTime(toString(data[6]))
			if err != nil {
				hlog.CtxErrorf(ctx, "parse item time error: %v, time string: %s", err, toString(data[6]))
				continue
			}
			if itemTime.After(stats.CreatedAt) {
				// 将时间转换为当天的 UTC 时间
				// 从日期字符串中提取日期部分（去掉时间部分）
				dateParts := strings.Split(toString(data[0]), " ")
				if len(dateParts) == 0 {
					hlog.CtxErrorf(ctx, "invalid date format: %s", toString(data[0]))
					continue
				}
				dateTime, err := time.Parse("2006-01-02", dateParts[0])
				if err != nil {
					hlog.CtxErrorf(ctx, "parse date error: %v, date: %s", err, toString(data[0]))
					continue
				}
				utcTime := time.Date(dateTime.Year(), dateTime.Month(), dateTime.Day(), 0, 0, 0, 0, time.UTC)
				stats.CreatedAt = utcTime
				// 更新summary数据
				stats.Summary.TotalDuration = avgDuration * float64(totalRequests)
				stats.Summary.TotalRequests = totalRequests
				stats.Summary.AvgDuration = avgDuration
			}
		}

		// 计算时间区间
		timeInterval := getTimeInterval(duration)

		// 查找或创建对应的桶
		var bucket *struct {
			TimeInterval string `bson:"time_interval"`
			RequestCount int64  `bson:"request_count"`
		}

		for i := range stats.Buckets {
			if stats.Buckets[i].TimeInterval == timeInterval {
				bucket = &stats.Buckets[i]
				break
			}
		}

		if bucket == nil {
			stats.Buckets = append(stats.Buckets, struct {
				TimeInterval string `bson:"time_interval"`
				RequestCount int64  `bson:"request_count"`
			}{
				TimeInterval: timeInterval,
				RequestCount: 1,
			})
			bucket = &stats.Buckets[len(stats.Buckets)-1]
		} else {
			bucket.RequestCount++
		}
	}

	// 保存到MongoDB
	hlog.CtxInfof(ctx, "准备保存到MongoDB的数据条数: %d", len(statsMap))
	for _, stats := range statsMap {
		// 验证总请求数是否等于所有区间请求数的总和
		var totalBucketRequests int64
		for _, bucket := range stats.Buckets {
			totalBucketRequests += bucket.RequestCount
		}

		if totalBucketRequests != stats.Summary.TotalRequests {
			hlog.CtxWarnf(ctx, "总请求数(%d)与区间请求数总和(%d)不一致，系统: %s, 版本: %s",
				stats.Summary.TotalRequests, totalBucketRequests, stats.System, stats.Version)
		}

		if err := dao.UpsertStats(ctx, stats); err != nil {
			hlog.CtxErrorf(ctx, "upsert stats error: %v", err)
			return fmt.Errorf("upsert stats error: %v", err)
		}
	}

	return nil
}

// SyncApiPerformanceTrend 从神策同步API性能趋势数据到MongoDB
func SyncApiPerformanceTrend(ctx context.Context, startTime, endTime time.Time) error {
	// 确保集合存在
	dao := empyrean_lens.NewApiPerformanceTrendDao()
	if err := dao.EnsureCollection(ctx); err != nil {
		hlog.CtxErrorf(ctx, "ensure collection error: %v", err)
		return fmt.Errorf("ensure collection error: %v", err)
	}

	// 获取iOS和Android最新一条记录的时间
	iosLatest, iosErr := dao.GetLatestCreatedAtBySystem(ctx, "iOS")
	androidLatest, androidErr := dao.GetLatestCreatedAtBySystem(ctx, "Android")

	// 取最晚的那个
	var maxLatest time.Time
	if iosErr != nil && androidErr != nil {
		maxLatest = startTime
	} else if iosErr != nil {
		maxLatest = androidLatest
	} else if androidErr != nil {
		maxLatest = iosLatest
	} else if iosLatest.After(androidLatest) {
		maxLatest = iosLatest
	} else {
		maxLatest = androidLatest
	}

	// 将最新记录时间向上取整到最近的2小时时间点
	var firstEndTime time.Time
	if !maxLatest.IsZero() {
		hours := maxLatest.Hour()
		roundedHours := ((hours + 1) / 2) * 2
		if roundedHours == 24 {
			roundedHours = 0
			maxLatest = maxLatest.AddDate(0, 0, 1)
		}
		firstEndTime = time.Date(maxLatest.Year(), maxLatest.Month(), maxLatest.Day(), roundedHours, 0, 0, 0, maxLatest.Location())
	} else {
		firstEndTime = startTime.Add(2 * time.Hour)
	}

	// 计算时间间隔
	timeIntervals := make([]struct {
		Start time.Time
		End   time.Time
	}, 0)

	currentEnd := firstEndTime
	for currentEnd.Before(endTime) || currentEnd.Equal(endTime) {
		timeIntervals = append(timeIntervals, struct {
			Start time.Time
			End   time.Time
		}{
			Start: startTime,  // 始终从开始时间开始
			End:   currentEnd, // 结束时间逐渐增加
		})
		currentEnd = currentEnd.Add(2 * time.Hour)
	}

	// 按时间间隔处理数据
	for _, interval := range timeIntervals {
		// 构建SQL查询
		sql := fmt.Sprintf(`
			SELECT
				date,
				$os,
				perf_article_title_shown,
				time
			FROM
				events
			WHERE
				event = 'Performance_Metrics_PerformanceTime'
				AND date BETWEEN '%s' AND '%s'
				AND time BETWEEN '%s' AND '%s'
				AND $os IN ('iOS', 'Android')
			ORDER BY
				date ASC, $os ASC, time ASC
		`, interval.Start.Format("2006-01-02"), interval.End.Format("2006-01-02"),
			interval.Start.Format("2006-01-02 15:04:05.000"), interval.End.Format("2006-01-02 15:04:05.000"))

		// 构建请求体
		reqBody := map[string]interface{}{
			"sql":   sql,
			"limit": 10000000,
		}
		reqBodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			hlog.CtxErrorf(ctx, "marshal request body error: %v", err)
			return fmt.Errorf("marshal request body error: %v", err)
		}

		// 创建HTTP请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", shenceAPIURL, nil)
		if err != nil {
			hlog.CtxErrorf(ctx, "create request error: %v", err)
			return fmt.Errorf("create request error: %v", err)
		}
		httpReq.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("api-key", apiKey)
		httpReq.Header.Set("sensorsdata-project", project)

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			hlog.CtxErrorf(ctx, "send request error: %v", err)
			return fmt.Errorf("send request error: %v", err)
		}
		defer resp.Body.Close()

		// 读取响应
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			hlog.CtxErrorf(ctx, "read response error: %v", err)
			return fmt.Errorf("read response error: %v", err)
		}

		// 按行分割响应内容
		lines := bytes.Split(respBody, []byte("\n"))

		// 按日期和系统分组处理数据
		trendMap := make(map[string]*empyrean_lens.ApiPerformanceTrend)
		intervalCounts := make(map[string]map[string]int64) // 用于临时存储每个区间的请求数

		// 处理每一行数据
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}

			// 解析单行JSON响应
			var result struct {
				Code      string `json:"code"`
				RequestID string `json:"request_id"`
				Data      struct {
					Data    []interface{} `json:"data"`
					Columns []string      `json:"columns"`
				} `json:"data"`
			}
			if err := json.Unmarshal(line, &result); err != nil {
				hlog.CtxErrorf(ctx, "unmarshal response line error: %v, line: %s", err, string(line))
				continue
			}

			// 检查API响应状态
			if result.Code != "SUCCESS" {
				hlog.CtxErrorf(ctx, "神策API返回错误: code=%s, request_id=%s", result.Code, result.RequestID)
				continue
			}

			// 解析数据
			data := result.Data.Data

			// 验证数据长度
			if len(data) < 4 {
				hlog.CtxErrorf(ctx, "数据长度不足，期望至少4个元素，实际长度: %d", len(data))
				continue
			}

			// 辅助函数：将interface{}转换为string
			toString := func(v interface{}) string {
				switch val := v.(type) {
				case string:
					return val
				case float64:
					return strconv.FormatFloat(val, 'f', -1, 64)
				case int:
					return strconv.Itoa(val)
				default:
					return ""
				}
			}

			// 辅助函数：将interface{}转换为float64
			toFloat64 := func(v interface{}) float64 {
				switch val := v.(type) {
				case string:
					f, _ := strconv.ParseFloat(val, 64)
					return f
				case float64:
					return val
				case int:
					return float64(val)
				default:
					return 0
				}
			}

			// 解析数据
			// date := toString(data[0]) // 已不再使用
			system := toString(data[1])
			duration := toFloat64(data[2])
			// timeStr := toString(data[3]) // 已不再使用

			// 使用date和system作为唯一标识
			key := fmt.Sprintf("%s_%s", toString(data[0]), system)
			trend, exists := trendMap[key]
			if !exists {
				// 解析时间
				_, err := parseShenceTime(toString(data[3]))
				if err != nil {
					hlog.CtxErrorf(ctx, "parse time error: %v, time string: %s", err, toString(data[3]))
					continue
				}

				// 使用time_point的日期部分作为date
				timePoint := interval.End.Format("2006-01-02 15:04:05")
				recordDate := interval.End.Format("2006-01-02")

				trend = &empyrean_lens.ApiPerformanceTrend{
					Date:      recordDate,
					System:    system,
					TimePoint: timePoint,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Buckets: make([]struct {
						TimeInterval string  `bson:"time_interval"`
						Percentage   float64 `bson:"percentage"`
					}, 0),
				}
				trendMap[key] = trend
				intervalCounts[key] = make(map[string]int64)
			}

			// 计算时间区间
			timeInterval := getTimeInterval(duration)
			intervalCounts[key][timeInterval]++
		}

		// 计算每个时间点的百分比
		for key, trend := range trendMap {
			var totalRequests int64
			for _, count := range intervalCounts[key] {
				totalRequests += count
			}

			if totalRequests > 0 {
				for interval, count := range intervalCounts[key] {
					percentage := float64(count) / float64(totalRequests) * 100
					trend.Buckets = append(trend.Buckets, struct {
						TimeInterval string  `bson:"time_interval"`
						Percentage   float64 `bson:"percentage"`
					}{
						TimeInterval: interval,
						Percentage:   percentage,
					})
				}
			}

			// 保存到MongoDB
			if err := dao.UpsertTrend(ctx, trend); err != nil {
				hlog.CtxErrorf(ctx, "upsert trend error: %v", err)
				return fmt.Errorf("upsert trend error: %v", err)
			}
		}
	}

	return nil
}

// SyncApiPerformanceVersionTrend 从神策同步所有版本API性能趋势数据到MongoDB
func SyncApiPerformanceVersionTrend(ctx context.Context, startTime, endTime time.Time) error {
	// 确保集合存在
	dao := empyrean_lens.NewApiPerformanceLatestVersionTrendDao()
	if err := dao.EnsureCollection(ctx); err != nil {
		hlog.CtxErrorf(ctx, "ensure collection error: %v", err)
		return fmt.Errorf("ensure collection error: %v", err)
	}

	// 获取iOS和Android最新一条记录的时间
	iosLatest, iosErr := dao.GetLatestCreatedAtBySystem(ctx, "iOS")
	androidLatest, androidErr := dao.GetLatestCreatedAtBySystem(ctx, "Android")

	// 取最晚的那个
	var maxLatest time.Time
	if iosErr != nil && androidErr != nil {
		maxLatest = startTime
	} else if iosErr != nil {
		maxLatest = androidLatest
	} else if androidErr != nil {
		maxLatest = iosLatest
	} else if iosLatest.After(androidLatest) {
		maxLatest = iosLatest
	} else {
		maxLatest = androidLatest
	}

	// 将最新记录时间向上取整到最近的2小时时间点
	var firstEndTime time.Time
	if !maxLatest.IsZero() {
		hours := maxLatest.Hour()
		roundedHours := ((hours + 1) / 2) * 2
		if roundedHours == 24 {
			roundedHours = 0
			maxLatest = maxLatest.AddDate(0, 0, 1)
		}
		firstEndTime = time.Date(maxLatest.Year(), maxLatest.Month(), maxLatest.Day(), roundedHours, 0, 0, 0, maxLatest.Location())
	} else {
		firstEndTime = startTime.Add(2 * time.Hour)
	}

	hlog.CtxInfof(ctx, "Start time: %v, First end time: %v, End time: %v", startTime, firstEndTime, endTime)

	// 计算时间间隔
	timeIntervals := make([]struct {
		Start time.Time
		End   time.Time
	}, 0)

	currentEnd := firstEndTime
	for currentEnd.Before(endTime) || currentEnd.Equal(endTime) {
		timeIntervals = append(timeIntervals, struct {
			Start time.Time
			End   time.Time
		}{
			Start: startTime,  // 始终从开始时间开始
			End:   currentEnd, // 结束时间逐渐增加
		})
		currentEnd = currentEnd.Add(2 * time.Hour)
	}

	// 按时间间隔处理数据
	for _, interval := range timeIntervals {
		// 构建SQL查询
		sql := fmt.Sprintf(`
			SELECT
				date,
				$os,
				$os_version,
				perf_article_title_shown,
				time
			FROM
				events
			WHERE
				event = 'Performance_Metrics_PerformanceTime'
				AND date BETWEEN '%s' AND '%s'
				AND time BETWEEN '%s' AND '%s'
				AND $os IN ('iOS', 'Android')
			ORDER BY
				date ASC, $os ASC, $os_version ASC, time ASC
		`, startTime.Format("2006-01-02"), interval.End.Format("2006-01-02"),
			startTime.Format("2006-01-02 15:04:05.000"), interval.End.Format("2006-01-02 15:04:05.000"))

		// 构建请求体
		reqBody := map[string]interface{}{
			"sql":   sql,
			"limit": 10000000,
		}
		reqBodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			hlog.CtxErrorf(ctx, "marshal request body error: %v", err)
			return fmt.Errorf("marshal request body error: %v", err)
		}

		// 创建HTTP请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", shenceAPIURL, nil)
		if err != nil {
			hlog.CtxErrorf(ctx, "create request error: %v", err)
			return fmt.Errorf("create request error: %v", err)
		}
		httpReq.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("api-key", apiKey)
		httpReq.Header.Set("sensorsdata-project", project)

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			hlog.CtxErrorf(ctx, "send request error: %v", err)
			return fmt.Errorf("send request error: %v", err)
		}
		defer resp.Body.Close()

		// 读取响应
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			hlog.CtxErrorf(ctx, "read response error: %v", err)
			return fmt.Errorf("read response error: %v", err)
		}

		// 按行分割响应内容
		lines := bytes.Split(respBody, []byte("\n"))

		// 按日期和系统分组处理数据
		trendMap := make(map[string]*empyrean_lens.ApiPerformanceLatestVersionTrend)
		intervalCounts := make(map[string]map[string]int64) // 用于临时存储每个区间的请求数

		// 处理每一行数据
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}

			// 解析单行JSON响应
			var result struct {
				Code      string `json:"code"`
				RequestID string `json:"request_id"`
				Data      struct {
					Data    []interface{} `json:"data"`
					Columns []string      `json:"columns"`
				} `json:"data"`
			}
			if err := json.Unmarshal(line, &result); err != nil {
				hlog.CtxErrorf(ctx, "unmarshal response line error: %v, line: %s", err, string(line))
				continue
			}

			// 检查API响应状态
			if result.Code != "SUCCESS" {
				hlog.CtxErrorf(ctx, "神策API返回错误: code=%s, request_id=%s", result.Code, result.RequestID)
				continue
			}

			// 解析数据
			data := result.Data.Data
			if len(data) < 5 {
				hlog.CtxErrorf(ctx, "数据长度不足，期望至少5个元素，实际长度: %d", len(data))
				continue
			}

			// 辅助函数：将interface{}转换为string
			toString := func(v interface{}) string {
				switch val := v.(type) {
				case string:
					return val
				case float64:
					return strconv.FormatFloat(val, 'f', -1, 64)
				case int:
					return strconv.Itoa(val)
				default:
					return ""
				}
			}

			// 辅助函数：将interface{}转换为float64
			toFloat64 := func(v interface{}) float64 {
				switch val := v.(type) {
				case string:
					f, _ := strconv.ParseFloat(val, 64)
					return f
				case float64:
					return val
				case int:
					return float64(val)
				default:
					return 0
				}
			}

			// 解析数据
			// date := toString(data[0]) // 已不再使用
			system := toString(data[2])
			version := toString(data[1])
			duration := toFloat64(data[3])
			// timeStr := toString(data[4]) // 已不再使用

			// 以recordDate、system、version、timePoint唯一标识一条入库记录
			timePoint := interval.End.Format("2006-01-02 15:04:05")
			recordDate := interval.End.Format("2006-01-02")
			key := fmt.Sprintf("%s_%s_%s_%s", recordDate, system, version, timePoint)
			trend, exists := trendMap[key]
			if !exists {
				trend = &empyrean_lens.ApiPerformanceLatestVersionTrend{
					Date:      recordDate,
					System:    system,
					Version:   version,
					TimePoint: timePoint,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Buckets: make([]struct {
						TimeInterval string  `bson:"time_interval"`
						Percentage   float64 `bson:"percentage"`
					}, 0),
				}
				trendMap[key] = trend
				intervalCounts[key] = make(map[string]int64)
			}
			// 计算时间区间
			timeInterval := getTimeInterval(duration)
			intervalCounts[key][timeInterval]++
		}

		// 计算每个时间点的百分比
		for key, trend := range trendMap {
			var totalRequests int64
			for _, count := range intervalCounts[key] {
				totalRequests += count
			}

			if totalRequests > 0 {
				// 清空之前的buckets，因为我们要重新计算累积数据
				trend.Buckets = make([]struct {
					TimeInterval string  `bson:"time_interval"`
					Percentage   float64 `bson:"percentage"`
				}, 0)

				// 按时间区间排序
				intervals := make([]string, 0)
				for interval := range intervalCounts[key] {
					intervals = append(intervals, interval)
				}
				sort.Strings(intervals)

				// 计算每个区间的百分比
				for _, interval := range intervals {
					count := intervalCounts[key][interval]
					percentage := float64(count) / float64(totalRequests) * 100
					trend.Buckets = append(trend.Buckets, struct {
						TimeInterval string  `bson:"time_interval"`
						Percentage   float64 `bson:"percentage"`
					}{
						TimeInterval: interval,
						Percentage:   percentage,
					})
				}
			}

			// 保存到MongoDB
			if err := dao.UpsertTrend(ctx, trend); err != nil {
				hlog.CtxErrorf(ctx, "upsert trend error: %v", err)
				return fmt.Errorf("upsert trend error: %v", err)
			}
		}

		// 打印每个时间点的版本数量
		versionCount := make(map[string]int)
		for _, trend := range trendMap {
			timePointKey := fmt.Sprintf("%s_%s", trend.Date, trend.System)
			versionCount[timePointKey]++
		}
		for timePointKey, count := range versionCount {
			hlog.CtxInfof(ctx, "时间点 %s 的版本数量: %d", timePointKey, count)
		}
	}

	return nil
}
