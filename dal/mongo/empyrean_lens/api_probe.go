package empyrean_lens

import (
	"context"
	"empyrean_lens/consts"
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
		Find(ctx, bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts.StatusValid})
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

func (self *ApiProbeLogModelDao) Tidy(ctx context.Context) error {
	// 找到所有scene为场景1或场景2或场景3的，删除
	_, err := probeDatabase.
		Collection(TableNameApiProbeLog).
		DeleteMany(ctx, bson.M{"scene": bson.M{"$in": []string{"场景1", "场景2", "场景3"}}})
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo delete many error:%v", err)
		return err
	}
	// 找到所有日期<2024-07-31的，删除
	_, err = probeDatabase.
		Collection(TableNameApiProbeLog).
		DeleteMany(ctx, bson.M{"create_time": bson.M{"$lt": time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC)}})
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo delete many error:%v", err)
		return err
	}
	return err
}
