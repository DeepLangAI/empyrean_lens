package empyrean_lens

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sync"
	"time"
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
	_, err := probeDatabase.Collection(TableNameApiTestDetail).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}
