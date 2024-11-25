package link_trace

import (
	"context"
	"regexp"
	"strings"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/dal/aliyun"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TraceIDToEntryID(ctx context.Context, req empyrean_lens.TraceIdToEntryIdReq) (*empyrean_lens.TraceIdToEntryIdRespData, *consts.BizCode) {
	// 查询 trace_id 对应的服务日志
	entryID, err := getEntryIdFromMongo(ctx, req.TraceID)
	data := &empyrean_lens.TraceIdToEntryIdRespData{}
	if err != nil {
		hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	if entryID != "" {
		data.EntryID = entryID
		return data, nil
	}
	// 从服务日志中获取entry_id
	timeAt, _ := time.Parse("2006-01-02 15:04:05", req.Time)
	beginAt, endAt := timeAt.Add(-24*time.Hour), timeAt.Add(24*time.Hour)
	entryIDs, err := getEntryIdFromAliyun(ctx, req.UserID, req.TraceID, beginAt, endAt)
	if err != nil {
		hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	// 判断是否存在
	if len(entryIDs) != 0 {
		entryInfoList, err := searchEntryID(ctx, entryIDs)
		if err != nil {
			hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(entryInfoList) > 0 {
			data.EntryID = entryInfoList[0].EntryID
		}
	}
	return data, nil
}

func getEntryIdFromMongo(ctx context.Context, traceID string) (string, *consts.BizCode) {
	// 从mongo获取记录
	actions, err := bi.NewEntryActionDao().FindByTraceID(ctx, traceID)
	if err != nil {
		hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
		return "", &consts.QueryRecordError
	}
	for _, action := range actions {
		return action.EntryID, nil
	}
	return "", nil
}

func getEntryIdFromAliyun(ctx context.Context, userID, traceID string, beginAt, endAt time.Time) ([]string, *consts.BizCode) {
	apiLogs, err := aliyun.BusinessLogQueryByTraceIdUserId(ctx, userID, traceID, beginAt, endAt)
	if err != nil {
		hlog.CtxErrorf(ctx, "[getEntryIdFromAliyun] get api logs failed, err: %v", err)
		return []string{}, &consts.QueryRecordError
	}
	entryIDs := []string{}
	for _, log := range apiLogs {
		entryIDs = append(entryIDs, getEntryIdFromLog(log)...)
	}
	// entryids去重
	newEntryIDs := []string{}
	entryIDMap := make(map[string]bool)
	for _, entryID := range entryIDs {
		if _, ok := entryIDMap[entryID]; ok {
			continue
		}
		entryIDMap[entryID] = true
		newEntryIDs = append(newEntryIDs, entryID)
	}
	return newEntryIDs, nil
}

func getEntryIdFromLog(log aliyun.EndToEndLog) []string {
	var entryIds []string
	if entryIds = extractEntryIds(log.Message, "summary lock pair_locker, resource_id:"); len(entryIds) != 0 {
		return entryIds
	}
	if entryIds = extractEntryIds(log.Message, "file_id:"); len(entryIds) != 0 {
		return entryIds
	}
	if entryIds = extractEntryIds(log.Message, "url_id:"); len(entryIds) != 0 {
		return entryIds
	}
	if entryIds = extractEntryIds(log.Message, "\"resource_ids\":"); len(entryIds) != 0 {
		return entryIds
	}
	if entryIds = extractEntryIds(log.Message, "\"resource_ids\":"); len(entryIds) != 0 {
		return entryIds
	}
	return []string{}
}

func searchEntryID(ctx context.Context, entryIDs []string) ([]*bi.EntryInfo, *consts.BizCode) {
	entryList, err := bi.NewEntryInfoDao().FindByEntryIDsWithoutCopy(ctx, entryIDs)
	if err != nil {
		hlog.CtxErrorf(ctx, "[searchEntryID] get entry info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	return entryList, nil
}

func extractEntryIds(message, contains string) []string {
	idx := strings.Index(message, contains)
	reg := regexp.MustCompile(`[a-z0-9]+`)
	matchs := reg.FindAllString(message[idx+1:], -1)
	entryIDs := []string{}
	for _, match := range matchs {
		if _, err := primitive.ObjectIDFromHex(match); err == nil {
			entryIDs = append(entryIDs, match)
		}
	}
	return entryIDs
}
