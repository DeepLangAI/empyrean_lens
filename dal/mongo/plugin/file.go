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

type File struct {
	ID          primitive.ObjectID `bson:"_id" json:"_id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	Name        string             `bson:"name" json:"name"`
	FileURL     string             `bson:"file_url" json:"file_url"`
	Status      int                `bson:"status" json:"status"`
	ChannelType int                `bson:"channel_type" json:"channel_type"`
	IsDelete    bool               `bson:"is_delete" json:"is_delete" default:"false"`
	CreateTime  time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime  time.Time          `bson:"update_time" json:"update_time"`
}

const TableNameFile = "file"

var fileDao *FileDao

type FileDao struct {
}

var FileDaoOnce sync.Once

func NewFileDao() *FileDao {
	FileDaoOnce.Do(func() {
		fileDao = &FileDao{}
	})
	return fileDao
}

func (d *FileDao) FindFileByCreateTime(ctx context.Context, startTime, endTime time.Time) ([]File, error) {
	var res []File

	filter := bson.M{"is_delete": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *FileDao) FindFileByUserIdAndCreateTime(ctx context.Context, userId string, startTime, endTime time.Time) ([]File, error) {
	var res []File

	filter := bson.M{"user_id": userId, "is_delete": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByUserIdAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByUserIdAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *FileDao) FindFileByIdAndCreateTime(ctx context.Context, id string, startTime, endTime time.Time) (File, error) {
	var res File

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": _id, "is_delete": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByIdAndCreateTime] mongo find error:%+v", err)
		return res, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		err = cur.Decode(&res)
		hlog.CtxInfof(ctx, "res=%+v", res)
		if err != nil {
			hlog.CtxErrorf(ctx, "db error, [FindFileByIdAndCreateTime], err:%v", err)
			return res, err
		}
	}
	if res.ID.IsZero() {
		hlog.CtxInfof(ctx, "[FindFileByIdAndCreateTime] mongo find nil: id=%s", id)
		return res, errors.New(consts.DB_NOT_FOUND_ERR)
	}
	return res, nil
}

func (d *FileDao) FindFileByFileURLAndCreateTime(ctx context.Context, url string, startTime, endTime time.Time) ([]File, error) {
	var res []File

	filter := bson.M{"file_url": url, "is_delete": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByFileUrlAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByFileUrlAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}

func (d *FileDao) FindFileByNameAndCreateTime(ctx context.Context, name string, startTime, endTime time.Time) ([]File, error) {
	var res []File

	filter := bson.M{"name": bson.M{"$regex": name, "$options": "i"}, "is_delete": false, "create_time": bson.M{"$gte": startTime, "$lt": endTime}}
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByNameAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByNameAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return res, nil
}
