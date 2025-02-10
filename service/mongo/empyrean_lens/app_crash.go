package empyrean_lens

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"time"
)

type AppCrushRespData struct {
	Date                         string  `json:"date"`
	IosCrashCnt                  int32   `json:"ios_crash_cnt"`
	AndroidCrashCnt              int32   `json:"android_crash_cnt"`
	IosCrashUserCnt              int32   `json:"ios_crash_user_cnt"`
	AndroidCrashUserCnt          int32   `json:"android_crash_user_cnt"`
	IosLaunchCnt                 int32   `json:"ios_launch_cnt"`
	AndroidLaunchCnt             int32   `json:"android_launch_cnt"`
	ActiveUserCnt                int32   `json:"active_user_cnt"`
	IosCrashRate                 float32 `json:"ios_crash_rate"`
	AndroidCrashRate             float32 `json:"android_crash_rate"`
	IosCrashRateDayOverDay       float32 `json:"ios_crash_rate_day_over_day"`
	AndroidCrashRateDayOverDay   float32 `json:"android_crash_rate_day_over_day"`
	IosCrashRateWeekOverWeek     float32 `json:"ios_crash_rate_week_over_week"`
	AndroidCrashRateWeekOverWeek float32 `json:"android_crash_rate_week_over_week"`
}

func GetDailyAppCrashByTime(ctx context.Context, beginTime, endTime time.Time) ([]AppCrushRespData, error) {
	appCrashDetails, e := empyrean_lens.NewAppCrashModelDao().GetAppCrashInfoByTime(ctx, beginTime, endTime)
	if e != nil {
		return nil, e
	}
	// 按照 date 分组
	dateMap := make(map[string][]empyrean_lens.AppCrashModel)
	dateMap2 := make(map[string]AppCrushRespData)
	for _, detail := range appCrashDetails {
		date := detail.Time.Local().Format("2006-01-02")
		dateMap[date] = append(dateMap[date], detail)
	}
	var respDetails []AppCrushRespData
	for date, details := range dateMap {
		respDetail := AppCrushRespData{
			Date: date,
		}
		for _, detail := range details {
			if detail.PlatformType == consts.Platform_IOS {
				respDetail.IosCrashCnt += detail.ErrorCount
				respDetail.IosCrashUserCnt += detail.AffectedUserCount
				respDetail.IosLaunchCnt += detail.LaunchCount
			} else if detail.PlatformType == consts.Platform_Android {
				respDetail.AndroidCrashCnt += detail.ErrorCount
				respDetail.AndroidCrashUserCnt += detail.AffectedUserCount
				respDetail.AndroidLaunchCnt += detail.LaunchCount
			}
			respDetail.ActiveUserCnt += detail.ActiveUserCount
		}
		respDetail.IosCrashRate = utils.Div(float32(respDetail.IosCrashCnt), float32(respDetail.IosLaunchCnt)) * 100
		respDetail.AndroidCrashRate = utils.Div(float32(respDetail.AndroidCrashCnt), float32(respDetail.AndroidLaunchCnt)) * 100
		dateMap2[date] = respDetail
	}
	for date, detail := range dateMap2 {
		today, err := time.ParseInLocation("2006-01-02", date, time.Local)
		if err != nil {
			hlog.CtxErrorf(ctx, "parse date error in GetDailyAppCrashByTime :%v", err)
			return nil, err
		}
		lastDay, lastWeek := today.AddDate(0, 0, -1), today.AddDate(0, 0, -7)
		lastDayStr, lastWeekStr := lastDay.Format("2006-01-02"), lastWeek.Format("2006-01-02")
		if lastDayDetail, ok := dateMap2[lastDayStr]; ok {
			detail.IosCrashRateDayOverDay = utils.Div(float32(detail.IosCrashCnt-lastDayDetail.IosCrashCnt), float32(lastDayDetail.IosCrashCnt)) * 100
			detail.AndroidCrashRateDayOverDay = utils.Div(float32(detail.AndroidCrashCnt-lastDayDetail.AndroidCrashCnt), float32(lastDayDetail.AndroidCrashCnt)) * 100
		}
		if lastWeekDetail, ok := dateMap2[lastWeekStr]; ok {
			detail.IosCrashRateWeekOverWeek = utils.Div(float32(detail.IosCrashCnt-lastWeekDetail.IosCrashCnt), float32(lastWeekDetail.IosCrashCnt)) * 100
			detail.AndroidCrashRateWeekOverWeek = utils.Div(float32(detail.AndroidCrashCnt-lastWeekDetail.AndroidCrashCnt), float32(lastWeekDetail.AndroidCrashCnt)) * 100
		}
		respDetails = append(respDetails, detail)
	}
	return respDetails, nil
}
