package plugin

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WebReader struct {
	ID          primitive.ObjectID `bson:"_id" json:"_id"`
	URL         string             `bson:"url" json:"url" validate:"required"`
	UserID      string             `bson:"user_id" json:"user_id" validate:"required,min=5,max=64"`
	Title       string             `bson:"title" json:"title" default:""`
	Status      int                `bson:"status" json:"status"`
	ChannelType int                `bson:"channel_type" json:"channel_type"`
	MultiId     string             `bson:"multi_id" json:"multi_id"`
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

func (d *WebReaderDao) FindWebReaderById(ctx context.Context, id string) (*WebReader, error) {
	var res []*WebReader

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"$and": []bson.M{
		//{"is_deleted": false},
		{"_id": _id},
	}}
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderById] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderById] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	res[0].CreateTime = res[0].CreateTime.Local()
	res[0].UpdateTime = res[0].UpdateTime.Local()
	return res[0], nil
}

func (d *WebReaderDao) FindWebReaderByTimeRange(ctx context.Context, status []int32, startTime, endTime time.Time, skip, limit int64) ([]*WebReader, error) {
	var res []*WebReader

	filter := bson.M{
		//"is_deleted": false,
		"create_time": bson.M{"$gte": startTime, "$lt": endTime},
	}
	if len(status) > 0 {
		filter["status"] = bson.M{"$in": status}
	}
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *WebReaderDao) FindWebReaderByQueryAndTimeRange(ctx context.Context, query string, status []int32, startTime, endTime time.Time, skip, limit int64) ([]*WebReader, error) {
	var res []*WebReader

	_id, _ := primitive.ObjectIDFromHex(query)
	queryFilter := bson.M{"$or": []bson.M{{"title": bson.M{"$regex": query, "$options": "i"}}, {"user_id": query}, {"_id": _id}}}
	filter := bson.M{
		"$and": []bson.M{
			queryFilter,
			//{"is_delete": false},
			{"create_time": bson.M{"$gte": startTime, "$lt": endTime}},
		},
	}
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByQueryAndTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByQueryAndTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}
