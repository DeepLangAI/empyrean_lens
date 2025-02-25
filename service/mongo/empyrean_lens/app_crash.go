package empyrean_lens

import (
	"context"
	empyrean_lens2 "empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/http"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"empyrean_lens/utils"
	"errors"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"sort"
	"time"
)

type AppCrashRespData struct {
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

func GetDailyAppCrashByTime(ctx context.Context, beginTime, endTime time.Time) ([]AppCrashRespData, error) {
	appCrashDetails, e := empyrean_lens.NewAppCrashModelDao().GetAppCrashInfoByTime(ctx, beginTime, endTime)
	if e != nil {
		return nil, e
	}
	// 按照 date 分组
	dateMap := make(map[string][]empyrean_lens.AppCrashModel)
	dateMap2 := make(map[string]AppCrashRespData)
	for _, detail := range appCrashDetails {
		date := detail.Date.Local().Format("2006-01-02")
		if date == "0001-01-01" {
			continue
		}
		dateMap[date] = append(dateMap[date], detail)
	}
	var respDetails []AppCrashRespData
	for date, details := range dateMap {
		respDetail := AppCrashRespData{
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
		//hlog.CtxDebugf(ctx, "today=%v, lastDay=%v, lastWeek=%v", today, lastDay, lastWeek)
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
	sort.Slice(respDetails, func(i, j int) bool {
		return respDetails[i].Date > respDetails[j].Date
	})
	return respDetails, nil
}

func UpdateLatestAppCrashInfo(ctx context.Context, dateStr string) error {
	// 查询今天的崩溃信息
	//todayBeginStr := time.Now().Local().Format("2006-01-02") + " 00:00:00"
	//todayBegin, err := time.ParseInLocation("2006-01-02 15:04:05", todayBeginStr, time.Local)
	//if err != nil {
	//	hlog.CtxErrorf(ctx, "parse date error in UpdateLatestAppCrashInfo :%v", err)
	//	return err
	//}
	today, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse date error in UpdateLatestAppCrashInfo :%v", err)
		return err
	}
	beginTime, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr+" 00:00:00", time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse begin date error in UpdateLatestAppCrashInfo :%v", err)
	}
	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr+" 23:59:59", time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse end date error in UpdateLatestAppCrashInfo :%v", err)
	}
	if time.Now().Format("2006-01-02") == dateStr {
		endTime = time.Now()
	}
	infos, err := http.UmengDal.GetCrashInfoByTime(ctx, beginTime, endTime)
	if err != nil {
		hlog.CtxErrorf(ctx, "get crash info error in UpdateLatestAppCrashInfo :%v", err)
		return err
	}
	// 保存到数据库
	var appCrashModels []empyrean_lens.AppCrashModel
	for _, info := range infos {
		appCrashModels = append(appCrashModels, empyrean_lens.AppCrashModel{
			Time:              info.Time,
			Date:              today,
			PlatformType:      info.PlatformType,
			ErrorCount:        info.ErrorCount,
			LaunchCount:       info.LaunchCount,
			AffectedUserCount: info.AffectedUserCount,
			ActiveUserCount:   info.ActiveUserCount,
			Status:            consts.StatusValid,
			CreateTime:        time.Now(),
			UpdateTime:        time.Now(),
		})
	}
	hlog.CtxDebugf(ctx, "app crash models: %+v", appCrashModels)
	err = empyrean_lens.NewAppCrashModelDao().SaveBatch(ctx, appCrashModels)
	if err != nil {
		hlog.CtxErrorf(ctx, "save app crash info error in UpdateLatestAppCrashInfo :%v", err)
		return err
	}
	return nil
}

func GetCrashDetailsByDate(ctx context.Context, date string, platform string) ([]*empyrean_lens2.AppCrashDetailRespData, error) {
	startTimeStr, endTimeStr := date+" 00:00:00", date+" 23:59:59"
	startTime, err := time.ParseInLocation("2006-01-02 15:04:05", startTimeStr, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse start time error in GetCrashDetailsByDate :%v", err)
		return nil, err
	}
	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", endTimeStr, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse end time error in GetCrashDetailsByDate :%v", err)
		return nil, err
	}
	if startTime.After(endTime) {
		hlog.CtxErrorf(ctx, "start time must before end time in GetCrashDetailsByDate :%v", err)
		return nil, errors.New("start time must before end time")
	}
	respData, err := http.UmengDal.GetCrashDetailsByTime(ctx, startTime, endTime, platform)
	if err != nil {
		hlog.CtxErrorf(ctx, "get crash details error in GetCrashDetailsByDate :%v", err)
		return nil, err
	}
	return respData, nil
}
