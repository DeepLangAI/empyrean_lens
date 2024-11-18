package plugin

import (
	"context"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type File struct {
	ID                 primitive.ObjectID `bson:"_id" json:"_id"`
	UserID             string             `bson:"user_id" json:"user_id"`
	Name               string             `bson:"name" json:"name"`
	FileURL            string             `bson:"file_url" json:"file_url"`
	Status             int                `bson:"status" json:"status"`
	ChannelType        int                `bson:"channel_type" json:"channel_type"`
	MultiId            string             `bson:"multi_id" json:"multi_id"`
	CopyFromFildID     string             `bson:"copy_from_file_id" json:"copy_from_file_id"`
	CopyFromResourceID string             `bson:"copy_from_resource_id" json:"copy_from_resource_id"`
	ContentSize        int                `bson:"content_size" json:"content_size"`
	IsDelete           bool               `bson:"is_delete" json:"is_delete" default:"false"`
	CreateTime         time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime         time.Time          `bson:"update_time" json:"update_time"`
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

func (d *FileDao) FindFileById(ctx context.Context, id string) (*File, error) {
	var res []*File

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"_id": _id},
	}}
	options := options.Find().SetProjection(bson.M{"_id": 1, "user_id": 1, "name": 1, "file_url": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_file_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$parsing_result"}, "is_delete": 1, "create_time": 1, "update_time": 1})
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileById] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileById] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	res[0].CreateTime = res[0].CreateTime.Local()
	res[0].UpdateTime = res[0].UpdateTime.Local()
	return res[0], nil
}

func (d *FileDao) FindFileByIds(ctx context.Context, ids []string) (map[string]*File, error) {
	var res []*File

	_ids := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		_id, _ := primitive.ObjectIDFromHex(id)
		_ids = append(_ids, _id)
	}
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"_id": bson.M{"$in": _ids}},
	}}
	options := options.Find().SetProjection(bson.M{"_id": 1, "user_id": 1, "name": 1, "file_url": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_file_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$parsing_result"}, "is_delete": 1, "create_time": 1, "update_time": 1})
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileById] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileById] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return map[string]*File{}, nil
	}
	fileMapping := make(map[string]*File)
	for _, v := range res {
		v.CreateTime = v.CreateTime.Local()
		v.UpdateTime = v.UpdateTime.Local()
		fileMapping[v.ID.Hex()] = v
	}
	return fileMapping, nil
}

func (d *FileDao) FindFileByTimeRange(ctx context.Context, status []int32, startTime, endTime time.Time, skip, limit int64) ([]*File, error) {
	var res []*File

	filter := bson.M{
		//"is_delete": false,
		"create_time":           bson.M{"$gte": startTime, "$lt": endTime},
		"copy_from_resource_id": "",
		"channel_type":          bson.M{"$nin": []int32{72, 82, 85}},
	}
	if len(status) > 0 {
		filter["status"] = bson.M{"$in": status}
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "user_id": 1, "name": 1, "file_url": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_file_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$parsing_result"}, "is_delete": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *FileDao) FindFileByTimeRangeForSave(ctx context.Context, startTime, endTime time.Time) ([]*File, error) {
	var res []*File

	filter := bson.M{
		//"is_delete": false,
		"create_time":           bson.M{"$gte": startTime, "$lt": endTime},
		"copy_from_resource_id": "",
		"channel_type":          bson.M{"$nin": []int32{72, 82, 85}},
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "user_id": 1, "name": 1, "file_url": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_file_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$parsing_result"}, "is_delete": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}})
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *FileDao) FindFileByQueryAndTimeRange(ctx context.Context, query string, status []int32, startTime, endTime time.Time, skip, limit int64) ([]*File, error) {
	var res []*File

	_id, _ := primitive.ObjectIDFromHex(query)
	queryFilter := bson.M{"$or": []bson.M{{"name": bson.M{"$regex": query, "$options": "i"}}, {"user_id": query}, {"_id": _id}}}
	filter := bson.M{"$and": []bson.M{
		queryFilter,
		//{"is_delete": false},
		{"create_time": bson.M{"$gte": startTime, "$lt": endTime}},
		{"copy_from_resource_id": ""},
	}}
	if len(status) > 0 {
		filter["status"] = bson.M{"$in": status}
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "user_id": 1, "name": 1, "file_url": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_file_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$parsing_result"}, "is_delete": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByQueryAndTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByQueryAndTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *File) TranslateEntryInfo() *bi.EntryInfo {
	parentEntryID := d.CopyFromFildID
	if d.CopyFromResourceID != "" {
		parentEntryID = d.CopyFromResourceID
	}
	return &bi.EntryInfo{
		ID:              primitive.NewObjectID(),
		EntryID:         d.ID.Hex(),
		EntryType:       int(empyrean_lens.EntryTypeEnum_FILE),
		EntrySource:     utils.GetEntrySource(d.UserID, d.CopyFromResourceID, d.CopyFromFildID),
		SourceTable:     TableNameFile,
		DataType:        utils.GetDataType(d.MultiId, d.CopyFromResourceID, empyrean_lens.EntryTypeEnum_FILE),
		UserID:          d.UserID,
		Title:           d.Name,
		ChannelType:     d.ChannelType,
		MultiID:         d.MultiId,
		ParentEntryID:   parentEntryID,
		EntryURL:        d.FileURL,
		Status:          int(utils.GetActionStatus(empyrean_lens.EntryTypeEnum_FILE, d.Status, 0, 0)),
		Cost:            0, //TODO
		ContentSize:     d.ContentSize,
		EntryCreateTime: d.CreateTime,
		EntryUpdateTime: d.UpdateTime,
		CreateTime:      time.Now(),
	}
}
