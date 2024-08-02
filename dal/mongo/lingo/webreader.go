package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameWebreader = "web_reader"

type Webreader struct {
	CopyFromUrlId string    `bson:"copy_from_url_id" json:"copy_from_url_id"`
	CreateTime    time.Time `bson:"create_time" json:"create_time"`
	UpdateTime    time.Time `bson:"update_time" json:"update_time"`
}

type WebreaderDao struct{}

var webreaderDao *WebreaderDao
var webreaderDaoInitOnce sync.Once

func NewWebreaderModelDao() *WebreaderDao {
	webreaderDaoInitOnce.Do(func() {
		webreaderDao = &WebreaderDao{}
	})
	return webreaderDao
}

func (self *WebreaderDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time, prebuild bool) ([]Webreader, error) {
	var result []Webreader
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	if !prebuild {
		filter["copy_from_url_id"] = bson.M{"$eq": ""}
	}
	cur, err := lingoDatabase.
		Collection(TableNameWebreader).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "Webreader FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "Webreader FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
