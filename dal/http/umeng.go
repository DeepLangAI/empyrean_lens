package http

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"encoding/json"
	"errors"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"strings"
	"time"
)

const (
	UmengAppId_IOS     = "668fab83cac2a664de67bea4"
	UmengAppId_Android = "668fab53940d5a4c4989f047"
	UmengCookie        = "uc_session_id=59ae5d09-d7a4-4205-807d-6a3b6be2ca1c; AGL_USER_ID=4e24f1f2-4c00-4958-a1f9-50140cdd6eb2; umplus_uc_token=18gP4n9q_mfNzMXULPyi9ew_7c3c146fff11460fb1bc4bb8a6f35624; umplus_uc_loginid=%E6%B7%B1%E8%A8%80%E5%8B%87%E8%80%85; Hm_lvt_289016bc8d714b0144dc729f1f2ddc0d=1742542550,1742608821,1742783127; HMACCOUNT=8968F62A414ABE57; EGG_SESS=qrz9WlwW8xoyJYkduieHYwPIZsD23xz8Wtnq6FzYJUi7q9jqe-StJ9Zw0dkxI4aIKidxc1EFKm74ly2N3zmxe5aga0l_Lsah4bNVxC4LwzQnvqgY8zPpO9vta2ibHdsS1kM1xYFKTyISeV_HbywRqOutbV2O-Nwm_wrdzTHODgzhdQgF1cHUiPY1lqMnh7qO_NwNrfE8qVymdo0zl8EUoMMisNOLEnJzl3OL9n8d1bKKHLl-rn5nsio6wQ-3vSRugveuBcyoZRflNj7_EDnpPGYdWD-_B5oJ3zHQVukeorGSbk6g950ohPIK4pRPm-UCMg03slE2X19uaSYnNpzD_ui4uDcUud6AhqbSVXeItDk8CfBHuFwr_qsSjx7WFKD7V2ne18uENro2uExKfdt1Hi81uk-wbGGdjJGpv8FX5XBNHwxE0nQNgkiJ5pCIrOgb1uBSD5yL39y-zD8I6dTsV7gjjNYdwilvpi0jbZd44JvebKhdM3JFzWAz5FZvIVt6jtByohIjQRosnzCSdLE49IG4HmV9j8HNgzlfmhd9PcBipFb4gWbCqTNEiHq-4HMOiUUaha4Bs4ofk0H1_gwQZj8iE9ZCExXr58m5_fL2SLdHTUv2ATNceiZJmGfwYkwvXgenQ6IEecuHQIpiKRGH8_pAA6vPTpQdUbv3T-YuzmQ-pEn58sezgRCaKYWCkcj9DryZ40_IryjaUIGjNHjxBYm4UlgKWYWoQuJUNOcYTI2CXXKYKytpNVGR_2XmfVgqC2DcZWaaptXNTGi1Yf55Wa9ppqhJroC21ZYen1iT0hA=; Hm_lpvt_289016bc8d714b0144dc729f1f2ddc0d=1742783141"
	UmengCrashInfoUrl  = "https://apm.umeng.com/hsf/analysis/statOverview"
	UmengErrorInfoUrl  = "https://apm.umeng.com/hsf/analysis/errorList"
)

var UmengDal *umengDal

type umengDal struct{}

func (u *umengDal) postRequest(ctx context.Context, url string, appId string, beginTime, endTime time.Time) (string, error) {
	hlog.CtxDebugf(ctx, "umeng post request url: %s", url)
	postData := make(map[string]interface{})
	postData["dataSourceId"] = appId
	postData["errorType"] = "crash"
	postData["timeUnit"] = "unit_custom"
	postData["dateType"] = "freeDays"
	beginTimeStr, endTimeStr := beginTime.Format("20060102 150405"), endTime.Format("20060102 150405")
	postData["startDay"] = beginTimeStr
	postData["endDay"] = endTimeStr
	postData["dateRange"] = []string{beginTimeStr, endTimeStr}
	if url == UmengErrorInfoUrl {
		postData["pageSize"] = 100
	}
	hlog.CtxDebugf(ctx, "umeng post request data: %v", postData)
	res, err := utils.UmengDoPost(ctx, url, postData, UmengCookie)
	if err != nil {
		hlog.CtxErrorf(ctx, "umeng post request failed: %s", err)
		return "", err
	}
	if strings.Contains(res, "您还没有该操作的权限") {
		hlog.CtxErrorf(ctx, "umeng cookie expired: %s", res)
		return "", errors.New("umeng cookie expired")
	}
	if strings.Contains(res, "请求中含有不正确参数，请确认后再试") {
		hlog.CtxErrorf(ctx, "umeng post request failed: %s", res)
		return "", errors.New("umeng post request failed")
	}
	return res, nil
}

type crashDataRow struct {
	Value        int32   `json:"value"`
	PreValue     int32   `json:"preValue"`
	Cycle        float32 `json:"cycle"`
	PreStartTime int64   `json:"preStartTime"`
	PreEndTime   int64   `json:"preEndTime"`
}

type UmengCrashResponse struct {
	Code      int32  `json:"code"`
	Msg       string `json:"msg"`
	DetailMsg string `json:"detailMsg"`
	TraceId   string `json:"traceId"`
	Data      struct {
		ErrorCount        crashDataRow `json:"errorCount"`
		LaunchCount       crashDataRow `json:"launchCount"`
		AffectedUserCount crashDataRow `json:"affectedUserCount"`
		ActiveUserCount   crashDataRow `json:"activeUserCount"`
	} `json:"data"`
}

type UmengCrashInfo struct {
	Time              time.Time `bson:"time"`
	PlatformType      string    `bson:"platform_type"`
	ErrorCount        int32     `bson:"error_count"`
	LaunchCount       int32     `bson:"launch_count"`
	AffectedUserCount int32     `bson:"affected_user_count"`
	ActiveUserCount   int32     `bson:"active_user_count"`
}

type UmengCrashDetailInfo struct {
	Code      int    `json:"code"`
	Msg       any    `json:"msg"`
	DetailMsg any    `json:"detailMsg"`
	TraceID   string `json:"traceId"`
	Data      struct {
		RecurrenceErrorCount int `json:"recurrenceErrorCount"`
		Total                int `json:"total"`
		Page                 int `json:"page"`
		List                 []struct {
			ID                    string    `json:"id"`
			AppKey                string    `json:"appKey"`
			AppName               string    `json:"appName"`
			Os                    string    `json:"os"`
			AppVersion            string    `json:"appVersion"`
			Summary               string    `json:"summary"`
			Status                int       `json:"status"`
			ErrorType             string    `json:"errorType"`
			CrashType             string    `json:"crashType"`
			HappenTimes           int       `json:"happenTimes"`
			AffectUsers           int       `json:"affectUsers"`
			SummaryMd5            string    `json:"summaryMd5"`
			HTTPErrorCode         any       `json:"httpErrorCode"`
			AggregationKey        string    `json:"aggregationKey"`
			AggregationName       any       `json:"aggregationName"`
			Color                 any       `json:"color"`
			ErrorLevel            any       `json:"errorLevel"`
			ErrorClass            any       `json:"errorClass"`
			ErrorPageCount        any       `json:"errorPageCount"`
			FlutterAppVersion     any       `json:"flutterAppVersion"`
			HasOom                bool      `json:"hasOom"`
			FirstHappenTime       time.Time `json:"firstHappenTime"`
			Tags                  []any     `json:"tags"`
			LastHappenTime        time.Time `json:"lastHappenTime"`
			RecurrenceAppVersions any       `json:"recurrenceAppVersions"`
			Processors            []any     `json:"processors"`
		} `json:"list"`
	} `json:"data"`
}

func (u *umengDal) GetCrashInfoByTime(ctx context.Context, beginTime, endTime time.Time) ([]UmengCrashInfo, error) {
	hlog.CtxDebugf(ctx, "GetCrashInfoByTime: beginTime: %v, endTime: %v", beginTime, endTime)
	iosCrashInfoStr, err := u.postRequest(ctx, UmengCrashInfoUrl, UmengAppId_IOS, beginTime, endTime)
	if err != nil {
		hlog.CtxErrorf(ctx, "GetIOSCrashInfoByTime failed: %s", err)
		return nil, err
	}
	var iosCrashInfo UmengCrashResponse
	if err = json.Unmarshal([]byte(iosCrashInfoStr), &iosCrashInfo); err != nil {
		hlog.CtxErrorf(ctx, "GetIOSCrashInfoByTime unmarshal failed: %s", err)
		return nil, err
	}
	androidCrashInfoStr, err := u.postRequest(ctx, UmengCrashInfoUrl, UmengAppId_Android, beginTime, endTime)
	if err != nil {
		hlog.CtxErrorf(ctx, "GetAndroidCrashInfoByTime failed: %s", err)
		return nil, err
	}
	var androidCrashInfo UmengCrashResponse
	if err = json.Unmarshal([]byte(androidCrashInfoStr), &androidCrashInfo); err != nil {
		hlog.CtxErrorf(ctx, "GetAndroidCrashInfoByTime unmarshal failed: %s", err)
		return nil, err
	}
	crashInfos := make([]UmengCrashInfo, 0)
	crashInfos = append(crashInfos, UmengCrashInfo{
		Time:              endTime,
		PlatformType:      consts.Platform_IOS,
		ErrorCount:        iosCrashInfo.Data.ErrorCount.Value,
		LaunchCount:       iosCrashInfo.Data.LaunchCount.Value,
		AffectedUserCount: iosCrashInfo.Data.AffectedUserCount.Value,
		ActiveUserCount:   iosCrashInfo.Data.ActiveUserCount.Value,
	})
	crashInfos = append(crashInfos, UmengCrashInfo{
		Time:              endTime,
		PlatformType:      consts.Platform_Android,
		ErrorCount:        androidCrashInfo.Data.ErrorCount.Value,
		LaunchCount:       androidCrashInfo.Data.LaunchCount.Value,
		AffectedUserCount: androidCrashInfo.Data.AffectedUserCount.Value,
		ActiveUserCount:   androidCrashInfo.Data.ActiveUserCount.Value,
	})
	return crashInfos, nil
}

func (u *umengDal) GetCrashDetailsByTime(ctx context.Context, beginTime, endTime time.Time, Platform string) ([]*empyrean_lens.AppCrashDetailRespData, error) {
	hlog.CtxDebugf(ctx, "in GetCrashDetailsByTime: beginTime = %v, endTime = %v", beginTime, endTime)
	var appId string
	if Platform == consts.Platform_IOS {
		appId = UmengAppId_IOS
	} else {
		appId = UmengAppId_Android
	}
	crashInfoStr, err := u.postRequest(ctx, UmengErrorInfoUrl, appId, beginTime, endTime)
	if err != nil {
		hlog.CtxErrorf(ctx, "GetCrashDetailsByTime failed: %s", err)
		return nil, err
	}
	jsonBytes := []byte(crashInfoStr)
	var errDetails UmengCrashDetailInfo
	//hlog.CtxInfof(ctx, "GetCrashDetailsByTime jsonBytes = %s", jsonBytes)
	if err = json.Unmarshal(jsonBytes, &errDetails); err != nil {
		hlog.CtxErrorf(ctx, "GetCrashDetailsByTime unmarshal failed: %s", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "GetCrashDetailsByTime errDetails = %+v", errDetails)
	//return nil, nil
	respData := make([]*empyrean_lens.AppCrashDetailRespData, 0)
	for _, errDetail := range errDetails.Data.List {
		resp := empyrean_lens.AppCrashDetailRespData{
			FirstHappenTime: errDetail.FirstHappenTime.Local().Format("2006-01-02 15:04:05"),
			LastHappenTime:  errDetail.LastHappenTime.Local().Format("2006-01-02 15:04:05"),
			AppVersion:      errDetail.AppVersion,
			Summary:         errDetail.Summary,
			HappenTimes:     int64(errDetail.HappenTimes),
			AffectUsers:     int64(errDetail.AffectUsers),
		}
		respData = append(respData, &resp)
	}
	return respData, nil
}
