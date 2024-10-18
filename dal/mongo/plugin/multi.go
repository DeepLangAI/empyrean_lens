package plugin

import (
	"context"
	"empyrean_lens/consts"
	"errors"
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
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	UserID         string             `json:"user_id" bson:"user_id" validate:"required"`
	ArticleList    []ArticleEntry     `json:"article_list" bson:"article_list"`
	Title          string             `json:"title" bson:"title"`
	AnalysisStatus int                `json:"analysis_status" bson:"analysis_status"`
	MergeStatus    int                `json:"merge_status" bson:"merge_status"`
	SummaryStatus  int                `json:"summary_status" bson:"summary_status"`
	ChannelType    int                `bson:"channel_type" json:"channel_type"`
	CreateTime     time.Time          `json:"create_time" bson:"create_time"`
	UpdateTime     time.Time          `json:"update_time" bson:"update_time"`
	IsDeleted      bool               `json:"is_deleted" bson:"is_deleted"`
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

func (d *MultiDao) FindMultiByCreateTime(ctx context.Context, startTime, endTime time.Time) ([]MultiModel, error) {
	var res []MultiModel

	filter := bson.M{"is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *MultiDao) FindMultiByUserIdAndCreateTime(ctx context.Context, userId string, startTime, endTime time.Time) ([]MultiModel, error) {
	var res []MultiModel

	filter := bson.M{"user_id": userId, "is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByUserIdAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByUserIdAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *MultiDao) FindMultiByIdAndCreateTime(ctx context.Context, id string, startTime, endTime time.Time) (MultiModel, error) {
	var res MultiModel

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": _id, "is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByIdAndCreateTime] mongo find error:%+v", err)
		return res, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		err = cur.Decode(&res)
		if err != nil {
			hlog.CtxErrorf(ctx, "db error, [FindMultiByIdAndCreateTime], err:%v", err)
			return res, err
		}
	}
	if res.ID.IsZero() {
		hlog.CtxInfof(ctx, "[FindMultiByIdAndCreateTime] mongo find nil: id=%s", id)
		return res, errors.New(consts.DB_NOT_FOUND_ERR)
	}
	return res, nil
}

func (d *MultiDao) FindMultiByTitleAndCreateTime(ctx context.Context, title string, startTime, endTime time.Time) ([]MultiModel, error) {
	var res []MultiModel

	filter := bson.M{"title": bson.M{"$regex": title, "$options": "i"}, "is_deleted": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByTitleAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByTitleAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}
