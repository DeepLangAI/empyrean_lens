package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CalculateAndSaveDailyStats 计算并保存每日统计数据
func CalculateAndSaveDailyStats(ctx context.Context, date time.Time) error {
	// 统一使用中国时区
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	date = date.In(loc)

	// 获取当天的开始时间和结束时间
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 59, loc)

	// 修改聚合管道
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_time": bson.M{
				"$gte": startOfDay,
				"$lt":  endOfDay,
			},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"api_name": "$api_name",
				"date": bson.M{
					"$dateToString": bson.M{
						"format":   "%Y-%m-%d",
						"date":     "$created_time",
						"timezone": "Asia/Shanghai",
					},
				},
			},
			"total": bson.M{"$sum": 1},
			"success": bson.M{
				"$sum": bson.M{
					"$cond": []interface{}{
						bson.M{"$eq": []interface{}{"$response_content.status_code", 200}},
						1,
						0,
					},
				},
			},
		}}},
	}

	// 执行聚合查询
	cursor, err := empyrean_lens.NewApiTestDetailDao().Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	// 创建一个 map 来存储所有 API 类型的统计数据
	statsMap := make(map[string][]empyrean_lens.ApiStats)

	// 使用统一的日期格式
	dateStr := date.Format("2006-01-02")

	// 初始化所有 API 类型的统计数据
	for _, apiType := range []empyrean_lens.ApiType{
		empyrean_lens.ApiTypeUploadPDF,
		empyrean_lens.ApiTypeUploadPDFParsing,
		empyrean_lens.ApiTypeUploadURL,
		empyrean_lens.ApiTypeEduInput,
		empyrean_lens.ApiTypeEduOutput,
		empyrean_lens.ApiTypeEduTree,
		empyrean_lens.ApiTypeSingleDocOutline,
		empyrean_lens.ApiTypePDFParsing,
		empyrean_lens.ApiTypeTextParse,
		empyrean_lens.ApiTypeWCD,
		empyrean_lens.ApiTypeCrawler,
		empyrean_lens.ApiTypeCrawlerImg,
		empyrean_lens.ApiTypeNovelFormGenerate,
		empyrean_lens.ApiTypeNovelFormGet,
		empyrean_lens.ApiTypeMasterThemeURL,
		empyrean_lens.ApiTypeSingleDocURL,
		empyrean_lens.ApiTypeKeyOpinion,
	} {
		statsMap[dateStr] = append(statsMap[dateStr], empyrean_lens.ApiStats{
			ApiType: string(apiType),
			Total:   0,
			Success: 0,
			Rate:    0,
		})
	}

	// 处理聚合结果
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				ApiName string `bson:"api_name"`
				Date    string `bson:"date"`
			} `bson:"_id"`
			Total   int `bson:"total"`
			Success int `bson:"success"`
		}
		if err := cursor.Decode(&result); err != nil {
			return err
		}

		rate := float64(0)
		if result.Total > 0 {
			rate = float64(result.Success) / float64(result.Total)
		}

		// 更新对应日期的统计数据
		dateStats := statsMap[result.ID.Date]
		for i := range dateStats {
			if dateStats[i].ApiType == result.ID.ApiName {
				dateStats[i].Total = result.Total
				dateStats[i].Success = result.Success
				dateStats[i].Rate = rate
				break
			}
		}
	}

	// 转换 map 为 slice
	stats := make([]empyrean_lens.ApiStatsDailyModel, 0, len(statsMap))
	for date, apiStats := range statsMap {
		parsedDate, _ := time.ParseInLocation("2006-01-02", date, loc)
		stats = append(stats, empyrean_lens.ApiStatsDailyModel{
			Date:        date,
			ApiStats:    apiStats,
			CreatedTime: parsedDate,
		})
	}

	// 保存统计结果
	for _, stat := range stats {
		if err := empyrean_lens.NewApiStatsDailyDao().UpsertDailyStats(ctx, stat.ApiStats, stat.CreatedTime); err != nil {
			return err
		}
	}

	return nil
}

// GetApiStatsDaily 获取API每日统计数据
func GetApiStatsDaily(ctx context.Context, startDate, endDate time.Time) ([]empyrean_lens.ApiStatsDailyModel, error) {
	// 如果没有提供日期范围，则获取所有数据
	if startDate.IsZero() && endDate.IsZero() {
		return empyrean_lens.NewApiStatsDailyDao().GetDailyStats(ctx, time.Time{}, time.Time{})
	}
	return empyrean_lens.NewApiStatsDailyDao().GetDailyStats(ctx, startDate, endDate)
}

// SaveOnceDailyApiStats 一次性补充所有历史数据（按日期倒序）
func SaveOnceDailyApiStats(ctx context.Context) error {
	// 获取 api_test_detail 最早的记录时间
	apiTestStartDate, err := empyrean_lens.NewApiTestDetailDao().GetEarliestRecordDate(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to get earliest record date from ApiTestDetail: %v", err)
		return err
	}

	// 获取 api_stats_daily 最新的记录时间
	latestStats, err := empyrean_lens.NewApiStatsDailyDao().GetLatestStats(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to get latest stats date: %v", err)
		return err
	}

	var startDate time.Time
	if latestStats != nil {
		statsDate, err := time.Parse("2006-01-02", latestStats.Date)
		if err != nil {
			hlog.CtxErrorf(ctx, "Failed to parse latest stats date: %v", err)
			return err
		}
		// 比较两个时间，取较新的那个
		if statsDate.After(apiTestStartDate) {
			startDate = statsDate
		} else {
			startDate = apiTestStartDate
		}
	} else {
		startDate = apiTestStartDate
	}

	// 从当前时间开始，倒序处理到最早的记录时间
	currentTime := time.Now()
	endTime := startDate.Truncate(24 * time.Hour) // 将时间调整到当天的0点

	for !currentTime.Before(endTime) {
		dateStr := currentTime.Format("2006-01-02")

		// 先查询该日期是否已有记录
		exists, err := empyrean_lens.NewApiStatsDailyDao().ExistsByDate(ctx, dateStr)
		if err != nil {
			hlog.CtxErrorf(ctx, "Failed to check stats existence for date %s: %v", dateStr, err)
			return err
		}

		// 如果记录已存在，跳过该日期
		if exists {
			hlog.CtxInfof(ctx, "Stats already exist for date: %s, skipping...", dateStr)
			currentTime = currentTime.AddDate(0, 0, -1)
			continue
		}

		// 如果记录不存在，则保存新数据
		if err := CalculateAndSaveDailyStats(ctx, currentTime); err != nil {
			hlog.CtxErrorf(ctx, "Failed to save stats for date %s: %v", dateStr, err)
			return err
		}
		hlog.CtxInfof(ctx, "Successfully migrated data for date: %s", dateStr)

		// 向前移动一天
		currentTime = currentTime.AddDate(0, 0, -1)
	}

	return nil
}
