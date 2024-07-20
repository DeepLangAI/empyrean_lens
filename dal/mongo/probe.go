package mongo

import (
	"context"
	"empyrean_lens/conf"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sync"
	"time"
)

var TableNameProbeLog = "probe" + conf.GetConfig().Mongo.Shadow

const (
	StatusValid = 0 //有效
	StatusDel   = 1 // 删除
)
const (
	RESULT_FAIL    = 0
	RESULT_SUCCESS = 1
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

func (self *ProbeLogModelDao) Save(ctx context.Context, model ProbeLogModel) error {
	_, err := probeDatabase.Collection(TableNameProbeLog).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}
