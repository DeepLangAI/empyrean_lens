package empyrean_lens

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var TableNameApiTestDetail = "api_test_detail"

type ApiTestDetailModel struct {
	Id              primitive.ObjectID `bson:"_id"`
	TraceId         string             `bson:"trace_id" json:"trace_id"`
	EntryId         string             `bson:"entry_id" json:"entry_id"`
	RequestContent  RequestContent     `bson:"request_content" json:"request_content"`
	ResponseContent ResponseContent    `bson:"response_content" json:"response_content"`
	CostTime        int64              `bson:"cost_time" json:"cost_time"`
	ApiName         string             `bson:"api_name" json:"api_name"`
	CreatedTime     time.Time          `bson:"created_time" json:"created_time"`
}

type RequestContent struct {
	Headers map[string]string `bson:"headers" json:"headers"`
	Body    interface{}       `bson:"body" json:"body"`
	Method  string            `bson:"method" json:"method"`
	Url     string            `bson:"url" json:"url"`
}

type ResponseContent struct {
	StatusCode int               `bson:"status_code" json:"status_code"`
	Headers    map[string]string `bson:"headers" json:"headers"`
	Body       interface{}       `bson:"body" json:"body"`
	Error      string            `bson:"error" json:"error"`
}

type ApiTestDetailDao struct{}

var apiTestDetailDao *ApiTestDetailDao
var apiTestDetailDaoInitOnce sync.Once

func NewApiTestDetailDao() *ApiTestDetailDao {
	apiTestDetailDaoInitOnce.Do(func() {
		apiTestDetailDao = &ApiTestDetailDao{}
	})
	return apiTestDetailDao
}

func (self *ApiTestDetailDao) Save(ctx context.Context, model ApiTestDetailModel) error {

	// 检查是否已存在相同的 trace_id
	count, err := probeDatabase.Collection(TableNameApiTestDetail).CountDocuments(ctx, bson.M{"trace_id": model.TraceId})
	if err != nil {
		hlog.CtxErrorf(ctx, "check duplicate trace_id error:%v", err)
		return err
	}
	if count > 0 {
		hlog.CtxErrorf(ctx, "check duplicate trace_id error:%v", model.TraceId)
		return fmt.Errorf("duplicate trace_id: %s", model.TraceId)
	}

	// 插入新记录
	_, err = probeDatabase.Collection(TableNameApiTestDetail).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}

// Aggregate 执行聚合查询
func (self *ApiTestDetailDao) Aggregate(ctx context.Context, pipeline interface{}) (*mongo.Cursor, error) {
	return probeDatabase.Collection(TableNameApiTestDetail).Aggregate(ctx, pipeline)
}

// FindByApiTypeAndDate 根据API类型和日期查询测试详情
func (self *ApiTestDetailDao) FindByApiTypeAndDate(ctx context.Context, apiType string, date time.Time) ([]ApiTestDetailModel, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	filter := bson.M{
		"api_name": apiType,
		"created_time": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
		"response_content.status_code": bson.M{
			"$ne": 200, // 只查询非200状态码的记录
		},
	}

	cursor, err := probeDatabase.Collection(TableNameApiTestDetail).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []ApiTestDetailModel
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// GetEarliestRecordDate 获取最早的记录时间
func (self *ApiTestDetailDao) GetEarliestRecordDate(ctx context.Context) (time.Time, error) {
	opts := options.FindOne().SetSort(bson.D{{"created_time", 1}})
	var result ApiTestDetailModel
	err := probeDatabase.Collection(TableNameApiTestDetail).FindOne(ctx, bson.M{}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return time.Now(), nil // 如果没有记录，返回当前时间
		}
		return time.Time{}, err
	}
	return result.CreatedTime, nil
}
