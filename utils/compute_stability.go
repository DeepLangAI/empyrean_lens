package utils

import (
	"empyrean_lens/consts"
	"fmt"
	"math"
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

	return int(log_norm(data[consts.ERROR_RATE_PARAMETER], 0, 0)*consts.ERROR_WEIGHT +
		log_norm(data[consts.PROBE_ERROR_RATE_PARAMETER], 0, 0)*consts.PROBE_WEIGHT +
		log_norm(data[consts.SLOW_SEARCH_RATE_PARAMETER], 0, 0)*consts.SLOW_SEARCH_WEIGHT)
}

func ComputeStability(data map[string]float64) int {
	return compute_satability_score(data)
}

func ComputeRevent(data map[string]int) {

}
