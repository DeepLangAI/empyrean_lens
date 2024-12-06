package plugin

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Summary struct {
	ID                 primitive.ObjectID `bson:"_id" json:"_id"`
	UserID             string             `bson:"user_id" json:"user_id"`
	Url                string             `bson:"url" json:"url"`
	EntryType          int                `bson:"entry_type" json:"entry_type"`
	ChannelType        int                `bson:"channel_type" json:"channel_type"`
	FileID             string             `bson:"file_id" json:"file_id"`
	PairID             string             `bson:"pair_id" json:"pair_id"`
	OutlineType        int                `bson:"outline_type" json:"outline_type"`
	CopyFromFildID     string             `bson:"copy_from_file_id" json:"copy_from_file_id"`
	CopyFromResourceID string             `bson:"copy_from_resource_id" json:"copy_from_resource_id"`
	ContentSize        int                `bson:"content_size" json:"content_size"`
	IsDelete           bool               `bson:"is_delete" json:"is_delete" default:"false"`
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
