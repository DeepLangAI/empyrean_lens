package empyrean_lens

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

var TableNameDailyAutomationStats = "daily_automation_stats"

type DailyAutomationStatsModel struct {
	Date           string    `bson:"date"`            // 统计日期 YYYY-MM-DD
	SingleFailNum  int       `bson:"single_fail_num"` // PDF文档失败数
	SingleTotalNum int       `bson:"single_total_num"`
	SingleDocNum   int       `bson:"single_doc_num"`
	WebFailNum     int       `bson:"web_fail_num"`
	WebTotalNum    int       `bson:"web_total_num"`
	WebDocNum      int       `bson:"web_doc_num"`
	MultiFailNum   int       `bson:"multi_fail_num"`
	MultiTotalNum  int       `bson:"multi_total_num"`
	MultiDocNum    int       `bson:"multi_doc_num"`
	CreateTime     time.Time `bson:"create_time"`
	UpdateTime     time.Time `bson:"update_time"`
}

type DailyAutomationStatsDao struct{}

var dailyAutomationStatsDao *DailyAutomationStatsDao
var dailyAutomationStatsInitOnce sync.Once

func NewDailyAutomationStatsDao() *DailyAutomationStatsDao {
	dailyAutomationStatsInitOnce.Do(func() {
		dailyAutomationStatsDao = &DailyAutomationStatsDao{}
	})
	return dailyAutomationStatsDao
}

// GetStatsByDateRange 根据日期范围获取统计数据
func (self *DailyAutomationStatsDao) GetStatsByDateRange(ctx context.Context, beginTime, endTime time.Time) ([]DailyAutomationStatsModel, error) {
	startDate := beginTime.Format("2006-01-02")
	endDate := endTime.Format("2006-01-02")

	var result []DailyAutomationStatsModel
	cur, err := probeDatabase.
		Collection(TableNameDailyAutomationStats).
		Find(ctx, bson.M{
			"date": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		})

	if err != nil {
		hlog.CtxErrorf(ctx, "find daily_automation_stats failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "decode daily_automation_stats failed, err: %v", err)
		return nil, err
	}
	return result, nil
}

// UpsertDailyStats 更新或插入每日统计数据
func (self *DailyAutomationStatsDao) UpsertDailyStats(ctx context.Context, stats *DailyAutomationStatsModel) error {
	stats.UpdateTime = time.Now()
	if stats.CreateTime.IsZero() {
		stats.CreateTime = stats.UpdateTime
	}

	filter := bson.M{"date": stats.Date}
	update := bson.M{"$set": stats}
	opts := options.Update().SetUpsert(true)

	_, err := probeDatabase.
		Collection(TableNameDailyAutomationStats).
		UpdateOne(ctx, filter, update, opts)

	if err != nil {
		hlog.CtxErrorf(ctx, "upsert daily_automation_stats failed, err: %v", err)
		return err
	}
	return nil
}

// SaveBatch 批量保存统计数据
func (self *DailyAutomationStatsDao) SaveBatch(ctx context.Context, models []DailyAutomationStatsModel) error {
	if len(models) == 0 {
		return nil
	}

	operations := make([]mongo.WriteModel, len(models))

	for i := range models {
		// 创建 upsert 操作
		operations[i] = mongo.NewUpdateOneModel().
			SetFilter(bson.M{"date": models[i].Date}). // 使用日期作为唯一键
			SetUpdate(bson.M{"$set": models[i]}).
			SetUpsert(true)
	}

	_, err := probeDatabase.
		Collection(TableNameDailyAutomationStats).
		BulkWrite(ctx, operations, options.BulkWrite().SetOrdered(false))

	if err != nil {
		hlog.CtxErrorf(ctx, "batch upsert daily_automation_stats failed, err: %v", err)
		return err
	}
	return nil
}

// GetStatsByDate 获取指定日期的统计数据
func (self *DailyAutomationStatsDao) GetStatsByDate(ctx context.Context, date string) (*DailyAutomationStatsModel, error) {
	var result DailyAutomationStatsModel
	err := probeDatabase.
		Collection(TableNameDailyAutomationStats).
		FindOne(ctx, bson.M{"date": date}).
		Decode(&result)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		hlog.CtxErrorf(ctx, "find daily_automation_stats by date failed, err: %v", err)
		return nil, err
	}
	return &result, nil
}

// GetLatestStats 获取最新的统计记录
func (self *DailyAutomationStatsDao) GetLatestStats(ctx context.Context) (*DailyAutomationStatsModel, error) {
	opts := options.FindOne().SetSort(bson.M{"date": -1})
	var result DailyAutomationStatsModel
	err := probeDatabase.
		Collection(TableNameDailyAutomationStats).
		FindOne(ctx, bson.M{}, opts).
		Decode(&result)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}
