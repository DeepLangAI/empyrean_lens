package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameMulti = "multi"

type Multi struct {
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type MultiDao struct{}

var multiDao *MultiDao
var multiDaoInitOnce sync.Once

func NewMultiModelDao() *MultiDao {
	multiDaoInitOnce.Do(func() {
		multiDao = &MultiDao{}
	})
	return multiDao
}

func (self *MultiDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]Multi, error) {
	var result []Multi
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	cur, err := lingoDatabase.
		Collection(TableNameMulti).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "Multi FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "Multi FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
