package empyrean_lens

import (
	"context"
	"empyrean_lens/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

var TableNameAppCrash = "app_crash"

type AppCrashModel struct {
	Time              time.Time `bson:"time"`
	PlatformType      string    `bson:"platform_type"`
	ErrorCount        int32     `bson:"error_count"`
	LaunchCount       int32     `bson:"launch_count"`
	AffectedUserCount int32     `bson:"affected_user_count"`
	ActiveUserCount   int32     `bson:"active_user_count"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type AppCrashModelDao struct{}

var appCrashModelDao *AppCrashModelDao
var appCrashInitOnce sync.Once

func NewAppCrashModelDao() *AppCrashModelDao {
	appCrashInitOnce.Do(func() {
		appCrashModelDao = &AppCrashModelDao{}
	})
	return appCrashModelDao
}

func (self *AppCrashModelDao) Save(ctx context.Context, model AppCrashModel) error {
	_, err := probeDatabase.Collection(TableNameAppCrash).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}

func (self *AppCrashModelDao) SaveBatch(ctx context.Context, models []AppCrashModel) error {
	// 检查输入数据是否为空
	if len(models) == 0 {
		hlog.CtxWarnf(ctx, "no models provided for batch insert or update")
		return nil
	}

	// 构造批量操作
	var operations []mongo.WriteModel
	for _, model := range models {
		// 定义过滤条件: time 和 platform_type
		filter := bson.M{
			"time":          model.Time,
			"platform_type": model.PlatformType,
		}

		update := bson.D{
			{Key: "$set", Value: model},
		}

		updateModel := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)
		operations = append(operations, updateModel)
	}
	// 执行批量操作
	collection := probeDatabase.Collection(TableNameAppCrash)
	bulkOption := options.BulkWrite().SetOrdered(false)
	result, err := collection.BulkWrite(ctx, operations, bulkOption)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
		return err
	}
	hlog.CtxInfof(ctx, "successfully upserted %d documents, matched %d documents, modified %d documents, TableName: %s",
		result.InsertedCount, result.MatchedCount, result.ModifiedCount, TableNameAppCrash)
	return nil
}

func (self *AppCrashModelDao) GetAppCrashInfoByTime(ctx context.Context, beginTime, endTime time.Time) ([]AppCrashModel, error) {
	var result []AppCrashModel
	cur, err := probeDatabase.
		Collection(TableNameAppCrash).
		Find(ctx, bson.M{
			"time": bson.M{
				"$gte": beginTime,
				"$lt":  endTime,
			},
			"status": consts.StatusValid,
		})
	if err != nil {
		hlog.CtxErrorf(ctx, "find app crash model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find app crash model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}
