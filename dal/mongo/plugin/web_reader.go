package plugin

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebReader struct {
	ID          primitive.ObjectID `bson:"_id" json:"_id"`
	URL         string             `bson:"url" json:"url" validate:"required"`
	UserID      string             `bson:"user_id" json:"user_id" validate:"required,min=5,max=64"`
	Title       string             `bson:"title" json:"title" default:""`
	Status      int                `bson:"status" json:"status"`
	ChannelType int                `bson:"channel_type" json:"channel_type"`
	IsDeleted   bool               `bson:"is_deleted" json:"is_deleted" default:"false"`
	CreateTime  time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime  time.Time          `bson:"update_time" json:"update_time"`
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

func (d *WebReaderDao) FindWebReaderByCreateTime(ctx context.Context, startTime, endTime time.Time) ([]WebReader, error) {
	var res []WebReader

	filter := bson.M{"is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *WebReaderDao) FindWebReaderByUserIdAndCreateTime(ctx context.Context, userId string, startTime, endTime time.Time) ([]WebReader, error) {
	var res []WebReader

	filter := bson.M{"user_id": userId, "is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByUserIdAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByUserIdAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *WebReaderDao) FindWebReaderByIdAndCreateTime(ctx context.Context, id string, startTime, endTime time.Time) (WebReader, error) {
	var res WebReader

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": _id, "is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByIdAndCreateTime] mongo find error:%+v", err)
		return res, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		err = cur.Decode(&res)
		if err != nil {
			hlog.CtxErrorf(ctx, "db error, [FindWebReaderByIdAndCreateTime], err:%v", err)
			return res, err
		}
	}
	if res.ID.IsZero() {
		hlog.CtxInfof(ctx, "[FindWebReaderByIdAndCreateTime] mongo find nil: id=%s", id)
		return res, errors.New("not found")
	}
	return res, nil
}

func (d *WebReaderDao) FindWebReaderByWebReaderURLAndCreateTime(ctx context.Context, url string, startTime, endTime time.Time) ([]WebReader, error) {
	var res []WebReader

	filter := bson.M{"url": url, "is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByUrlAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByUrlAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *WebReaderDao) FindWebReaderByTitleAndCreateTime(ctx context.Context, title string, startTime, endTime time.Time) ([]WebReader, error) {
	var res []WebReader

	filter := bson.M{"title": bson.M{"$regex": title, "$options": "i"}, "is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByTitleAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByTitleAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}
