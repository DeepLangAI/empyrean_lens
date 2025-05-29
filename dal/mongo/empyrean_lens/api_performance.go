package empyrean_lens

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var TableNameApiPerformance = "api_performance_stats"

// ApiPerformanceStats 表示新内容形态页加载性能数据
type ApiPerformanceStats struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	System    string             `bson:"system"`
	Version   string             `bson:"version"`
	Date      string             `bson:"date"`
	CreatedAt time.Time          `bson:"create_time"`
	UpdatedAt time.Time          `bson:"update_time"`

	Summary struct {
		TotalDuration float64 `bson:"total_duration"`
		TotalRequests int64   `bson:"total_requests"`
		AvgDuration   float64 `bson:"avg_duration"`
	} `bson:"summary"`

	Buckets []struct {
		TimeInterval string `bson:"time_interval"`
		RequestCount int64  `bson:"request_count"`
	} `bson:"buckets"`
}

// ApiPerformanceStatsDao 处理新内容形态页加载性能数据的数据库操作
type ApiPerformanceStatsDao struct{}

var apiPerformanceStatsDao *ApiPerformanceStatsDao
var apiPerformanceStatsDaoInitOnce sync.Once

// NewApiPerformanceStatsDao 创建新的ApiPerformanceStatsDao实例
func NewApiPerformanceStatsDao() *ApiPerformanceStatsDao {
	apiPerformanceStatsDaoInitOnce.Do(func() {
		apiPerformanceStatsDao = &ApiPerformanceStatsDao{}
	})
	return apiPerformanceStatsDao
}

// EnsureCollection 确保数据库集合存在
func (d *ApiPerformanceStatsDao) EnsureCollection(ctx context.Context) error {
	// 检查集合是否存在
	collections, err := probeDatabase.ListCollectionNames(ctx, bson.M{"name": TableNameApiPerformance})
	if err != nil {
		return err
	}

	// 如果集合不存在，创建它
	if len(collections) == 0 {
		err := probeDatabase.CreateCollection(ctx, TableNameApiPerformance)
		if err != nil {
			return err
		}
	}
	return nil
}

// DropTable 删除表
func (d *ApiPerformanceStatsDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameApiPerformance).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "drop api performance table failed, err: %v", err)
		return err
	}
	return nil
}

// RmRecentDays 删除最近几天的数据
func (d *ApiPerformanceStatsDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameApiPerformance).
		DeleteMany(ctx, bson.M{"create_time": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete api performance data failed, err: %v", err)
		return err
	}
	return nil
}

// UpsertStats 更新或插入性能统计数据
func (d *ApiPerformanceStatsDao) UpsertStats(ctx context.Context, stats *ApiPerformanceStats) error {
	filter := bson.M{
		"system":  stats.System,
		"version": stats.Version,
		"date":    stats.Date,
	}
	update := bson.M{
		"$set": bson.M{
			"update_time": stats.UpdatedAt,
			"summary":     stats.Summary,
			"buckets":     stats.Buckets,
		},
		"$setOnInsert": bson.M{
			"create_time": stats.CreatedAt,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := probeDatabase.Collection(TableNameApiPerformance).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		hlog.CtxErrorf(ctx, "upsert api performance stats failed, err: %v", err)
		return err
	}
	return nil
}

// GetStats 获取指定系统和版本的性能统计数据
func (d *ApiPerformanceStatsDao) GetStats(ctx context.Context, system, version string) (*ApiPerformanceStats, error) {
	filter := bson.M{
		"system":  system,
		"version": version,
	}
	var stats ApiPerformanceStats
	err := probeDatabase.Collection(TableNameApiPerformance).FindOne(ctx, filter).Decode(&stats)
	if err != nil {
		hlog.CtxErrorf(ctx, "get api performance stats failed, err: %v", err)
		return nil, err
	}
	return &stats, nil
}

// GetStatsByTimeRange 获取指定时间范围内的性能统计数据
func (d *ApiPerformanceStatsDao) GetStatsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*ApiPerformanceStats, error) {
	startDate := startTime.Format("2006-01-02")
	endDate := endTime.Format("2006-01-02")

	filter := bson.M{
		"date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	cursor, err := probeDatabase.Collection(TableNameApiPerformance).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "find api performance stats failed, err: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []*ApiPerformanceStats
	if err := cursor.All(ctx, &stats); err != nil {
		hlog.CtxErrorf(ctx, "decode api performance stats failed, err: %v", err)
		return nil, err
	}

	// 按日期和系统聚合数据
	dateSystemStats := make(map[string]*ApiPerformanceStats)
	for _, stat := range stats {
		// 跳过无效数据
		if stat == nil || stat.System == "" {
			continue
		}

		// 验证系统名称
		if stat.System != "iOS" && stat.System != "Android" {
			hlog.CtxWarnf(ctx, "跳过无效的系统名称: %s", stat.System)
			continue
		}

		// 使用日期和系统作为key
		key := fmt.Sprintf("%s_%s", stat.Date, stat.System)

		if existingStat, exists := dateSystemStats[key]; exists {
			// 如果新数据的时间更晚，则更新summary数据
			if stat.CreatedAt.After(existingStat.CreatedAt) {
				existingStat.Summary = stat.Summary
				existingStat.CreatedAt = stat.CreatedAt
			}

			// 合并时间区间数据
			for _, newBucket := range stat.Buckets {
				found := false
				for i, existingBucket := range existingStat.Buckets {
					if existingBucket.TimeInterval == newBucket.TimeInterval {
						// 累加请求数
						existingStat.Buckets[i].RequestCount += newBucket.RequestCount
						found = true
						break
					}
				}
				if !found {
					// 添加新的时间区间
					existingStat.Buckets = append(existingStat.Buckets, newBucket)
				}
			}
		} else {
			// 创建新的统计记录
			newStat := &ApiPerformanceStats{
				System:    stat.System,
				Version:   stat.Version,
				Date:      stat.Date,
				CreatedAt: stat.CreatedAt,
				UpdatedAt: stat.UpdatedAt,
				Summary:   stat.Summary,
				Buckets:   stat.Buckets,
			}
			dateSystemStats[key] = newStat
		}
	}

	// 将map转换为slice
	result := make([]*ApiPerformanceStats, 0, len(dateSystemStats))
	for _, stat := range dateSystemStats {
		result = append(result, stat)
	}

	// 按日期倒序和系统排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date == result[j].Date {
			return result[i].System < result[j].System
		}
		return result[i].Date > result[j].Date
	})

	return result, nil
}

func (d *ApiPerformanceStatsDao) GetLatestCreatedAtBySystem(ctx context.Context, system string) (time.Time, error) {
	var stat ApiPerformanceStats
	opts := options.FindOne().SetSort(bson.M{"create_time": -1})
	filter := bson.M{"system": system}
	err := probeDatabase.Collection(TableNameApiPerformance).FindOne(ctx, filter, opts).Decode(&stat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return time.Time{}, fmt.Errorf("no records found for system: %s", system)
		}
		return time.Time{}, err
	}
	return stat.CreatedAt, nil
}
