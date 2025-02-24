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
	UmengCookie        = "UM_distinctid=194e42f9677734-0fe6bcad00218-1d525636-1fa400-194e42f96789c2; cna=59b07be2eb2f4199832de5a715f27f16; dplus_finger_print=3386071896; isg=BBUVQR8L3vL75PoO_ySd3kqyJBfPEskkU8n9W5e6kQzb7jTgX2BU9OqtvPLYbuHc; tfstk=gcrS02O5eWE2LGO0E_Bq11bxeimC3WswJpMLIJKyp0n-9HwgB8uU4JKLA5P4a2Co4kNbTWc8TTVUpSPaGT2zTgqQO5V78LSoaHhQ95ty4JEEpkiu0z-Ea_mQp5oC_1SNb82ojDCN_QwxCootQUKJT2exDDkINaDBo82oxKv29NzYEB16eANKvWnxDYDSvY389mLxivDpwvKRh-hmMvp-w2nxkAk9vXFKvtwxivnK9AkWCv_SZ86S6qWVCLkaejtpvuInPfMJ8HtQcYg7v8hbF8ZjF4GtogD8QlwTelDQxOLIOza8axVCfsiTSSZIWDsBm0y_k7gbP6tjoregvqac1F2gpR4KlkORU5eLZoiofNL767w_R0VPLeDbRJEiqD_6Exr7Hl0Qq_txNS2Y70qOiHNT3Rrook16MX4rImHQxOLIO2Iyahl1GXLBhVxIhfWfheYHyvFjf8mSFUg-nYgVhtO8-CxpbhqFheijyxDSFt6Xwyf..; AGL_USER_ID=f4a64b11-9c99-457a-a405-f5591fd57c4f; Hm_lvt_289016bc8d714b0144dc729f1f2ddc0d=1738995111,1739168652,1739176248; HMACCOUNT=6F7A54812AC487BC; uc_session_id=e3da5ff9-164a-4c45-b443-6233a32d0d50; umplus_uc_token=1yKwn0NAS6smxTZE0M5hnjw_0b70115e36d94b9aba956d8a113233ed; umplus_uc_loginid=youmeng452436577095; EGG_SESS=qrz9WlwW8xoyJYkduieHYwPIZsD23xz8Wtnq6FzYJUjqmCEIlEUfTT5IgaMRotArOYCXior1GWTr1sUVE9A_gNxNGBx1pn0dJz-Ue65Guolr4Nd3u3UISzp5N72PHRu1hn0KC32Si9rU33rptl5jhTa5D05F42Uzpo70uOkFRyjC7GjhwCeodrnoySFQV56BeV1dBeHwpvoPeODOL7tOSpyNs1LP3PQM49x-Hnj7LS9ZqT2Aett8CT4cnGmQxREONq1i02P2nTKk3_I8g9lwsRUgZ6EPZKB1dAgQsAjOBW7jvn5TEuNjTP0MJHEv0IMMsorx0FSln7G9Br4F7xV5MNEVhTayGuW30C8YrH8RgvBbcSHdsdVQ2B2kILjSujkNx5oCPesVO9HvtPUay-I2NfxIrRWsQby8h4D0XtzeDKpMmOpcbhP5VXQrsXJW2sbYAiBnlVbd7E-YrLWw7lOiCThJx4Peo_fNDCBjsQrSqp_UTlspwo2m8ewbivlAoOjyoxf91ybkYpuRI2NH87PaIbnu-NFC1fOwy_Bo2YR9ub3qYHDtghjYBsVzLe9N7SOFdHi8QEjFCOKJooA4vZyrDY4RL6GiXZE_b9Dw5F589sTokUWC9saGDmJMN1Uwp5lA6oILoKtcc6IKt3ohkBz9p1OpfIAAqXZQl6Zxwiodv36nduLA5fTa3vc61TLtOiQCMLMmtizmlKPRYS4wtaXNqN6fb3FyQ7ZPif8L2O02At4aK04u4BPIz6jQ4m8Wxd9LYpddoQbl78-KjqA0lLdlEiJ1S52Mk8wv4wKy6H9TYYg=; CNZZDATA1281115298=802761754-1739003815-https%253A%252F%252Fwww.umeng.com%252F%7C1740107668; Hm_lpvt_289016bc8d714b0144dc729f1f2ddc0d=1740107668"
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
