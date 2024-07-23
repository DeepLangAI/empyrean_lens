package mongo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sync"
	"time"
)

const TableNameProbeLog = "probe"

const (
	StatusValid = 0 //有效
	StatusDel   = 1 // 删除
)
const (
	RESULT_NOT_STARTED = 0
	RESULT_SUCCESS     = 1
	RESULT_FAIL        = 2
)

type NodeDetail struct {
	Name   string  `bson:"name" json:"name"`
	Cost   float64 `bson:"cost" json:"cost"`
	Result int32   `bson:"result" json:"result"`
}

type ProbeLogModel struct {
	Id           primitive.ObjectID `bson:"_id" json:"id"`
	TotalNodes   int                `bson:"total_nodes" json:"total_nodes"`
	SuccessNodes int                `bson:"success_nodes" json:"success_nodes"`
	NodesDetail  []NodeDetail       `bson:"nodes_detail" json:"nodes_detail"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}
type ProbeLogModelDao struct{}

var probeLogModelDao *ProbeLogModelDao
var initOnce sync.Once

func NewProbeLogModelDao() *ProbeLogModelDao {
	initOnce.Do(func() {
		probeLogModelDao = &ProbeLogModelDao{}
	})
	return probeLogModelDao
}

func (self *ProbeLogModelDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameProbeLog).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo drop table error:%v", err)
	}
	return err
}

func (self *ProbeLogModelDao) Save(ctx context.Context, model ProbeLogModel) error {
	_, err := probeDatabase.Collection(TableNameProbeLog).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}

func (self *ProbeLogModelDao) FindTimespanProbeLog(ctx context.Context, timeBegin, timeEnd time.Time) ([]ProbeLogModel, error) {
	/*
	   查询指定时间范围内的探针日志,
	   查询条件为create_time在[timeBegin, timeEnd)区间，且status为StatusValid
	*/
	var result []ProbeLogModel
	cur, err := probeDatabase.
		Collection(TableNameProbeLog).
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
