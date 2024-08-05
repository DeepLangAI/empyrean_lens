package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

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
	Name     string
	Costs    []float64
	TotalReq int64
	FailReq  int64
	SlowReq  int64

	FailRate   float64
	SlowRate   float64
	FailReason string
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

func aigcCostAnlz(report SceneOverview, slowQueryThreshold int, autoModify bool) SceneOverview {
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

	for _, cost := range report.Costs {
		if cost > float64(slowQueryThreshold) {
			report.SlowReq++
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
		} else if log.CoreName == consts.CORE_NAME_OUTLINE {
			outlineOverview.FailReq += 1
			outlineOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
		} else if log.CoreName == consts.CORE_NAME_VIEWPOINT {
			viewpointOverview.FailReq += 1
			viewpointOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
		}
	}
	abstractOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(abstractOverview.FailReason, "\t"))), "、")
	outlineOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(outlineOverview.FailReason, "\t"))), "、")
	viewpointOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(viewpointOverview.FailReason, "\t"))), "、")

	coreLogs, _ := CommonCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_VIEWPOINT)
	for _, log := range coreLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST {
			viewpointOverview.Costs = append(viewpointOverview.Costs, log.Cost)
		}
	}

	abstractLogs, err := SummaryCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_ABSTRACT)
	for _, log := range abstractLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_ABSTRACT_ETE_COST {
			abstractOverview.Costs = append(abstractOverview.Costs, log.Cost)
		}
	}

	outlineLogs, err := SummaryCoreLogQuery(ctx, daysLookback, consts.CORE_NAME_OUTLINE)
	for _, log := range outlineLogs {
		if log.Node == consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST {
			outlineOverview.Costs = append(outlineOverview.Costs, log.Cost)
		}
	}

	//qaLogs, err := NginxIngressBasicQuery(ctx, daysLookback, consts.HOST_QA_BACKEND)
	chatCoreErrLogs, err := LingoChatCoreErrorLogs(ctx, daysLookback)
	for _, log := range chatCoreErrLogs {
		if log.CoreName == consts.CORE_NAME_CHAT {
			qaOverview.FailReq += 1
			qaOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
		}
		//} else if log.CoreName == consts.CORE_NAME_CHAT_RECOMMEND {
		//	qaRecommendOverview.FailReq += 1
		//	qaRecommendOverview.FailReason += fmt.Sprintf("\t%v", log.Msg)
		//}
	}
	qaOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(qaOverview.FailReason, "\t"))), "、")

	recommendFailCnt := QaRecommendFailcntQuery(ctx, daysLookback)
	recommendLogs, err := QaRecommendAllQuerry(ctx, daysLookback)
	for _, log := range recommendLogs {
		if log.Status == consts.StatusSuccess {
			qaRecommendOverview.Costs = append(qaRecommendOverview.Costs, log.Cost)
		}
	}
	qaRecommendOverview.TotalReq = int64(len(recommendLogs))
	qaRecommendOverview.FailReq = recommendFailCnt

	//qaRecommendOverview.FailReason = strings.Join(utils.FilterEmpty(utils.Set(strings.Split(qaRecommendOverview.FailReason, "\t"))), "、")

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
			}
		}
		//else if log.CleanUrl == "/api/chat/recommend" {
		//	qaRecommendOverview.TotalReq += 1
		//	if log.Status == "200" {
		//		qaRecommendOverview.Costs = append(qaRecommendOverview.Costs, log.Cost)
		//	}
		//}
	}

	abstractOverview = aigcCostAnlz(abstractOverview, consts.SLOWQUERY_THRESHOLD_ABSTRACT, false)
	outlineOverview = aigcCostAnlz(outlineOverview, consts.SLOWQUERY_THRESHOLD_OUTLINE, false)
	viewpointOverview = aigcCostAnlz(viewpointOverview, consts.SLOWQUERY_THRESHOLD_VIEWPOINT, false)
	qaOverview = aigcCostAnlz(qaOverview, consts.SLOWQUERY_THRESHOLD_QA, false)
	qaRecommendOverview = aigcCostAnlz(qaRecommendOverview, consts.SLOWQUERY_THRESHOLD_QA_RECOMMEND, false)

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
		} else if log.Node == consts.ALIYUN_LOG_NODE_THEME_ALL_SUMMARY {
			ov_multi_summary.Costs = append(ov_multi_summary.Costs, log.Cost)
		} else if log.Node == consts.ALIYUN_LOG_NODE_ANALYSIS_ALL {
			ov_multi_analysis.Costs = append(ov_multi_analysis.Costs, log.Cost)
		} else if log.Node == consts.ALIYUN_LOG_NODE_MERGE {
			ov_multi_merge.Costs = append(ov_multi_merge.Costs, log.Cost)
		}
	}
	ov_multi_analysis.TotalReq = ov_multi_upload.TotalReq - ov_multi_upload.FailReq
	ov_multi_merge.TotalReq = int64(len(ov_multi_analysis.Costs))
	ov_multi_summary.TotalReq = int64(len(ov_multi_merge.Costs))
	ov_multi_ete.TotalReq = ov_multi_upload.TotalReq

	ete_anlz := aigcCostAnlz(ov_multi_ete, consts.SLOWQUERY_THRESHOLD_MULTI_ETE, true)
	summary_anlz := aigcCostAnlz(ov_multi_summary, consts.SLOWQUERY_THRESHOLD_MULTI_SUMMARY, true)
	analysis_anlz := aigcCostAnlz(ov_multi_analysis, consts.SLOWQUERY_THRESHOLD_MULTI_ANALYSIS, true)
	merge_anlz := aigcCostAnlz(ov_multi_merge, consts.SLOWQUERY_THRESHOLD_MULTI_MERGE, true)
	upload_anlz := aigcCostAnlz(ov_multi_upload, consts.SLOWQUERY_THRESHOLD_FAST, false)
	//summary_anlz := aigcCostAnlz(*ov_multi_summary, consts.SLOWQUERY_THRESHOLD_MULTI_ETE)

	ov.MultiEteOverview = ete_anlz
	ov.MultiSummaryOverview = summary_anlz
	ov.MultiMergeOverview = analysis_anlz
	ov.MultiAnalysisOverview = merge_anlz
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
