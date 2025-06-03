package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/dal/redis"
	"empyrean_lens/utils"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func ModelNginxIngressLogQuery(ctx context.Context, daysLookback int) ([]NginxLog, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var nlogs []NginxLog
	for host, _ := range consts.MODEL_NGINX_INGRESS_APIS {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			nginxLogs, err := ModelNginxIngressBasicQuery(ctx, daysLookback, host)
			if err != nil {
				hlog.CtxErrorf(ctx, "query nginx log error: %v", err)
				return
			}
			mu.Lock()
			nlogs = append(nlogs, nginxLogs...)
			mu.Unlock()

		}(host)

	}
	wg.Wait()
	return nlogs, nil
}

func NginxIngressLogQuery(ctx context.Context, daysLookback int) ([]NginxLog, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var nlogs []NginxLog
	for host, _ := range consts.NGINX_INGRESS_APIS {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			nginxLogs, err := NginxIngressBasicQuery(ctx, daysLookback, host)
			if err != nil {
				hlog.CtxErrorf(ctx, "query nginx log error: %v", err)
				return
			}
			mu.Lock()
			nlogs = append(nlogs, nginxLogs...)
			mu.Unlock()

		}(host)
	}
	wg.Wait()
	return nlogs, nil
}

func NginxLogsToday(ctx context.Context) ([]NginxLog, error) {
	//now := time.Now()
	//startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	//endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	nginxLogs := []NginxLog{}
	logs, err := NginxIngressLogQuery(ctx, 0)
	if err != nil {
		hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
		return nil, err
	}
	mlogs, err := ModelNginxIngressLogQuery(ctx, 0)
	if err != nil {
		hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
		return nil, err
	}
	nginxLogs = append(nginxLogs, logs...)
	nginxLogs = append(nginxLogs, mlogs...)

	startingDay := time.Now()
	from := time.Date(startingDay.Year(), startingDay.Month(), startingDay.Day(), 0, 0, 0, 0, startingDay.Location()).Format("2006-01-02")
	filteredLogs := []NginxLog{}
	// 由于采集日志有时延，当天的日志可能被落在第二天
	for _, log := range nginxLogs {
		//if log.Time.Unix() >= from {
		if log.Time.Format("2006-01-02") == from {
			filteredLogs = append(filteredLogs, log)
		}
	}

	return filteredLogs, nil
}

func NginxLogsDaysAgo(ctx context.Context, daysLookback int) ([]NginxLog, error) {
	nginxLogs := []NginxLog{}

	now := time.Now()
	startingDay := now.AddDate(0, 0, -daysLookback)
	cacheKey := fmt.Sprintf(consts.CacheKeyRequestTrend, startingDay.Format(consts.DateTemplate))
	rCache := redis.GetVal(ctx, cacheKey)
	if rCache != nil {
		bytes, err := rCache.Bytes()
		if err == nil {
			ok := utils.JSONUnMarshal(bytes, &nginxLogs)
			if ok != nil {
				hlog.CtxInfof(ctx, "成功从缓存获取nginx日志")
				return nginxLogs, nil
			}
		}
	}

	logs, err := NginxIngressLogQuery(ctx, daysLookback)
	if err != nil {
		hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
		return nil, nil
	}
	mlogs, err := ModelNginxIngressLogQuery(ctx, daysLookback)
	if err != nil {
		hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
		return nil, nil
	}
	nginxLogs = append(nginxLogs, logs...)
	nginxLogs = append(nginxLogs, mlogs...)
	//if startingDay.Format(consts.DateTemplate) != now.Format(consts.DateTemplate) {
	if daysLookback != 0 {
		bytes := utils.JSONMarshal(nginxLogs)
		err := redis.KeySet(ctx, cacheKey, bytes, time.Hour*24)
		if err != nil {
			hlog.CtxErrorf(ctx, "redis set failed: %v", err)
		}
	}
	return nginxLogs, nil
}

func NginxReportOneWeek(ctx context.Context) ([]NginxLog, error) {
	//startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	//startingDay := time.Now().AddDate(0, 0, -7)
	//from := time.Date(startingDay.Year(), startingDay.Month(), startingDay.Day(), 0, 0, 0, 0, startingDay.Location()).Unix()
	//totalDays := int(time.Since(startingDay).Hours() / 24)

	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	nginxLogs := []NginxLog{}
	for i := 0; i <= 7; i++ {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			logs, err := NginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mlogs, err := ModelNginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mutex.Lock()
			nginxLogs = append(nginxLogs, logs...)
			nginxLogs = append(nginxLogs, mlogs...)
			mutex.Unlock()
		}(i)
	}
	wg.Wait()
	return nginxLogs, nil
}
func NginxReportLongTime(ctx context.Context) ([]NginxLog, error) {
	now := time.Now()
	//startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfMonth := time.Date(2024, 7, 1, 0, 0, 0, 0, now.Location())
	totalDays := int(time.Since(startOfMonth).Hours() / 24)

	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	nginxLogs := []NginxLog{}
	for i := 0; i <= totalDays; i++ {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			logs, err := NginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mlogs, err := ModelNginxIngressLogQuery(ctx, daysLookback)
			if err != nil {
				hlog.CtxErrorf(ctx, "NginxIngressLogQuery failed: %v", err)
				return
			}
			mutex.Lock()
			nginxLogs = append(nginxLogs, logs...)
			nginxLogs = append(nginxLogs, mlogs...)
			mutex.Unlock()
		}(i)
	}
	wg.Wait()
	return nginxLogs, nil
}

func coreReportOfDays(ctx context.Context, coreName string, days []int) ([]CoreLog, error) {
	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	coreLogs := []CoreLog{}
	for _, day := range days {
		wg.Add(1)
		go func(daysLookback int) {
			defer wg.Done()
			vlogs, err := CommonCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "SummaryCoreLogQuery failed: %v")
				return
			}
			logs, err := SummaryCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "SummaryCoreLogQuery failed: %v")
				return
			}
			qaLogs, err := QaCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "QaCoreLogQuery failed: %v")
				return
			}
			multiLogs, err := MultiCoreLogQuery(ctx, daysLookback, coreName)
			if err != nil {
				hlog.CtxErrorf(ctx, "QaCoreLogQuery failed: %v")
				return
			}
			mutex.Lock()
			coreLogs = append(coreLogs, vlogs...)
			coreLogs = append(coreLogs, logs...)
			coreLogs = append(coreLogs, qaLogs...)
			coreLogs = append(coreLogs, multiLogs...)
			mutex.Unlock()
		}(day)
	}
	wg.Wait()
	return coreLogs, nil

}

func CoreReportToday(ctx context.Context, coreName string) ([]CoreLog, error) {
	days := []int{0}
	//for i := 0; i <= 7; i++ {
	//	days = append(days, i)
	//}
	logs, err := coreReportOfDays(ctx, coreName, days)
	return logs, err
}
func CoreReportOneWeek(ctx context.Context, coreName string) ([]CoreLog, error) {
	days := []int{}
	for i := 0; i <= 7; i++ {
		days = append(days, i)
	}
	logs, err := coreReportOfDays(ctx, coreName, days)
	return logs, err
}

func CoreReportLongTime(ctx context.Context, coreName string) ([]CoreLog, error) {
	now := time.Now()
	startOfMonth := time.Date(2024, 7, 1, 0, 0, 0, 0, now.Location())
	totalDays := int(time.Since(startOfMonth).Hours() / 24)

	days := []int{}
	for i := 0; i <= totalDays; i++ {
		days = append(days, i)
	}
	return coreReportOfDays(ctx, coreName, days)
}

type SceneOverview struct {
	Name      string
	Costs     []float64
	EntryLens []int
	TotalReq  int64
	FailReq   int64
	SlowReq   int64

	FailRate    float64
	SlowRate    float64
	FailReason  string
	FailDetails []string // 异常的详情信息，list of jsonString
	SlowDetails []string // 慢查询的详细信息, list of jsonString
}

type SceneOverviews struct {
	Date      string
	Overviews []SceneOverview
	//AbstractOverview      SceneOverview
	//OutlineOverview       SceneOverview
	//ViewpointOverview     SceneOverview
	//MultiOverview         SceneOverview
	//MultiAnalysisOverview SceneOverview
	//MultiMergeOverview    SceneOverview
	//MultiUploadOverview   SceneOverview
	//QaOverview            SceneOverview
	//QaRecommendOverview   SceneOverview
}

func calcOutlineSlowQuery(entryLen int) float64 {
	if entryLen >= 45000 {
		return float64(360)
	}
	return consts.SLOWQUERY_THRESHOLD_OUTLINE
}

func calcQaRecommendSlowQuery(entryLen int) float64 {
	//if entryLen == 0 {
	//	return consts.SLOWQUERY_THRESHOLD_QA_RECOMMEND
	//}
	//return float64(entryLen) * consts.SLOWQUERY_THRESHOLD_QA_RECOMMEND / 1000
	return consts.SLOWQUERY_THRESHOLD_QA_RECOMMEND
}

const (
	SCENE_REPORT_TYPE_GENERAL = iota
	SCENE_REPORT_TYPE_OUTLINE
	SCENE_REPORT_TYPE_QA_RECOMMEND
)

// 判断是否为新增的 feed 接口
func isNewFeedApi(apiPath string) bool {
	// 只对你新增的接口做判断
	switch apiPath {
	case "/api/feed/v1/lingowhale_daily/list",
		"/api/feed/v1/lingowhale_daily/get",
		"/api/feed/v1/topic/list",
		"/api/feed/v1/topic/get",
		"/api/feed/v1/feed/topic",
		"/api/feed/v1/user_subscribe/list",
		"/api/feed/v2/feed/subscription",
		"/api/feed/v1/search/list",
		"/api/feed/v1/subscription_channel/search",
		"/api/feed/v1/user_subscribe/upsert",
		"/api/feed/v1/subscription_channel/upsert",
		"/api/feed/v1/subscription_channel/category",
		"/api/feed/v1/feed/recommend",
		"/api/feed/v1/subscription_channel/get":
		return true
	default:
		return false
	}
}

func aigcCostAnlz(report SceneOverview, slowQueryThreshold int, autoModify bool, reportType int) SceneOverview {
	if autoModify {
		if report.FailReq == 0 {
			report.FailReq = report.TotalReq - int64(len(report.Costs))
		}
		if report.FailReq < 0 {
			report.FailReq = 0
		}
	}
	if report.FailReq != 0 {
		report.FailRate = float64(report.FailReq) / float64(report.TotalReq) * 100
	}

	for i, cost := range report.Costs {
		switch reportType {
		case SCENE_REPORT_TYPE_GENERAL:
			var threshold float64
			if isNewFeedApi(report.Name) {
				threshold = 1.0
			} else {
				threshold = float64(slowQueryThreshold)
			}
			if cost > threshold {
				report.SlowReq++
			}
		case SCENE_REPORT_TYPE_OUTLINE:
			if cost > calcOutlineSlowQuery(report.EntryLens[i]) {
				report.SlowReq++
			}
		case SCENE_REPORT_TYPE_QA_RECOMMEND:
			if cost > calcQaRecommendSlowQuery(report.EntryLens[i]) {
				report.SlowReq++
			}
		}
	}
	// 慢查询率，即成功的响应中，慢查询的比例
	if report.SlowReq != 0 {
		report.SlowRate = float64(report.SlowReq) / float64(report.TotalReq-report.FailReq) * 100
	}
	return report
}

func SceneGeneralOfDay(ctx context.Context, daysLookback int) (*SceneOverviews, error) {
	overviews := &SceneOverviews{}
	cnts, err := SummreqCntQuery(ctx, daysLookback)
	if err != nil {
		hlog.CtxErrorf(ctx, "err: %v", err)
		return nil, err
	}
	multiOv, err := MultiGeneralOfDay(ctx, daysLookback)
	if err != nil {
		hlog.CtxErrorf(ctx, "err: %v", err)
		return nil, err
	}
	//multiEteOv, multiAnalysisOv, multiMergeOv, err := MultiGeneralOfDay(ctx, daysLookback)
	overviews.Overviews = append(overviews.Overviews, multiOv.MultiEteOverview)
	overviews.Overviews = append(overviews.Overviews, multiOv.MultiSummaryOverview)
	overviews.Overviews = append(overviews.Overviews, multiOv.MultiAnalysisOverview)
	overviews.Overviews = append(overviews.Overviews, multiOv.MultiMergeOverview)
	overviews.Overviews = append(overviews.Overviews, multiOv.MultiUploadOverview)

	abstractOverview := SceneOverview{Name: "单文档：全文速览", Costs: []float64{}, TotalReq: int64(cnts["0"]), FailReq: 0}
	outlineOverview := SceneOverview{Name: "单文档：智能大纲", Costs: []float64{}, TotalReq: int64(cnts["1"]), FailReq: 0}
	viewpointOverview := SceneOverview{Name: "单文档：关键信息", Costs: []float64{}, TotalReq: int64(cnts["3"]), FailReq: 0}
	qaOverview := SceneOverview{Name: "问答：问答", Costs: []float64{}, TotalReq: 0, FailReq: 0}
	qaRecommendOverview := SceneOverview{Name: "问答：问题推荐", Costs: []float64{}, TotalReq: 0, FailReq: 0}

	lingoCoreErrLogs, err := LingoCoreErrorLogs(ctx, daysLookback)
	for _, log := range lingoCoreErrLogs {
		if log.CoreName == consts.CORE_NAME_ABSTRACT {
			abstractOverview.FailReq += 1
			abstractOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
			abstractOverview.FailDetails = append(abstractOverview.FailDetails, utils.JSONMarshal(log))
		} else if log.CoreName == consts.CORE_NAME_OUTLINE {
			outlineOverview.FailReq += 1
			outlineOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
			outlineOverview.FailDetails = append(outlineOverview.FailDetails, utils.JSONMarshal(log))
		} else if log.CoreName == consts.CORE_NAME_VIEWPOINT {
			viewpointOverview.FailReq += 1
			viewpointOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
			viewpointOverview.FailDetails = append(viewpointOverview.FailDetails, utils.JSONMarshal(log))
		}
	}
	abstractOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(abstractOverview.FailReason, "\t"))), "、")
	outlineOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(outlineOverview.FailReason, "\t"))), "、")
	viewpointOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(viewpointOverview.FailReason, "\t"))), "、")

	coreLogs, _ := CommonCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_VIEWPOINT)
	for _, log := range coreLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST {
			viewpointOverview.Costs = append(viewpointOverview.Costs, log.Cost)
			if log.Cost > consts.SLOWQUERY_THRESHOLD_VIEWPOINT {
				viewpointOverview.SlowDetails = append(viewpointOverview.SlowDetails, utils.JSONMarshal(log))
			}
		}
	}

	abstractLogs, err := SummaryCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_ABSTRACT)
	for _, log := range abstractLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_ABSTRACT_ETE_COST {
			abstractOverview.Costs = append(abstractOverview.Costs, log.Cost)
			if log.Cost > consts.SLOWQUERY_THRESHOLD_ABSTRACT {
				abstractOverview.SlowDetails = append(abstractOverview.SlowDetails, utils.JSONMarshal(log))
			}
		}
	}

	outlineLogs, err := SummaryCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_OUTLINE)
	for _, log := range outlineLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST {
			outlineOverview.Costs = append(outlineOverview.Costs, log.Cost)
			outlineOverview.EntryLens = append(outlineOverview.EntryLens, log.EntryLen)
			if log.Cost > calcOutlineSlowQuery(log.EntryLen) {
				outlineOverview.SlowDetails = append(outlineOverview.SlowDetails, utils.JSONMarshal(log))
			}
		}
	}

	chatCoreErrLogs, err := LingoChatCoreErrorLogs(ctx, daysLookback)
	for _, log := range chatCoreErrLogs {
		if log.CoreName == consts.CORE_NAME_CHAT {
			qaOverview.FailReq += 1
			qaOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
			qaOverview.FailDetails = append(qaOverview.FailDetails, utils.JSONMarshal(log))
		}
	}
	qaOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(qaOverview.FailReason, "\t"))), "、")

	recommendFailCnt := QaRecommendFailcntQuery(ctx, daysLookback)
	recommendLogs, err := QaRecommendAllQuerry(ctx, daysLookback)
	for _, log := range recommendLogs {
		if log.Status == consts.StatusSuccess {
			qaRecommendOverview.Costs = append(qaRecommendOverview.Costs, log.Cost)
			qaRecommendOverview.EntryLens = append(qaRecommendOverview.EntryLens, log.EntryLen)
			if log.Cost > calcQaRecommendSlowQuery(log.EntryLen) {
				qaRecommendOverview.SlowDetails = append(qaRecommendOverview.SlowDetails, utils.JSONMarshal(log))
			}
		}
	}
	qaRecommendOverview.TotalReq = int64(len(recommendLogs))
	qaRecommendOverview.FailReq = recommendFailCnt

	apis := []string{}
	for _, val := range consts.NGINX_INGRESS_APIS[consts.HOST_QA_BACKEND] {
		apis = append(apis, val.Api)
	}
	qaLogs, err := QaMiddlewareRespLogQuery(ctx, daysLookback, apis)
	for _, log := range qaLogs {
		if log.CleanUrl == "/api/chat/qa" {
			qaOverview.TotalReq += 1
			if log.Status == "200" {
				qaOverview.Costs = append(qaOverview.Costs, log.Cost)
				if log.Cost > consts.SLOWQUERY_THRESHOLD_QA {
					qaOverview.SlowDetails = append(qaOverview.SlowDetails, utils.JSONMarshal(log))
				}
			}
		}
	}

	abstractOverview = aigcCostAnlz(abstractOverview, consts.SLOWQUERY_THRESHOLD_ABSTRACT, false, SCENE_REPORT_TYPE_GENERAL)
	outlineOverview = aigcCostAnlz(outlineOverview, consts.SLOWQUERY_THRESHOLD_OUTLINE, false, SCENE_REPORT_TYPE_OUTLINE)
	viewpointOverview = aigcCostAnlz(viewpointOverview, consts.SLOWQUERY_THRESHOLD_VIEWPOINT, false, SCENE_REPORT_TYPE_GENERAL)
	qaOverview = aigcCostAnlz(qaOverview, consts.SLOWQUERY_THRESHOLD_QA, false, SCENE_REPORT_TYPE_GENERAL)
	qaRecommendOverview = aigcCostAnlz(qaRecommendOverview, consts.SLOWQUERY_THRESHOLD_QA_RECOMMEND, false, SCENE_REPORT_TYPE_QA_RECOMMEND)

	overviews.Overviews = append(overviews.Overviews, abstractOverview)
	overviews.Overviews = append(overviews.Overviews, outlineOverview)
	overviews.Overviews = append(overviews.Overviews, viewpointOverview)
	overviews.Overviews = append(overviews.Overviews, qaOverview)
	overviews.Overviews = append(overviews.Overviews, qaRecommendOverview)

	overviews.Date = time.Now().AddDate(0, 0, -daysLookback).Format("2006-01-02")

	return overviews, nil
}

type MultiOverviews struct {
	MultiEteOverview      SceneOverview
	MultiSummaryOverview  SceneOverview
	MultiMergeOverview    SceneOverview
	MultiAnalysisOverview SceneOverview
	MultiUploadOverview   SceneOverview
}

func MultiGeneralOfDay(ctx context.Context, daysLookback int) (*MultiOverviews, error) {
	ov := &MultiOverviews{}
	ov_multi_ete := SceneOverview{Name: "多文档：4总结端到端"}
	ov_multi_summary := SceneOverview{Name: "多文档：4多文档总结"}
	ov_multi_merge := SceneOverview{Name: "多文档：3多文档整合"}
	ov_multi_analysis := SceneOverview{Name: "多文档：2单文档分析全部完成"}
	ov_multi_upload := SceneOverview{Name: "多文档：1多文档上传"}
	//ov_multi_summary := &SceneOverview{Name: "多文档：多文档总结"}
	total, err := MultiTotalRequestQuery(ctx, daysLookback)
	if err != nil {
		return nil, err
	}
	multiNodeErrorLogs, err := MultiNodeErrorQuery(ctx, daysLookback)
	if err != nil {
		return nil, err
	}
	for _, log := range multiNodeErrorLogs {
		if log.NodeName == "多文档整合" {
			ov_multi_merge.FailDetails = append(ov_multi_merge.FailDetails, utils.JSONMarshal(log))
		} else if log.NodeName == "多文档总结" {
			ov_multi_summary.FailDetails = append(ov_multi_summary.FailDetails, utils.JSONMarshal(log))
		}
	}

	multiUploadErrCnt := 0
	errorLogs, err := LingoCoreErrorLogs(ctx, daysLookback)
	for _, log := range errorLogs {
		if log.CoreName == consts.CORE_NAME_MULTI {
			multiUploadErrCnt += 1
		}
	}
	ov_multi_upload.TotalReq = int64(total)
	ov_multi_upload.FailReq = int64(multiUploadErrCnt)

	logs, err := MultiCoreLogQuery(ctx, daysLookback, "multi")
	for _, log := range logs {
		if log.Node == consts.ALIYUN_LOG_NODE_MULTI_ALL_SUCCESS {
			ov_multi_ete.Costs = append(ov_multi_ete.Costs, log.Cost)
			if log.Cost > consts.SLOWQUERY_THRESHOLD_MULTI_ETE {
				ov_multi_ete.SlowDetails = append(ov_multi_ete.SlowDetails, utils.JSONMarshal(log))
			}
		} else if log.Node == consts.ALIYUN_LOG_NODE_THEME_ALL_SUMMARY {
			ov_multi_summary.Costs = append(ov_multi_summary.Costs, log.Cost)
			if log.Cost > consts.SLOWQUERY_THRESHOLD_MULTI_SUMMARY {
				ov_multi_summary.SlowDetails = append(ov_multi_summary.SlowDetails, utils.JSONMarshal(log))
			}
		} else if log.Node == consts.ALIYUN_LOG_NODE_ANALYSIS_ALL {
			ov_multi_analysis.Costs = append(ov_multi_analysis.Costs, log.Cost)
			if log.Cost > consts.SLOWQUERY_THRESHOLD_MULTI_ANALYSIS {
				ov_multi_analysis.SlowDetails = append(ov_multi_analysis.SlowDetails, utils.JSONMarshal(log))
			}
		} else if log.Node == consts.ALIYUN_LOG_NODE_MERGE {
			ov_multi_merge.Costs = append(ov_multi_merge.Costs, log.Cost)
			if log.Cost > consts.SLOWQUERY_THRESHOLD_MULTI_MERGE {
				ov_multi_merge.SlowDetails = append(ov_multi_merge.SlowDetails, utils.JSONMarshal(log))
			}
		}
	}
	ov_multi_analysis.TotalReq = ov_multi_upload.TotalReq - ov_multi_upload.FailReq
	ov_multi_merge.TotalReq = int64(len(ov_multi_analysis.Costs))
	ov_multi_summary.TotalReq = int64(len(ov_multi_merge.Costs))
	ov_multi_ete.TotalReq = ov_multi_upload.TotalReq

	ete_anlz := aigcCostAnlz(ov_multi_ete, consts.SLOWQUERY_THRESHOLD_MULTI_ETE, true, SCENE_REPORT_TYPE_GENERAL)
	summary_anlz := aigcCostAnlz(ov_multi_summary, consts.SLOWQUERY_THRESHOLD_MULTI_SUMMARY, true, SCENE_REPORT_TYPE_GENERAL)
	analysis_anlz := aigcCostAnlz(ov_multi_analysis, consts.SLOWQUERY_THRESHOLD_MULTI_ANALYSIS, true, SCENE_REPORT_TYPE_GENERAL)
	merge_anlz := aigcCostAnlz(ov_multi_merge, consts.SLOWQUERY_THRESHOLD_MULTI_MERGE, true, SCENE_REPORT_TYPE_GENERAL)
	upload_anlz := aigcCostAnlz(ov_multi_upload, consts.SLOWQUERY_THRESHOLD_FAST, false, SCENE_REPORT_TYPE_GENERAL)
	//summary_anlz := aigcCostAnlz(*ov_multi_summary, consts.SLOWQUERY_THRESHOLD_MULTI_ETE)

	ov.MultiEteOverview = ete_anlz
	ov.MultiSummaryOverview = summary_anlz
	ov.MultiMergeOverview = merge_anlz
	ov.MultiAnalysisOverview = analysis_anlz
	ov.MultiUploadOverview = upload_anlz
	return ov, nil
}

func SceneGeneralOverview(ctx context.Context, days []int) []SceneOverviews {
	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	overviews := []SceneOverviews{}
	for _, day := range days {
		wg.Add(1)
		go func(lookbackDay int) {
			defer wg.Done()
			ov, err := SceneGeneralOfDay(ctx, lookbackDay)
			if err != nil {
				hlog.CtxErrorf(ctx, "err: %v", err)
				return
			}
			mutex.Lock()
			overviews = append(overviews, *ov)
			mutex.Unlock()
		}(day)
	}
	wg.Wait()
	sort.Slice(overviews, func(i, j int) bool {
		return overviews[i].Date > overviews[j].Date
	})
	return overviews
}

func NginxBizErrorLogsOfAPI(ctx context.Context, host, url, date string) ([]NginxErrorLog, error) {
	//if _, ok := consts.MODEL_NGINX_INGRESS_APIS[host]; ok {
	//	errLogs, err := ModelNginxErrlogsQuery(ctx, host, url, date)
	//	return errLogs, err
	//}
	if _, ok := consts.NGINX_INGRESS_APIS[host]; ok {
		errLogs, err := NginxBizErrlogsQuery(ctx, host, url, date)
		return errLogs, err
	}
	return nil, errors.New("query nginx biz error logs, error")
}
func NginxErrorLogsOfAPI(ctx context.Context, host, url, date string) ([]NginxErrorLog, error) {
	if _, ok := consts.MODEL_NGINX_INGRESS_APIS[host]; ok {
		errLogs, err := ModelNginxErrlogsQuery(ctx, host, url, date)
		return errLogs, err
	}
	if _, ok := consts.NGINX_INGRESS_APIS[host]; ok {
		errLogs, err := NginxErrlogsQuery(ctx, host, url, date)
		return errLogs, err
	}
	errLogs := []NginxErrorLog{}
	if logs, err := ModelNginxErrlogsQuery(ctx, host, url, date); err == nil {
		errLogs = append(errLogs, logs...)
	}
	if logs, err := NginxErrlogsQuery(ctx, host, url, date); err == nil {
		errLogs = append(errLogs, logs...)
	}
	if len(errLogs) != 0 {
		sort.Slice(errLogs, func(i, j int) bool {
			return errLogs[i].Time.Unix() > errLogs[j].Time.Unix()
		})
		return errLogs, nil
	}
	return nil, errors.New("query nginx error logs, error")
}
func modifyOriginLog(originLog map[string]string) map[string]string {
	for key1, key2 := range map[string]string{
		"asctime":         "time",
		"msg":             "message",
		"ip":              "client_ip",
		"http_user_agent": "ua",
		"vhost":           "host",
		"url":             "path",
		"request_time":    "cost",
		"duration":        "cost",
		"levelname":       "level",
	} {
		if _, ok1 := originLog[key1]; ok1 {
			if _, ok2 := originLog[key2]; ok2 {
				delete(originLog, key1)
			}
		}
	}
	for key1, key2 := range map[string]string{
		"levelname": "level",
	} {
		if _, ok1 := originLog[key1]; ok1 {
			if _, ok2 := originLog[key2]; !ok2 {
				value := originLog[key1]
				delete(originLog, key1)
				originLog[key2] = value
			}
		}
	}
	for _, key := range []string{
		"level",
	} {
		if _, ok := originLog[key]; ok {
			originLog[key] = strings.ToUpper(originLog[key])
		}
	}
	for _, key := range []string{
		"upstream_addr",
		"request_length",
		"body_bytes_sent",
		"upstream_response_length",
		"upstream_response_time",
		"upstream_status",
		"x_forward_for",
		"proxy_upstream_name",
		"thread",
		"threadname",
		"version",
	} {
		if _, ok := originLog[key]; ok {
			delete(originLog, key)
		}
	}
	return originLog
}

func EndToEndLogsQuery(ctx context.Context, traceId, date string) ([]EndToEndLog, error) {
	mu := sync.Mutex{}
	results := []EndToEndLog{}
	funcList := []utillib.AsyncFunc{}
	funcList = append(funcList, func() error {
		logs, err := NginxLogQueryByTraceId(ctx, traceId, date)
		if err != nil {
			return err
		}
		mu.Lock()
		results = append(results, logs...)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		logs, err := ModelNginxLogQueryByTraceId(ctx, traceId, date)
		if err != nil {
			return err
		}
		mu.Lock()
		results = append(results, logs...)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		logs, err := BusinessLogQueryByTraceId(ctx, traceId, date)
		if err != nil {
			return err
		}
		mu.Lock()
		results = append(results, logs...)
		mu.Unlock()
		return nil
	})
	errs := utillib.ParallelExec(ctx, funcList, len(funcList))
	if len(errs) != 0 {
		return nil, errors.New("query end to end trace logs, error")
	}
	sort.Slice(results, func(i, j int) bool {
		//return results[i].Time[11:19] > results[j].Time[11:19]
		return results[i].Time > results[j].Time
	})
	for _, result := range results {
		result.OriginLog = modifyOriginLog(result.OriginLog)
	}
	return results, nil
}

func EndToEndUserLogsQuery(ctx context.Context, userId, timeBegin, timeEnd string) ([]EndToEndLog, error) {
	mu := sync.Mutex{}
	results := []EndToEndLog{}
	funcList := []utillib.AsyncFunc{}
	startTime, e := time.ParseInLocation(consts.DateHourMinuteTemplate, timeBegin, time.Local)
	if e != nil {
		return nil, e
	}
	endTime, e := time.ParseInLocation(consts.DateHourMinuteTemplate, timeEnd, time.Local)
	if e != nil {
		return nil, e
	}
	funcList = append(funcList, func() error {
		logs, err := NginxLogQueryByUserId(ctx, userId, startTime, endTime)
		if err != nil {
			return err
		}
		mu.Lock()
		results = append(results, logs...)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		logs, err := ModelNginxLogQueryByUserId(ctx, userId, startTime, endTime)
		if err != nil {
			return err
		}
		mu.Lock()
		results = append(results, logs...)
		mu.Unlock()
		return nil
	})
	funcList = append(funcList, func() error {
		logs, err := BusinessLogQueryByUserId(ctx, userId, startTime, endTime)
		if err != nil {
			return err
		}
		mu.Lock()
		results = append(results, logs...)
		mu.Unlock()
		return nil
	})
	errs := utillib.ParallelExec(ctx, funcList, len(funcList))
	if len(errs) != 0 {
		return nil, errors.New("query end to end userTrace logs, error")
	}
	sort.Slice(results, func(i, j int) bool {
		//return results[i].Time[11:19] > results[j].Time[11:19]
		return results[i].Time > results[j].Time
	})
	for _, result := range results {
		result.OriginLog = modifyOriginLog(result.OriginLog)
	}
	return results, nil
}

func TracebackQueryOfTimespan(ctx context.Context, timespan int) []TracebackDetail {
	days := []int{}
	if timespan == consts.TIMESPAN_TODAY {
		days = append(days, 0)
	} else if timespan == consts.TIMESPAN_WEEK {
		for i := 0; i < 7; i++ {
			days = append(days, i)
		}
	} else if timespan == consts.TIMESPAN_MONTH {
		for i := 0; i < 30; i++ {
			days = append(days, i)
		}
	}

	return TracebackQueryOfDays(ctx, days)
}

func TracebackQueryOfDays(ctx context.Context, days []int) []TracebackDetail {
	var mutex sync.Mutex
	wg := sync.WaitGroup{}

	results := []TracebackDetail{}
	for _, day := range days {
		wg.Add(1)
		go func(lookbackDay int) {
			defer wg.Done()
			logs, err := TracebackQuery(ctx, lookbackDay)
			if err != nil {
				hlog.CtxErrorf(ctx, "err: %v", err)
				return
			}
			mutex.Lock()
			results = append(results, logs...)
			mutex.Unlock()
		}(day)
	}
	wg.Wait()
	sort.Slice(results, func(i, j int) bool {
		return results[i].Time.Unix() > results[j].Time.Unix()
	})
	return results
}

// func TraceSingleDocument(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error) {
// 	res, funcList := []FileProcessLog{}, []utillib.AsyncFunc{}
// 	mu := &sync.Mutex{}
// 	funcList = append(funcList, func() error {
// 		overviewLogs, err := QuerySingleLogs(ctx, resourceId, timeBegin, timeEnd, SingleOverviewBeginQuery, SingleOverviewEndQuery)
// 		if err != nil {
// 			return err
// 		}
// 		mu.Lock()
// 		res = append(res, overviewLogs...)
// 		mu.Unlock()
// 		return nil
// 	})
// 	funcList = append(funcList, func() error {
// 		overviewLogs, err := QuerySingleLogs(ctx, resourceId, timeBegin, timeEnd, SingleOutlineBeginQuery, SingleOutlineEndQuery)
// 		if err != nil {
// 			return err
// 		}
// 		mu.Lock()
// 		res = append(res, overviewLogs...)
// 		mu.Unlock()
// 		return nil
// 	})
// 	funcList = append(funcList, func() error {
// 		overviewLogs, err := QuerySingleLogs(ctx, resourceId, timeBegin, timeEnd, SingleViewpointBeginQuery, SingleViewpointEndQuery)
// 		if err != nil {
// 			return err
// 		}
// 		mu.Lock()
// 		res = append(res, overviewLogs...)
// 		mu.Unlock()
// 		return nil
// 	})

// 	errs := utillib.ParallelExec(ctx, funcList, len(funcList))
// 	if len(errs) > 0 {
// 		hlog.CtxErrorf(ctx, "[TraceSingleDocument] error:%+v", utils.JSONMarshal(errs))
// 		return nil, errs[0]
// 	}
// 	sort.Slice(res, func(i, j int) bool {
// 		return res[i].Asctime.Before(res[i].Asctime)
// 	})
// 	return res, nil
// }

type SingleQueryFunc func(ctx context.Context, id string, timeBegin, timeEnd time.Time) ([]FileProcessLog, error)

func QuerySingleLogs(ctx context.Context, resourceId string, timeBegin, timeEnd time.Time, beginFunc, endFunc SingleQueryFunc) ([]FileProcessLog, error) {
	var res []FileProcessLog
	logs, err := beginFunc(ctx, resourceId, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	res = append(res, logs...)

	wg, logChannel, errChan := &sync.WaitGroup{}, make(chan []FileProcessLog, len(res)), make(chan error, len(res))
	for _, log := range logs {
		wg.Add(1)
		go func(traceId string) {
			defer wg.Done()
			logs, err := endFunc(ctx, traceId, timeBegin, timeEnd)
			if err != nil {
				errChan <- err
				return
			}
			logChannel <- logs
		}(log.TraceId)
	}
	wg.Wait()
	close(logChannel)
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	for logs := range logChannel {
		res = append(res, logs...)
	}
	return res, nil
}
