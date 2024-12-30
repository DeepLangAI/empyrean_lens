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

type WebReader struct {
	ID                  primitive.ObjectID `bson:"_id" json:"_id"`
	URL                 string             `bson:"url" json:"url" validate:"required"`
	UserID              string             `bson:"user_id" json:"user_id" validate:"required,min=5,max=64"`
	Title               string             `bson:"title" json:"title" default:""`
	Status              int                `bson:"status" json:"status"`
	ChannelType         int                `bson:"channel_type" json:"channel_type"`
	RealChannelType     int                `bson:"real_channel_type" json:"real_channel_type"`
	MultiId             string             `bson:"multi_id" json:"multi_id"`
	CopyFromUrlID       string             `bson:"copy_from_url_id" json:"copy_from_url_id"`
	CopyFromResourceID  string             `bson:"copy_from_resource_id" json:"copy_from_resource_id"`
	CopyParseResultFrom string             `bson:"copy_parse_result_from" json:"copy_parse_result_from"`
	ContentSize         int                `bson:"content_size" json:"content_size"`
	IsDeleted           bool               `bson:"is_deleted" json:"is_deleted" default:"false"`
	CreateTime          time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime          time.Time          `bson:"update_time" json:"update_time"`
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
	var res *WebReader
	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"$and": []bson.M{
		//{"is_deleted": false},
		{"_id": _id},
	}}
	options := options.FindOne().SetProjection(bson.M{"_id": 1, "url": 1, "user_id": 1, "title": 1, "status": 1, "channel_type": 1, "real_channel_type": 1, "multi_id": 1, "copy_from_url_id": 1, "copy_from_resource_id": 1, "copy_parse_result_from": 1, "content_size": bson.M{"$strLenCP": "$content"}, "is_deleted": 1, "create_time": 1, "update_time": 1})
	err := pluginCollection.Collection(TableNameWebReader).FindOne(ctx, filter, options).Decode(&res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderById] mongo find error:%+v", err)
		return nil, err
	}
	res.CreateTime = res.CreateTime.Local()
	res.UpdateTime = res.UpdateTime.Local()
	return res, nil
}

func (d *WebReaderDao) FindWebReaderByUserIDAndUrl(ctx context.Context, userID, url string) (*WebReader, error) {
	var res *WebReader
	filter := bson.M{"$and": []bson.M{
		//{"is_deleted": false},
		{"user_id": userID},
		{"url": url},
	}}
	options := options.FindOne().SetProjection(bson.M{"_id": 1, "url": 1, "user_id": 1, "title": 1, "status": 1, "channel_type": 1, "real_channel_type": 1, "multi_id": 1, "copy_from_url_id": 1, "copy_from_resource_id": 1, "copy_parse_result_from": 1, "content_size": bson.M{"$strLenCP": "$content"}, "is_deleted": 1, "create_time": 1, "update_time": 1})
	err := pluginCollection.Collection(TableNameWebReader).FindOne(ctx, filter, options).Decode(&res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByUserIDAndUrl] mongo find error:%+v", err)
		return nil, err
	}
	res.CreateTime = res.CreateTime.Local()
	res.UpdateTime = res.UpdateTime.Local()
	return res, nil
}

func (d *WebReaderDao) FindWebReaderByIds(ctx context.Context, ids []string) (map[string]*WebReader, error) {
	var res []*WebReader

	_ids := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		_id, _ := primitive.ObjectIDFromHex(id)
		_ids = append(_ids, _id)
	}
	filter := bson.M{"$and": []bson.M{
		//{"is_deleted": false},
		{"_id": bson.M{"$in": _ids}},
	}}
	options := options.Find().SetProjection(bson.M{"_id": 1, "url": 1, "user_id": 1, "title": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_url_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$content"}, "is_deleted": 1, "create_time": 1, "update_time": 1})
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter, options)
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
		return map[string]*WebReader{}, nil
	}
	webReaderMapping := make(map[string]*WebReader, len(res))
	for i := range res {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
		webReaderMapping[res[i].ID.Hex()] = res[i]
	}
	return webReaderMapping, nil
}

func (d *WebReaderDao) FindWebReaderByTimeRange(ctx context.Context, status []int32, startTime, endTime time.Time, skip, limit int64) ([]*WebReader, error) {
	var res []*WebReader

	filter := bson.M{
		//"is_deleted": false,
		"create_time":           bson.M{"$gte": startTime, "$lt": endTime},
		"copy_from_resource_id": "",
		"channel_type":          bson.M{"$nin": []int32{72, 82, 85}},
	}
	if len(status) > 0 {
		filter["status"] = bson.M{"$in": status}
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "url": 1, "user_id": 1, "title": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_url_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$content"}, "is_deleted": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
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

func (d *WebReaderDao) FindWebReaderByTimeRangeForSave(ctx context.Context, startTime, endTime time.Time) ([]*WebReader, error) {
	var res []*WebReader

	filter := bson.M{
		//"is_deleted": false,
		"create_time": bson.M{"$gte": startTime, "$lt": endTime},
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "url": 1, "user_id": 1, "title": 1, "status": 1, "channel_type": 1, "real_channel_type": 1, "multi_id": 1, "copy_from_url_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$content"}, "is_deleted": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}})
	cur, err := pluginCollection.Collection(TableNameWebReader).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByTimeRangeForSave] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindWebReaderByTimeRangeForSave] mongo all error:%+v", err)
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
	queryFilter := bson.M{"$or": []bson.M{{"title": bson.M{"$regex": query, "$options": "i"}}, {"user_id": query}, {"url": query}, {"_id": _id}}}
	filter := bson.M{
		"$and": []bson.M{
			queryFilter,
			//{"is_delete": false},
			{"create_time": bson.M{"$gte": startTime, "$lt": endTime}},
			{"copy_from_resource_id": ""},
		},
	}
	if len(status) > 0 {
		filter["status"] = bson.M{"$in": status}
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "url": 1, "user_id": 1, "title": 1, "status": 1, "channel_type": 1, "multi_id": 1, "copy_from_url_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$content"}, "is_deleted": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
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

func (d *WebReaderDao) FindByUserIDAndUrl(ctx context.Context, userID, url string) (*WebReader, error) {
	res := &WebReader{}
	filter := bson.M{
		"user_id": userID,
		"url":     url,
	}
	err := pluginCollection.Collection(TableNameWebReader).FindOne(ctx, filter).Decode(&res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByUserIDAndUrl] mongo find error:%+v", err)
		return nil, err
	}
	res.CreateTime = res.CreateTime.Local()
	res.UpdateTime = res.UpdateTime.Local()
	return res, nil
}

func (d *WebReader) TranslateEntryInfo() *bi.EntryInfo {
	parentEntryID := d.CopyFromUrlID
	parentEntryType := int(empyrean_lens.EntryTypeEnum_WEB)
	if d.CopyParseResultFrom != "" {
		parentEntryID = d.CopyParseResultFrom
		parentEntryType = int(empyrean_lens.EntryTypeEnum_WEB)
	}
	if d.CopyFromUrlID != "" {
		parentEntryID = d.CopyFromUrlID
		parentEntryType = int(empyrean_lens.EntryTypeEnum_WEB)
	}
	if d.CopyFromResourceID != "" {
		parentEntryID = d.CopyFromResourceID
		parentEntryType = int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB)
	}
	if d.RealChannelType >= int(empyrean_lens.ChannelType_IosUrl) {
		d.ChannelType = d.RealChannelType
	}
	return &bi.EntryInfo{
		ID:              primitive.NewObjectID(),
		EntryID:         d.ID.Hex(),
		EntryType:       int(empyrean_lens.EntryTypeEnum_WEB),
		EntrySource:     utils.GetEntrySource(d.UserID, d.CopyFromResourceID, d.CopyFromUrlID),
		SourceTable:     TableNameWebReader,
		DataType:        utils.GetDataType(d.MultiId, d.CopyFromResourceID, empyrean_lens.EntryTypeEnum_FILE),
		UserID:          d.UserID,
		Title:           d.Title,
		ChannelType:     d.ChannelType,
		MultiID:         d.MultiId,
		ParentEntryID:   parentEntryID,
		ParentEntryType: parentEntryType,
		EntryURL:        d.URL,
		Status:          int(utils.GetActionStatus(empyrean_lens.EntryTypeEnum_WEB, d.Status, 0, 0)),
		Cost:            0, // TODO
		ContentSize:     d.ContentSize,
		WebSite:         utils.ChannelIntToString(d.ChannelType),
		ActionName:      utils.GetActionName(int(empyrean_lens.EntryTypeEnum_WEB), d.MultiId, d.CopyFromResourceID, "", 0),
		EntryCreateTime: d.CreateTime,
		EntryUpdateTime: d.UpdateTime,
		CreateTime:      time.Now(),
	}
}
