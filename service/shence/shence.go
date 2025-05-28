package shence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"empyrean_lens/dal/mongo/empyrean_lens"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

const (
	shenceAPIURL = "https://shenyankeji.cloud.sensorsdata.cn/api/v3/analytics/v1/model/sql/query"
	apiKey       = "#K-EkE7btXHodiCAwtJqyD3PPzDhrk0jYV5"
	project      = "production"
)

// getTimeInterval 根据总耗时确定时间区间
func getTimeInterval(totalDuration float64) string {
	switch {
	case totalDuration >= 0 && totalDuration < 2000:
		return "0~2000"
	case totalDuration >= 2000 && totalDuration < 3000:
		return "2000~3000"
	case totalDuration >= 3000 && totalDuration < 5000:
		return "3000~5000"
	case totalDuration >= 5000:
		return ">5000"
	default:
		return "0~2000" // 处理负数等异常情况
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

		// 辅助函数：解析神策时间格式
		parseShenceTime := func(timeStr string) (time.Time, error) {
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

		// 根据新的响应格式解析数据
		date := toString(data[0])
		avgDuration := toFloat64(data[1])
		version := toString(data[2])
		system := toString(data[3])
		totalRequests := int64(toFloat64(data[4]))
		duration := toFloat64(data[5])
		timeStr := toString(data[6])

		// 使用date、system和version作为唯一标识
		key := fmt.Sprintf("%s_%s_%s", date, system, version)
		stats, exists := statsMap[key]
		if !exists {
			// 解析时间
			latestTime, err := parseShenceTime(timeStr)
			if err != nil {
				hlog.CtxErrorf(ctx, "parse time error: %v, time string: %s", err, timeStr)
				continue
			}

			stats = &empyrean_lens.ApiPerformanceStats{
				System:    system,
				Version:   version,
				Date:      date,
				CreatedAt: latestTime,
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
			itemTime, err := parseShenceTime(timeStr)
			if err != nil {
				hlog.CtxErrorf(ctx, "parse item time error: %v, time string: %s", err, timeStr)
				continue
			}
			if itemTime.After(stats.CreatedAt) {
				stats.CreatedAt = itemTime
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
