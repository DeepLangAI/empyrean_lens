package plugin

import (
	"context"
	"sync"
	"time"

	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Summary struct {
	ID                 primitive.ObjectID `bson:"_id" json:"_id"`
	UserID             string             `bson:"user_id" json:"user_id"`
	Url                string             `bson:"url" json:"url"`
	EntryType          int                `bson:"entry_type" json:"entry_type"`
	ChannelType        int                `bson:"channel_type" json:"channel_type"`
	FileID             string             `bson:"file_id" json:"file_id"`
	PairID             string             `bson:"pair_id" json:"pair_id"`
	Content            string             `bson:"content" json:"content"`
	OutlineType        int                `bson:"outline_type" json:"outline_type"`
	SummaryLangType    string             `bson:"summary_lang_type" json:"summary_lang_type"`
	CopyFromSummaryID  string             `bson:"copy_from_summary_id" json:"copy_from_summary_id"`
	CopyFromResourceID string             `bson:"copy_from_resource_id" json:"copy_from_resource_id"`
	IsDelete           bool               `bson:"is_delete" json:"is_delete" default:"false"`
	IsStopped          bool               `bson:"is_stopped" json:"is_stopped"`
	CreateTime         time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime         time.Time          `bson:"update_time" json:"update_time"`

	// 不是从数据库获取的
	SourceEntryType int    `json:"source_entry_type" bson:"source_entry_type"`
	SourceEntryID   string `json:"source_entry_id" bson:"source_entry_id"`
	SourceTitle     string `json:"source_title" bson:"source_title"`
}

const TableNameSummary = "summary"

var summaryDao *SummaryDao

type SummaryDao struct {
}

var SummaryDaoOnce sync.Once

func NewSummaryDao() *SummaryDao {
	SummaryDaoOnce.Do(func() {
		summaryDao = &SummaryDao{}
	})
	return summaryDao
}

func (d *SummaryDao) FindBySummaryID(ctx context.Context, summaryID string) (*Summary, error) {
	var res *Summary
	_id, _ := primitive.ObjectIDFromHex(summaryID)
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"_id": _id},
	}}
	err := pluginCollection.Collection(TableNameSummary).FindOne(ctx, filter).Decode(&res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindBySummaryID] mongo find error:%+v", err)
		return nil, err
	}
	res.CreateTime = res.CreateTime.Local()
	res.UpdateTime = res.UpdateTime.Local()
	return res, nil
}

func (d *SummaryDao) FindByUserIDAndTypeAndPairID(ctx context.Context, userID string, entryType, outlineType int, pairID string) (*Summary, error) {
	var res *Summary
	filter := bson.M{}
	if outlineType != 0 {
		filter = bson.M{"$and": []bson.M{
			//{"is_delete": false},
			{"user_id": userID},
			{"entry_type": entryType},
			{"outline_type": outlineType},
			{"pair_id": pairID},
		}}
	} else {
		filter = bson.M{"$and": []bson.M{
			//{"is_delete": false},
			{"user_id": userID},
			{"entry_type": entryType},
			{"pair_id": pairID},
			{"$or": []bson.M{
				{"outline_type": bson.M{"$exists": false}},
				{"outline_type": 0},
			}},
		}}
	}
	err := pluginCollection.Collection(TableNameSummary).FindOne(ctx, filter).Decode(&res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByUserIDAndTypeAndPairID] mongo find error:%+v", err)
		return nil, err
	}
	res.CreateTime = res.CreateTime.Local()
	res.UpdateTime = res.UpdateTime.Local()
	return res, nil
}

func (d *SummaryDao) CountByUserIDAndUrl(ctx context.Context, userID, url string, entryType int) (int64, error) {
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"user_id": userID},
		{"url": url},
		{"entry_type": entryType},
	}}
	count, err := pluginCollection.Collection(TableNameSummary).CountDocuments(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[CountByUserIDAndUrl] mongo find error:%+v", err)
		return 0, err
	}
	return count, nil
}

func (d *SummaryDao) FindByUserIDAndUrlAndType(ctx context.Context, userID, url string, entryType int, outLineType int) (*Summary, error) {
	var res *Summary
	var filter bson.M
	if outLineType != 0 {
		filter = bson.M{"$and": []bson.M{
			//{"is_delete": false},
			{"user_id": userID},
			{"url": url},
			{"entry_type": entryType},
			{"outline_type": outLineType},
		}}
	} else {
		filter = bson.M{"$and": []bson.M{
			//{"is_delete": false},
			{"user_id": userID},
			{"url": url},
			{"entry_type": entryType},
			{"$or": []bson.M{
				{"outline_type": bson.M{"$exists": false}},
				{"outline_type": bson.M{"$in": []int{0, 1}}},
			}},
		}}
	}
	options := options.FindOne().SetSort(bson.D{{Key: "create_time", Value: 1}})
	err := pluginCollection.Collection(TableNameSummary).FindOne(ctx, filter, options).Decode(&res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByUserIDAndUrlAndType] mongo find error:%+v", err)
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	res.CreateTime = res.CreateTime.Local()
	res.UpdateTime = res.UpdateTime.Local()
	return res, nil
}

func (d *SummaryDao) FindByUserIDAndUrl(ctx context.Context, userID, url string) ([]*Summary, error) {
	var res []*Summary
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"user_id": userID},
		{"url": url},
	}}
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: 1}})
	cur, err := pluginCollection.Collection(TableNameSummary).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[QueryByTypeAndID] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[QueryByTypeAndID] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	if len(res) == 0 {
		return nil, nil
	}
	return res, nil
}

func (d *SummaryDao) QueryByTypeAndID(ctx context.Context, entryType int, entryID string) (*Summary, error) {
	var res []*Summary
	_id, _ := primitive.ObjectIDFromHex(entryID)
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"entry_type": entryType},
		{"_id": _id},
	}}
	cur, err := pluginCollection.Collection(TableNameSummary).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[QueryByTypeAndID] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[QueryByTypeAndID] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	if len(res) == 0 {
		return nil, nil
	}
	return res[0], nil
}

func (d *SummaryDao) FindSummaryByTimeRangeForSave(ctx context.Context, entryType int, startTime, endTime time.Time) ([]*Summary, error) {
	var res []*Summary

	filter := bson.M{
		//"is_delete": false,
		"create_time":           bson.M{"$gte": startTime, "$lt": endTime},
		"copy_from_resource_id": "",
		"copy_from_summary_id":  "",
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "user_id": 1, "url": 1, "entry_type": 1, "channel_type": 1, "file_id": 1, "pair_id": 1, "outline_type": 1, "copy_from_summary_id": 1, "copy_from_resource_id": 1, "is_delete": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}})
	cur, err := pluginCollection.Collection(TableNameSummary).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindSummaryByTimeRangeForSave] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindSummaryByTimeRangeForSave] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *SummaryDao) QueryFirstSummary(ctx context.Context, entryType int, userID, url string) (*Summary, error) {
	var res []*Summary

	filter := bson.M{
		//"is_delete": false,
		"entry_type": entryType,
		"user_id":    userID,
		"url":        url,
	}

	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: 1}}).SetLimit(1)
	cur, err := pluginCollection.Collection(TableNameSummary).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[QueryFirstSummary] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[QueryFirstSummary] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	if len(res) != 0 {
		return res[0], nil
	}
	return nil, nil
}

func (d *Summary) TranslateEntryInfo() *bi.EntryInfo {
	return &bi.EntryInfo{
		ID:          primitive.NewObjectID(),
		EntryID:     d.ID.Hex(),
		EntryType:   int(d.EntryType),
		EntrySource: consts.EntryInfoEntrySourceSummary,
		SourceTable: TableNameSummary,
		SourceEntryInfo: bi.SourceEntryInfo{
			EntryType: d.SourceEntryType,
			EntryID:   d.SourceEntryID,
		},
		DataType:        consts.EntryInfoDataTypeSummary,
		Language:        d.SummaryLangType,
		UserID:          d.UserID,
		Title:           d.SourceTitle,
		EntryURL:        d.Url,
		ChannelType:     d.ChannelType,
		OutlineType:     d.OutlineType,
		EntryCreateTime: d.CreateTime,
		EntryUpdateTime: d.UpdateTime,
		CreateTime:      time.Now(),
	}
}
