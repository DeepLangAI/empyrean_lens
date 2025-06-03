package empyrean_lens

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var TableNameApiPerformanceLatestVersionTrend = "api_performance_latest_version_trend"

// ApiPerformanceLatestVersionTrend 最新版本API性能趋势数据
type ApiPerformanceLatestVersionTrend struct {
	Date      string    `bson:"date"`       // 日期
	System    string    `bson:"system"`     // 系统类型
	Version   string    `bson:"version"`    // 版本号
	TimePoint string    `bson:"time_point"` // 时间点
	CreatedAt time.Time `bson:"created_at"` // 创建时间
	UpdatedAt time.Time `bson:"updated_at"` // 更新时间
	Buckets   []struct {
		TimeInterval string  `bson:"time_interval"` // 时间区间
		Percentage   float64 `bson:"percentage"`    // 占比
	} `bson:"buckets"` // 各时间区间的统计数据
	AvgDuration float64 `bson:"avg_duration"` // 平均耗时（新增）
}

// ApiPerformanceLatestVersionTrendDao 最新版本API性能趋势数据访问对象
type ApiPerformanceLatestVersionTrendDao struct{}

var apiPerformanceLatestVersionTrendDao *ApiPerformanceLatestVersionTrendDao
var apiPerformanceLatestVersionTrendDaoInitOnce sync.Once

// NewApiPerformanceLatestVersionTrendDao 创建最新版本API性能趋势数据访问对象
func NewApiPerformanceLatestVersionTrendDao() *ApiPerformanceLatestVersionTrendDao {
	apiPerformanceLatestVersionTrendDaoInitOnce.Do(func() {
		apiPerformanceLatestVersionTrendDao = &ApiPerformanceLatestVersionTrendDao{}
	})
	return apiPerformanceLatestVersionTrendDao
}

// EnsureCollection 确保集合存在
func (d *ApiPerformanceLatestVersionTrendDao) EnsureCollection(ctx context.Context) error {
	// 检查集合是否存在
	collections, err := probeDatabase.ListCollectionNames(ctx, bson.M{"name": TableNameApiPerformanceLatestVersionTrend})
	if err != nil {
		return err
	}

	// 如果集合不存在，创建它
	if len(collections) == 0 {
		err := probeDatabase.CreateCollection(ctx, TableNameApiPerformanceLatestVersionTrend)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpsertTrend 更新或插入趋势数据
func (d *ApiPerformanceLatestVersionTrendDao) UpsertTrend(ctx context.Context, trend *ApiPerformanceLatestVersionTrend) error {
	filter := bson.M{
		"date":       trend.Date,
		"system":     trend.System,
		"time_point": trend.TimePoint,
		"version":    trend.Version,
	}
	update := bson.M{
		"$set": bson.M{
			"buckets":      trend.Buckets,
			"updated_at":   trend.UpdatedAt,
			"avg_duration": trend.AvgDuration,
		},
		"$setOnInsert": bson.M{
			"created_at": trend.CreatedAt,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err := probeDatabase.Collection(TableNameApiPerformanceLatestVersionTrend).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		hlog.CtxErrorf(ctx, "upsert api performance latest version trend failed, err: %v", err)
		return err
	}
	return nil
}

// GetLatestVersion 获取指定系统的最新版本
func (d *ApiPerformanceLatestVersionTrendDao) GetLatestVersion(ctx context.Context, system string) (string, error) {
	filter := bson.M{
		"system": system,
	}
	opts := options.Find().SetProjection(bson.M{"version": 1})

	cursor, err := probeDatabase.Collection(TableNameApiPerformanceLatestVersionTrend).Find(ctx, filter, opts)
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	var versions []struct {
		Version string `bson:"version"`
	}
	if err := cursor.All(ctx, &versions); err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", nil
	}

	var latestVersion string
	for _, v := range versions {
		if latestVersion == "" {
			latestVersion = v.Version
			continue
		}

		if system == "Android" {
			// Android版本直接比较数字大小
			latestNum, _ := strconv.Atoi(latestVersion)
			currentNum, _ := strconv.Atoi(v.Version)
			if currentNum > latestNum {
				latestVersion = v.Version
			}
		} else if system == "iOS" {
			// iOS版本需要分段比较
			latestParts := strings.Split(latestVersion, ".")
			currentParts := strings.Split(v.Version, ".")

			// 确保两个版本号都有相同数量的段
			maxLen := len(latestParts)
			if len(currentParts) > maxLen {
				maxLen = len(currentParts)
			}

			// 比较每一段
			for i := 0; i < maxLen; i++ {
				// 如果当前版本号段数更多，且前面的段都相同，则当前版本更新
				if i >= len(latestParts) {
					latestVersion = v.Version
					break
				}
				// 如果比较版本号段数更多，且前面的段都相同，则保持当前最新版本
				if i >= len(currentParts) {
					break
				}

				latestNum, _ := strconv.Atoi(latestParts[i])
				currentNum, _ := strconv.Atoi(currentParts[i])
				if currentNum > latestNum {
					latestVersion = v.Version
					break
				} else if currentNum < latestNum {
					break
				}
			}
		}
	}

	return latestVersion, nil
}

// GetTrendByDateRange 根据日期范围获取趋势数据
func (d *ApiPerformanceLatestVersionTrendDao) GetTrendByDateRange(ctx context.Context, startTime, endTime time.Time, system string) ([]*ApiPerformanceLatestVersionTrend, error) {
	// 首先获取该系统的最新版本
	latestVersion, err := d.GetLatestVersion(ctx, system)
	if err != nil {
		return nil, err
	}

	// 构建查询条件
	filter := bson.M{
		"date": bson.M{
			"$gte": startTime.Format("2006-01-02"),
			"$lte": endTime.Format("2006-01-02"),
		},
		"system":  system,
		"version": latestVersion,
	}
	opts := options.Find().SetSort(bson.D{{Key: "time_point", Value: 1}})

	cursor, err := probeDatabase.Collection(TableNameApiPerformanceLatestVersionTrend).Find(ctx, filter, opts)
	if err != nil {
		hlog.CtxErrorf(ctx, "find api performance latest version trend failed, err: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var trends []*ApiPerformanceLatestVersionTrend
	if err := cursor.All(ctx, &trends); err != nil {
		hlog.CtxErrorf(ctx, "decode api performance latest version trend failed, err: %v", err)
		return nil, err
	}

	return trends, nil
}

// GetLatestCreatedAtBySystem 获取指定系统的最新记录时间
func (d *ApiPerformanceLatestVersionTrendDao) GetLatestCreatedAtBySystem(ctx context.Context, system string) (time.Time, error) {
	var trend ApiPerformanceLatestVersionTrend
	opts := options.FindOne().SetSort(bson.M{"time_point": -1})
	filter := bson.M{"system": system}
	err := probeDatabase.Collection(TableNameApiPerformanceLatestVersionTrend).FindOne(ctx, filter, opts).Decode(&trend)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return time.Time{}, fmt.Errorf("no records found for system: %s", system)
		}
		return time.Time{}, err
	}
	// 解析time_point字符串为北京时间
	loc, _ := time.LoadLocation("Asia/Shanghai")
	timePoint, err := time.ParseInLocation("2006-01-02 15:04:05", trend.TimePoint, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time_point failed: %v", err)
	}
	return timePoint, nil
}
