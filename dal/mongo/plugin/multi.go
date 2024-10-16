package plugin

import (
	"context"
	"empyrean_lens/consts"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const TableNameMulti = "multi"

type ArticleEntry struct {
	EntryId   string           `json:"entry_id" bson:"entry_id"`
	EntryType consts.EntryType `json:"entry_type" bson:"entry_type"`
}

type MultiModel struct {
	Id          primitive.ObjectID `bson:"_id" json:"id"`
	UserID      string             `json:"user_id" bson:"user_id" validate:"required"`
	ArticleList []ArticleEntry     `json:"article_list" bson:"article_list"`
	Title       string             `json:"title" bson:"title"`

	CreateTime time.Time `json:"create_time" bson:"create_time"`
	UpdateTime time.Time `json:"update_time" bson:"update_time"`
	IsDeleted  bool      `json:"is_deleted" bson:"is_deleted"`
}

var multiDao *MultiDao

type MultiDao struct {
}

var MultiDaoOnce sync.Once

func NewMultiDao() *MultiDao {
	MultiDaoOnce.Do(func() {
		multiDao = &MultiDao{}
	})
	return multiDao
}

func (d *MultiDao) FindMultiByUserIdAndCreateTime(ctx context.Context, userId string, startTime, endTime time.Time) ([]MultiModel, error) {
	var result []MultiModel

	filter := bson.M{"is_delete": false}
	if userId != "" {
		filter["user_id"] = userId
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		filter["create_time"] = bson.M{"$gte": startTime, "$lt": endTime}
	}
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, filter)

	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByUserIdAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByUserIdAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return result, nil
}
