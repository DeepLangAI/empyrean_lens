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
	entryID, err = getEntryIdFromAliyun(ctx, req.UserID, req.TraceID, timeAt)
	if err != nil {
		hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	// 判断是否存在
	if entryID != "" {
		entryInfoList, err := searchEntryID(ctx, entryID, timeAt)
		if err != nil {
			hlog.CtxErrorf(ctx, "[EntryAction] get entry action failed, err: %v", err)
			return nil, &consts.QueryRecordError
		}
		if len(entryInfoList) > 0 {
			data.EntryID = entryID
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

func getEntryIdFromAliyun(ctx context.Context, userID, traceID string, timeAt time.Time) (string, *consts.BizCode) {
	beginAt, endAt := timeAt.Add(-24*time.Hour), timeAt.Add(24*time.Hour)
	apiLogs, err := aliyun.BusinessLogQueryByTraceIdUserId(ctx, userID, traceID, beginAt, endAt)
	if err != nil {
		hlog.CtxErrorf(ctx, "[getEntryIdFromAliyun] get api logs failed, err: %v", err)
		return "", &consts.QueryRecordError
	}
	for _, log := range apiLogs {
		entryID := getEntryIdFromLog(log)
		if entryID != "" {
			return entryID, nil
		}
	}
	return "", nil
}

func getEntryIdFromLog(log aliyun.EndToEndLog) string {
	var entryId string
	if entryId = extractEntryId(log.Message, "summary lock pair_locker, resource_id:"); entryId != "" {
		return entryId
	}
	if entryId = extractEntryId(log.Message, "file_id:"); entryId != "" {
		return entryId
	}
	if entryId = extractEntryId(log.Message, "url_id:"); entryId != "" {
		return entryId
	}
	if entryId = extractEntryId(log.Message, "\"resource_ids\":"); entryId != "" {
		return entryId
	}
	return ""
}

func searchEntryID(ctx context.Context, entryID string, timeAt time.Time) ([]*bi.EntryInfo, *consts.BizCode) {
	beginAt, endAt := timeAt.Add(-24*time.Hour), timeAt.Add(24*time.Hour)
	entryList, err := bi.NewEntryInfoDao().FindByQueryAndTimeRange(ctx, entryID, []int32{}, false, beginAt, endAt, 0, 10)
	if err != nil {
		hlog.CtxErrorf(ctx, "[searchEntryID] get entry info failed, err: %v", err)
		return nil, &consts.QueryRecordError
	}
	return entryList, nil
}

func extractEntryId(message, contains string) string {
	idx := strings.Index(message, contains)
	reg := regexp.MustCompile(`[a-z0-9]+`)
	entryIDs := reg.FindAllString(message[idx+1:], -1)
	for _, entryID := range entryIDs {
		if _, err := primitive.ObjectIDFromHex(entryID); err == nil {
			return entryID
		}
	}
	return ""
}
