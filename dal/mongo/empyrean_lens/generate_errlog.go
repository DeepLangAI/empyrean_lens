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

var TableNameGenerateErrLog = "generate_err_log"

type GenerateErrLogModel struct {
	UserId        string    `bson:"user_id" json:"user_id"`
	TraceId       string    `bson:"trace_id" json:"trace_id"`
	EntryId       string    `bson:"entry_id" json:"entry_id"`
	Ip            string    `bson:"ip" json:"ip"`
	IpRegion      string    `bson:"ip_region" json:"ip_region"`
	FailureReason string    `bson:"failure_reason" json:"failure_reason"`
	Date          string    `bson:"date" json:"date"`
	Time          time.Time `bson:"time" json:"time"`
	UpdateTime    time.Time `bson:"update_time" json:"update_time"`
}
type GenerateErrLogDao struct{}

var generateErrLogDao *GenerateErrLogDao
var generateErrlogInitOnce sync.Once

func NewGeneratorErrlogModelDao() *GenerateErrLogDao {
	generateErrlogInitOnce.Do(func() {
		generateErrLogDao = &GenerateErrLogDao{}
	})
	return generateErrLogDao
}

func (self *GenerateErrLogDao) Save(ctx context.Context, model GenerateErrLogModel) error {
	_, err := probeDatabase.Collection(TableNameGenerateErrLog).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}

func (self *GenerateErrLogDao) SaveBatch(ctx context.Context, models []GenerateErrLogModel) error {
	// 检查输入数据是否为空
	if len(models) == 0 {
		hlog.CtxWarnf(ctx, "no models provided for batch insert or update")
		return nil
	}

	// 构造批量操作
	var operations []mongo.WriteModel
	for _, model := range models {
		// 定义过滤条件: user_id 和 time
		filter := bson.M{
			"user_id": model.UserId,
			"time":    model.Time,
		}

		update := bson.D{
			{Key: "$set", Value: model},
		}

		// 使用 Upsert 选项：如果不存在则插入
		updateModel := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)

		operations = append(operations, updateModel)
	}

	// 批量执行操作
	collection := probeDatabase.Collection(TableNameGenerateErrLog)
	bulkOption := options.BulkWrite().SetOrdered(false) // 设置 unordered 模式以提高性能
	result, err := collection.BulkWrite(ctx, operations, bulkOption)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo bulk write error: %v", err)
		return err
	}

	// 输出操作结果
	hlog.CtxInfof(ctx, "successfully inserted %d documents, matched %d documents, modified %d documents. TableName",
		result.UpsertedCount, result.MatchedCount, result.ModifiedCount, TableNameGenerateErrLog)
	return nil
}

func (self *GenerateErrLogDao) GetGenerateErrlogByTime(ctx context.Context, dateStr string) ([]GenerateErrLogModel, error) {
	generateErrLogInfos := make([]GenerateErrLogModel, 0)
	collection := probeDatabase.Collection(TableNameGenerateErrLog)
	var filter bson.M
	if dateStr == "" {
		filter = bson.M{}
	} else {
		filter = bson.M{"date": dateStr}
	}

	findOptions := options.Find().SetSort(bson.M{"update_time": -1})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo find error:%v", err)
		return generateErrLogInfos, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var generateErrLogInfo GenerateErrLogModel
		err := cursor.Decode(&generateErrLogInfo)
		if err != nil {
			hlog.CtxErrorf(ctx, "mongo decode error:%v", err)
			return generateErrLogInfos, err
		}
		generateErrLogInfos = append(generateErrLogInfos, generateErrLogInfo)
	}
	return generateErrLogInfos, nil
}

func (self *GenerateErrLogDao) CountGenerateErrLogsByTimeRange(ctx context.Context, startDate, endDate string) (map[string]int64, error) {
	collection := probeDatabase.Collection(TableNameGenerateErrLog)

	// 定义开始时间的过滤条件
	conditions := []bson.M{}
	if startDate != "" {
		conditions = append(conditions, bson.M{"date": bson.M{"$gte": startDate}})
	}
	if endDate != "" {
		conditions = append(conditions, bson.M{"date": bson.M{"$lte": endDate}})
	}
	filter := bson.M{}
	if len(conditions) > 0 {
		filter = bson.M{"$and": conditions}
	}

	// 聚合管道
	pipeline := mongo.Pipeline{
		{{"$match", filter}}, // 过滤条件
		{{"$group", bson.M{
			"_id":   "$date",           // 按 date 分组
			"count": bson.M{"$sum": 1}, // 统计每组条目数
		}}},
		{{"$sort", bson.M{"_id": -1}}}, // 按日期升序排序
	}

	// 执行聚合查询
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo aggregation error:%v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// 解析聚合结果
	rawResults := make(map[string]int64)
	minDate := ""
	for cursor.Next(ctx) {
		var item struct {
			Date  string `bson:"_id"`   // 分组字段对应 _id
			Count int64  `bson:"count"` // 统计结果
		}
		if err := cursor.Decode(&item); err != nil {
			hlog.CtxErrorf(ctx, "mongo decode error:%v", err)
			return nil, err
		}
		rawResults[item.Date] = item.Count
		if minDate == "" || item.Date < minDate {
			minDate = item.Date
		}
	}

	// 检查游标迭代错误
	if err := cursor.Err(); err != nil {
		hlog.CtxErrorf(ctx, "cursor iteration error:%v", err)
		return nil, err
	}

	result := make(map[string]int64)
	// 获取当前日期
	if endDate == "" {
		endDate = time.Now().Format(consts.DateTemplate)
	}
	if startDate == "" {
		startDate = minDate
	}
	currentDate, err := time.ParseInLocation(consts.DateTemplate, endDate, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "Invalid endDate format: %v", err)
		return nil, err
	}

	// 补全缺失的日期
	startTime, err := time.ParseInLocation(consts.DateTemplate, startDate, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "Invalid startDate format: %v", err)
		return nil, err
	}

	// 从 startDate 到当前日期之间的每一天
	for startTime.Before(currentDate) || startTime.Equal(currentDate) {
		dateStr := startTime.Format(consts.DateTemplate)
		if count, exists := rawResults[dateStr]; exists {
			result[dateStr] = count
		} else {
			result[dateStr] = 0
		}
		startTime = startTime.AddDate(0, 0, 1) // 增加一天
	}

	return result, nil
}
