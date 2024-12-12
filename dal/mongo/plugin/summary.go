package plugin

import (
	"context"
	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"sync"
	"time"

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
	SafeCount          int                `bson:"safe_count" json:"safe_count"`
	OutlineType        int                `bson:"outline_type" json:"outline_type"`
	CopyFromSummaryID  string             `bson:"copy_from_summary_id" json:"copy_from_summary_id"`
	CopyFromResourceID string             `bson:"copy_from_resource_id" json:"copy_from_resource_id"`
	IsDelete           bool               `bson:"is_delete" json:"is_delete" default:"false"`
	IsStopped          bool               `bson:"is_stopped" json:"is_stopped"`
	CreateTime         time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime         time.Time          `bson:"update_time" json:"update_time"`
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

func (d *SummaryDao) CountByUserIDAndUrl(ctx context.Context, userID, url string, entryType int) (int64, error) {
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"user_id": userID},
		{"url": url},
		{"entry_type": entryType},
	}}
	count, err := pluginCollection.Collection(TableNameSummary).CountDocuments(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileById] mongo find error:%+v", err)
		return 0, err
	}

	return count, nil
}

func (d *SummaryDao) FindSummaryByTimeRangeForSave(ctx context.Context, entryType int, startTime, endTime time.Time) ([]*Summary, error) {
	var res []*Summary

	filter := bson.M{
		//"is_delete": false,
		"create_time":           bson.M{"$gte": startTime, "$lt": endTime},
		"copy_from_resource_id": "",
		"copy_from_summary_id":  "",
	}
	options := options.Find().SetProjection(bson.M{"_id": 1, "user_id": 1, "url": 1, "entry_type": 1, "channel_type": 1, "file_id": 1, "pair_id": 1, "outline_type": 1, "copy_from_summary_id": 1, "copy_from_resource_id": 1, "content_size": bson.M{"$strLenCP": "$parsing_result"}, "is_delete": 1, "create_time": 1, "update_time": 1}).
		SetSort(bson.D{{Key: "create_time", Value: -1}})
	cur, err := pluginCollection.Collection(TableNameSummary).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindSummaryByTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindSummaryByTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *Summary) TranslateEntryInfo() *bi.EntryInfo {
	return &bi.EntryInfo{
		ID:              primitive.NewObjectID(),
		EntryID:         d.ID.Hex(),
		EntryType:       int(d.EntryType),
		EntrySource:     consts.EntryInfoEntrySourceSummary,
		SourceTable:     TableNameFile,
		DataType:        consts.EntryInfoDataTypeSummary,
		UserID:          d.UserID,
		ChannelType:     d.ChannelType,
		Status:          0,
		Cost:            0, // TODO
		EntryCreateTime: d.CreateTime,
		EntryUpdateTime: d.UpdateTime,
		CreateTime:      time.Now(),
	}
}
