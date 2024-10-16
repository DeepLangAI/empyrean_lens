package plugin

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebReader struct {
	ID     primitive.ObjectID `bson:"_id" json:"_id"`
	URL    string             `bson:"url" json:"url" validate:"required"`
	UserID string             `bson:"user_id" json:"user_id" validate:"required,min=5,max=64"`
	Title  string             `bson:"title" json:"title" default:""`
	// Content     string             `bson:"content" json:"content" default:""`
	// HTML        string             `bson:"html" json:"html" default:""`
	IsDeleted   bool      `bson:"is_deleted" json:"is_deleted" default:"false"`
	SafeStatus  bool      `bson:"safe_status" json:"safe_status" default:"true"`
	IsGenerated bool      `bson:"is_generated" json:"is_generated" default:"false"`
	CreateTime  time.Time `bson:"create_time" json:"create_time"`
	UpdateTime  time.Time `bson:"update_time" json:"update_time"`
}

const TableNameWebReader = "web_reader"

var webReaderDao *WebReaderDao

type WebReaderDao struct {
}

var WebReaderDaoOnce sync.Once

func NewWebReaderDao() *WebReaderDao {
	WebReaderDaoOnce.Do(func() {
		webReaderDao = &WebReaderDao{}
	})
	return webReaderDao
}

func (d *WebReaderDao) FindWebReaderByUserIdAndCreateTime(ctx context.Context, userId string, startTime, endTime time.Time) ([]WebReader, error) {
	var result []WebReader

	filter := bson.M{"is_delete": false}
	if userId != "" {
		filter["user_id"] = userId
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		filter["create_time"] = bson.M{"$gte": startTime, "$lt": endTime}
	}
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter)

	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByUserIdAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByUserIdAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return result, nil
}
