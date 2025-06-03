package model

import (
	"time"
)

// ApiPerformanceStats 存储API性能统计数据
type ApiPerformanceStats struct {
	ID         string    `bson:"_id,omitempty"`
	System     string    `bson:"system"`
	Version    string    `bson:"version"`
	CreateTime time.Time `bson:"create_time"`
	UpdateTime time.Time `bson:"update_time"`

	Summary struct {
		TotalDuration float64 `bson:"total_duration"`
		TotalRequests int64   `bson:"total_requests"`
		AvgDuration   float64 `bson:"avg_duration"`
	} `bson:"summary"`

	Buckets []TimeIntervalBucket `bson:"buckets"`
}

// TimeIntervalBucket 时间区间桶
type TimeIntervalBucket struct {
	TimeInterval   string `bson:"time_interval"`
	RequestCount   int64  `bson:"request_count"`
	PhaseBreakdown struct {
		Stage1CostTime   float64 `bson:"stage1_cost_time"`
		Stage1Proportion float64 `bson:"stage1_proportion"`
		Stage2CostTime   float64 `bson:"stage2_cost_time"`
		Stage2Proportion float64 `bson:"stage2_proportion"`
		Stage3CostTime   float64 `bson:"stage3_cost_time"`
		Stage3Proportion float64 `bson:"stage3_proportion"`
		Stage4CostTime   float64 `bson:"stage4_cost_time"`
		Stage4Proportion float64 `bson:"stage4_proportion"`
	} `bson:"phase_breakdown"`
}

// SensorsDataResponse 神策数据响应
type SensorsDataResponse struct {
	Data []SensorsDataItem `json:"data"`
}

// SensorsDataItem 神策数据项
type SensorsDataItem struct {
	Date                  string  `json:"date"`
	OS                    string  `json:"$os"`
	OSVersion             string  `json:"$os_version"`
	AvgDuration           float64 `json:"平均耗时"`
	TotalRequests         int64   `json:"总请求数"`
	PerfArticleTitleShown float64 `json:"perf_article_title_shown"`
	PerfDCL               float64 `json:"perf_dcl"`
	PerfL                 float64 `json:"perf_l"`
	PerfFCP               float64 `json:"perf_fcp"`
	Time                  string  `json:"time"`
}
