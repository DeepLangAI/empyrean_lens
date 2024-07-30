package mongo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameApiProbeLog = "api_probe"

type ApiProbeLogModel struct {
	//Id           primitive.ObjectID `bson:"_id" json:"id"`
	Scene   string  `bson:"scene"`
	Api     string  `bson:"api"`
	Host    string  `bson:"host"`
	IsCore  bool    `bson:"is_core"`
	Success bool    `bson:"success"`
	Correct bool    `bson:"correct"`
	Cost    float64 `bson:"cost"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}
type ApiProbeLogModelDao struct{}

var apiProbeLogModelDao *ApiProbeLogModelDao
var apiProbeInitOnce sync.Once

func NewApiProbeLogModelDao() *ApiProbeLogModelDao {
	apiProbeInitOnce.Do(func() {
		apiProbeLogModelDao = &ApiProbeLogModelDao{}
	})
	return apiProbeLogModelDao
}

func (self *ApiProbeLogModelDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameApiProbeLog).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo drop table error:%v", err)
	}
	return err
}

func (self *ApiProbeLogModelDao) Save(ctx context.Context, model ApiProbeLogModel) error {
	_, err := probeDatabase.Collection(TableNameApiProbeLog).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}

func (self *ApiProbeLogModelDao) FindTimespanApiProbeLog(ctx context.Context, timeBegin, timeEnd time.Time) ([]ApiProbeLogModel, error) {
	/*
	   查询指定时间范围内的探针日志,
	   查询条件为create_time在[timeBegin, timeEnd)区间，且status为StatusValid
	*/
	var result []ApiProbeLogModel
	cur, err := probeDatabase.
		Collection(TableNameApiProbeLog).
		Find(ctx, bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": StatusValid})
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo find error:%v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "mongo all error:%v", err)
		return nil, err
	}
	return result, nil
}
