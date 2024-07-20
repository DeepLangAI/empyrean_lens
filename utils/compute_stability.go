package utils

import (
	"empyrean_lens/consts"
	"fmt"
	"math"
	"sort"
	"time"
)

// 指标归一化处理
func log_norm(data float64, minVal float64, maxVal float64) float64 {
	//设置默认值
	if minVal == 0 {
		minVal = 0.0001
	}
	if maxVal == 0 {
		maxVal = 1.0
	}

	if data > maxVal {
		return 1
	}

	if data < minVal {
		return 0
	}

	// Logarithmic transformation
	logData := math.Log(data)

	// Logarithm of minVal and maxVal
	logMin := math.Log(minVal)
	logMax := math.Log(maxVal)

	// Normalize to the range [0, 1]
	normalizedData := (logData - logMin) / (logMax - logMin)

	return normalizedData
}

// 计算稳定性得分
func compute_satability_score(data map[string]float64) int {
	if _, ok := data[consts.ERROR_RATE_PARAMETER]; !ok {
		fmt.Println("data error, no key = ", consts.ERROR_RATE_PARAMETER)
		return 0
	}
	if _, ok := data[consts.PROBE_ERROR_RATE_PARAMETER]; !ok {
		fmt.Println("data error, no key = ", consts.PROBE_ERROR_RATE_PARAMETER)
		return 0
	}
	if _, ok := data[consts.SLOW_SEARCH_RATE_PARAMETER]; !ok {
		fmt.Println("data error, no key = ", consts.SLOW_SEARCH_RATE_PARAMETER)
		return 0
	}

	return int((log_norm(data[consts.ERROR_RATE_PARAMETER], 0, 0)*consts.ERROR_WEIGHT +
		log_norm(data[consts.PROBE_ERROR_RATE_PARAMETER], 0, 0)*consts.PROBE_WEIGHT +
		log_norm(data[consts.SLOW_SEARCH_RATE_PARAMETER], 0, 0)*consts.SLOW_SEARCH_WEIGHT) * 100)
}

func ComputeStability(data map[string]float64) int {
	return compute_satability_score(data)
}

func ComputeRevent(scores map[string]int) map[string][3]float64 {
	result := make(map[string][3]float64)
	// 解析输入日期，并创建辅助 map
	parsedDates := make(map[string]time.Time)
	for dateStr := range scores {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			fmt.Println("Error parsing date:", err)
			continue
		}
		parsedDates[dateStr] = date
	}

	for dateStr, score := range scores {
		date := parsedDates[dateStr]

		// 计算前一天和前一周的日期
		prevDay := date.AddDate(0, 0, -1).Format("2006-01-02")
		prevWeek := date.AddDate(0, 0, -7).Format("2006-01-02")

		// 初始化日环比和周同比
		dayOverDay := 0.0
		weekOverWeek := 0.0

		// 计算日环比
		if prevScore, exists := scores[prevDay]; exists {
			dayOverDay = float64(score-prevScore) / float64(prevScore)
		}

		// 计算周同比
		if prevWeekScore, exists := scores[prevWeek]; exists {
			weekOverWeek = float64(score-prevWeekScore) / float64(prevWeekScore)
		}

		// 填充结果
		result[dateStr] = [3]float64{float64(score), dayOverDay, weekOverWeek}
	}

	// 按日期排序
	sortedResult := make(map[string][3]float64)
	var dates []string
	for date := range result {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i] < dates[j]
	})

	for _, date := range dates {
		sortedResult[date] = result[date]
	}

	return sortedResult
}
