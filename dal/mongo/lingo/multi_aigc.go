package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameMultiAigc = "multi_aigc"

type MultiAigc struct {
	AigcType   int       `bson:"aigc_type" json:"aigc_type"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type MultiAigcDao struct{}

var multiAigcDao *MultiAigcDao
var multiAigcDaoInitOnce sync.Once

func NewMultiAigcModelDao() *MultiAigcDao {
	multiAigcDaoInitOnce.Do(func() {
		multiAigcDao = &MultiAigcDao{}
	})
	return multiAigcDao
}

func (self *MultiAigcDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]MultiAigc, error) {
	var result []MultiAigc
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	cur, err := lingoDatabase.
		Collection(TableNameMultiAigc).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "MultiAigc FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "MultiAigc FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
