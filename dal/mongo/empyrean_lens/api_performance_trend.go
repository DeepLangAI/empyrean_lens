package empyrean_lens

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var TableNameApiPerformanceTrend = "api_performance_trend"

// ApiPerformanceTrend 存储API性能趋势数据
type ApiPerformanceTrend struct {
	Date      string `bson:"date"`       // 日期
	System    string `bson:"system"`     // 系统（iOS/Android）
	TimePoint string `bson:"time_point"` // 时间点（HH:00）
	Buckets   []struct {
		TimeInterval string  `bson:"time_interval"` // 耗时区间
		Percentage   float64 `bson:"percentage"`    // 占比
	} `bson:"buckets"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// ApiPerformanceTrendDao 处理API性能趋势数据的数据库操作
type ApiPerformanceTrendDao struct{}

var apiPerformanceTrendDao *ApiPerformanceTrendDao
var apiPerformanceTrendDaoInitOnce sync.Once

// NewApiPerformanceTrendDao 创建新的ApiPerformanceTrendDao实例
func NewApiPerformanceTrendDao() *ApiPerformanceTrendDao {
	apiPerformanceTrendDaoInitOnce.Do(func() {
		apiPerformanceTrendDao = &ApiPerformanceTrendDao{}
	})
	return apiPerformanceTrendDao
}

// EnsureCollection 确保数据库集合存在
func (d *ApiPerformanceTrendDao) EnsureCollection(ctx context.Context) error {
	// 检查集合是否存在
	collections, err := probeDatabase.ListCollectionNames(ctx, bson.M{"name": TableNameApiPerformanceTrend})
	if err != nil {
		return err
	}

	// 如果集合不存在，创建它
	if len(collections) == 0 {
		err := probeDatabase.CreateCollection(ctx, TableNameApiPerformanceTrend)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpsertTrend 更新或插入趋势数据
func (d *ApiPerformanceTrendDao) UpsertTrend(ctx context.Context, trend *ApiPerformanceTrend) error {
	filter := bson.M{
		"date":       trend.Date,
		"system":     trend.System,
		"time_point": trend.TimePoint,
	}
	update := bson.M{
		"$set": bson.M{
			"updated_at": trend.UpdatedAt,
			"buckets":    trend.Buckets,
		},
		"$setOnInsert": bson.M{
			"created_at": trend.CreatedAt,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := probeDatabase.Collection(TableNameApiPerformanceTrend).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		hlog.CtxErrorf(ctx, "upsert api performance trend failed, err: %v", err)
		return err
	}
	return nil
}

// GetTrendByDateRange 获取指定日期范围内的趋势数据
func (d *ApiPerformanceTrendDao) GetTrendByDateRange(ctx context.Context, startDate, endDate string, system string) ([]*ApiPerformanceTrend, error) {
	filter := bson.M{
		"date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
		"system": system,
	}
	cursor, err := probeDatabase.Collection(TableNameApiPerformanceTrend).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "find api performance trend failed, err: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var trends []*ApiPerformanceTrend
	if err := cursor.All(ctx, &trends); err != nil {
		hlog.CtxErrorf(ctx, "decode api performance trend failed, err: %v", err)
		return nil, err
	}
	return trends, nil
}

// GetLatestVersion 获取指定系统的最新版本
func (d *ApiPerformanceTrendDao) GetLatestVersion(ctx context.Context, system string) (string, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"system": system}},
		{"$sort": bson.M{"version": -1}},
		{"$limit": 1},
		{"$project": bson.M{"version": 1, "_id": 0}},
	}

	cursor, err := probeDatabase.Collection(TableNameApiPerformanceTrend).Aggregate(ctx, pipeline)
	if err != nil {
		hlog.CtxErrorf(ctx, "aggregate api performance trend failed, err: %v", err)
		return "", err
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "decode aggregate result failed, err: %v", err)
		return "", err
	}

	if len(result) == 0 {
		return "", nil
	}
	return result[0]["version"].(string), nil
}

func (d *ApiPerformanceTrendDao) GetLatestCreatedAtBySystem(ctx context.Context, system string) (time.Time, error) {
	var trend ApiPerformanceTrend
	opts := options.FindOne().SetSort(bson.M{"time_point": -1})
	filter := bson.M{"system": system}
	err := probeDatabase.Collection(TableNameApiPerformanceTrend).FindOne(ctx, filter, opts).Decode(&trend)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return time.Time{}, fmt.Errorf("no records found for system: %s", system)
		}
		return time.Time{}, err
	}
	// 解析time_point字符串为time.Time
	timePoint, err := time.Parse("2006-01-02 15:04:05", trend.TimePoint)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time_point failed: %v", err)
	}
	return timePoint, nil
}
