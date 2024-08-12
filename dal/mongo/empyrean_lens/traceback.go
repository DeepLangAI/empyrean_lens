package empyrean_lens

import (
	"context"
	"empyrean_lens/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

const TableNameTracebackLog = "traceback"

type TracebackLogModel struct {
	//Id           primitive.ObjectID `bson:"_id" json:"id"`
	ExcInfo   string            `bson:"exc_info" json:"exc_info"`
	Msg       string            `bson:"msg" json:"msg"`
	TraceId   string            `bson:"trace_id" json:"trace_id"`
	UserId    string            `bson:"user_id" json:"user_id"`
	Time      time.Time         `bson:"time" json:"time"`
	OriginLog map[string]string `bson:"origin_log" json:"origin_log"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}
type TracebackLogModelDao struct{}

var tracebackLogModelDao *TracebackLogModelDao
var tracebackInitOnce sync.Once

func NewTracebackLogModelDao() *TracebackLogModelDao {
	tracebackInitOnce.Do(func() {
		tracebackLogModelDao = &TracebackLogModelDao{}
	})
	return tracebackLogModelDao
}

func (self *TracebackLogModelDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameTracebackLog).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo drop table error:%v", err)
	}
	return err
}

func (self *TracebackLogModelDao) Save(ctx context.Context, model TracebackLogModel) error {
	_, err := probeDatabase.Collection(TableNameTracebackLog).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo insert one error:%v", err)
	}
	return err
}

func (self *TracebackLogModelDao) FindTimespanTracebackLog(ctx context.Context, timeBegin, timeEnd time.Time) ([]TracebackLogModel, error) {
	/*
	   查询指定时间范围内的探针日志,
	   查询条件为time在[timeBegin, timeEnd)区间，且status为StatusValid
	*/
	var result []TracebackLogModel
	cur, err := probeDatabase.
		Collection(TableNameTracebackLog).
		Find(ctx, bson.M{"time": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts.StatusValid})
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

func (self *TracebackLogModelDao) CreateOrUpdate(ctx context.Context, update TracebackLogModel) error {
	_, err := probeDatabase.Collection(TableNameTracebackLog).
		UpdateOne(
			ctx,
			bson.M{"time": update.Time, "status": consts.StatusValid},
			bson.M{"$set": update},
			options.Update().SetUpsert(true),
		)
	if err != nil {
		hlog.CtxErrorf(ctx, "update traceback log model failed, err: %v", err)
		return err
	}
	return nil
}

func (self *TracebackLogModelDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameTracebackLog).
		DeleteMany(ctx, bson.M{"time": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete traceback log model failed, err: %v", err)
		return err
	}
	return nil

}
