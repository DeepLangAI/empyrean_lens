package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

func SaveDailyAutomationStatsByDateRange(ctx context.Context, dateStr time.Time) error {
	// 获取最新的统计记录
	latestStats, err := empyrean_lens.NewDailyAutomationStatsDao().GetLatestStats(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to get latest stats: %v", err)
		return err
	}

	inputDate := dateStr

	var beginTime time.Time
	// 如果有最新记录，使用最新记录的日期作为开始时间
	if latestStats != nil {
		latestDate, err := time.Parse("2006-01-02", latestStats.Date)
		if err != nil {
			hlog.CtxErrorf(ctx, "Failed to parse latest date: %v", err)
			return err
		}

		// 设置开始时间为最新记录日期的0点
		beginTime = latestDate

		// 如果最新记录的日期就是要插入的日期，直接返回
		if latestDate.Year() == inputDate.Year() &&
			latestDate.Month() == inputDate.Month() &&
			latestDate.Day() == inputDate.Day() {
			hlog.CtxInfof(ctx, "Data for date %s already exists, skipping", dateStr)
			return nil
		}
	} else {
		// 如果没有最新记录，使用输入日期作为开始时间
		beginTime = inputDate
	}

	// 设置结束时间为当前时间
	endTime := time.Now()

	// 获取当时间段的统计数据
	stats, err := GetDailyModelAutomationByTime(ctx, beginTime, endTime, 0, 1)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to get daily automation stats: %v", err)
		return err
	}

	var statsModels []empyrean_lens.DailyAutomationStatsModel

	for _, stat := range stats {
		statsModel := empyrean_lens.DailyAutomationStatsModel{
			Date:           stat.Date,
			SingleFailNum:  stat.SingleFailNum,
			SingleTotalNum: stat.SingleTotalNum,
			SingleDocNum:   stat.SingleDocNum,
			WebFailNum:     stat.WebFailNum,
			WebTotalNum:    stat.WebTotalNum,
			WebDocNum:      stat.WebDocNum,
			MultiFailNum:   stat.MultiFailNum,
			MultiTotalNum:  stat.MultiTotalNum,
			MultiDocNum:    stat.MultiDocNum,
			CreateTime:     stat.CreateTime,
			UpdateTime:     stat.CreateTime,
		}
		statsModels = append(statsModels, statsModel)
	}

	err = empyrean_lens.NewDailyAutomationStatsDao().SaveBatch(ctx, statsModels)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to save daily automation stats: %v", err)
		return err
	}
	hlog.CtxInfof(ctx, "Successfully saved daily automation stats for date: %s", dateStr)
	return nil
}

func SaveDailyAutomationStatsByDate(ctx context.Context, dateStr string) error {
	// 设置时间范围为指定日期的 00:00:00 到 23:59:59
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		hlog.CtxErrorf(ctx, "Invalid date format: %v", err)
		return err
	}

	beginTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, time.Local)

	// 获取当天的统计数据
	stats, err := GetDailyModelAutomationByTime(ctx, beginTime, endTime, 0, 1)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to get daily automation stats: %v", err)
		return err
	}

	if len(stats) > 0 {
		statsModel := empyrean_lens.DailyAutomationStatsModel{
			Date:           dateStr,
			SingleFailNum:  stats[0].SingleFailNum,
			SingleTotalNum: stats[0].SingleTotalNum,
			SingleDocNum:   stats[0].SingleDocNum,
			WebFailNum:     stats[0].WebFailNum,
			WebTotalNum:    stats[0].WebTotalNum,
			WebDocNum:      stats[0].WebDocNum,
			MultiFailNum:   stats[0].MultiFailNum,
			MultiTotalNum:  stats[0].MultiTotalNum,
			MultiDocNum:    stats[0].MultiDocNum,
			CreateTime:     stats[0].CreateTime,
			UpdateTime:     stats[0].CreateTime,
		}

		err = empyrean_lens.NewDailyAutomationStatsDao().SaveBatch(ctx, []empyrean_lens.DailyAutomationStatsModel{statsModel})
		if err != nil {
			hlog.CtxErrorf(ctx, "Failed to save daily automation stats: %v", err)
			return err
		}
		hlog.CtxInfof(ctx, "Successfully saved daily automation stats for date: %s", dateStr)
	}

	return nil
}

// SaveOnceDailyAutomationStats 一次性迁移所有历史数据（按日期倒序）
func SaveOnceDailyAutomationStats(ctx context.Context) error {
	// 获取最早的记录时间作为结束日期
	startDate, err := empyrean_lens.NewModelCaseResultDao().GetEarliestRecordDate(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "Failed to get earliest record date: %v", err)
		return err
	}

	// 从当前时间开始，倒序处理到最早的记录时间
	currentTime := time.Now()
	endTime := startDate.Truncate(24 * time.Hour) // 将时间调整到当天的0点

	for !currentTime.Before(endTime) {
		dateStr := currentTime.Format("2006-01-02")
		if err := SaveDailyAutomationStatsByDate(ctx, dateStr); err != nil {
			hlog.CtxErrorf(ctx, "Failed to save stats for date %s: %v", dateStr, err)
			return err
		}
		hlog.CtxInfof(ctx, "Successfully migrated data for date: %s", dateStr)

		// 向前移动一天
		currentTime = currentTime.AddDate(0, 0, -1)
	}

	return nil
}

// SaveYesterdayAutomationStats
func SaveYesterdayAutomationStats(ctx context.Context) error {
	yesterday := time.Now().AddDate(0, 0, -1)
	//dateStr := yesterday.Format("2006-01-02")
	//
	//hlog.CtxInfof(ctx, "Yesterday: %v", yesterday)
	//hlog.CtxInfof(ctx, "Date string to process: %v", dateStr)
	//bufio.NewReader(os.Stdin).ReadBytes('\n')
	return SaveDailyAutomationStatsByDateRange(ctx, yesterday)
}

// SaveMissingDaysAutomationStats 保存最近几天可能漏掉的数据
//func SaveMissingDaysAutomationStats(ctx context.Context) error {
//	// 获取最近一条记录的日期
//	lastStats, err := empyrean_lens.NewDailyAutomationStatsDao().GetLatestStats(ctx)
//	if err != nil {
//		hlog.CtxErrorf(ctx, "Failed to get latest stats: %v", err)
//		return err
//	}
//
//	var lastDate time.Time
//	if lastStats != nil {
//		// 如果有记录，从最后一条记录的第二天开始
//		lastDate, err = time.Parse("2006-01-02", lastStats.Date)
//		if err != nil {
//			return err
//		}
//		lastDate = lastDate.AddDate(0, 0, 1)
//	} else {
//		// 如果没有记录，默认从7天前开始
//		lastDate = time.Now().AddDate(0, 0, -7)
//	}
//
//	// 处理到昨天为止的所有数据
//	yesterday := time.Now().AddDate(0, 0, -1)
//	for !lastDate.After(yesterday) {
//		dateStr := lastDate.Format("2006-01-02")
//		if err := SaveDailyAutomationStatsByDate(ctx, dateStr); err != nil {
//			hlog.CtxErrorf(ctx, "Failed to save stats for date %s: %v", dateStr, err)
//			return err
//		}
//		hlog.CtxInfof(ctx, "Successfully saved missing data for date: %s", dateStr)
//		lastDate = lastDate.AddDate(0, 0, 1)
//	}
//
//	return nil
//}
