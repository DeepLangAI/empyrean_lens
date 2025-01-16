package plugin

import (
	"context"
	"empyrean_lens/utils"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SummaryType int

var (
	SummaryTypeSummary       SummaryType = 1
	SummaryTypeViewPoint     SummaryType = 2
	SummaryTypeSimpleOutline SummaryType = 3
	SummaryTypeDetailOutline SummaryType = 4
)

type ResourceSummary struct {
	ID             primitive.ObjectID `bson:"_id" json:"_id"`
	EntryType      int                `bson:"entry_type" json:"entry_type"`
	EntryID        string             `bson:"entry_id" json:"entry_id"`
	SummaryType    SummaryType        `bson:"summary_type" json:"summary_type"`
	SummaryContent []any              `bson:"summary_content" json:"summary_content"`
	CreateTime     time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime     time.Time          `bson:"update_time" json:"update_time"`
}

const TableNameResourceSummary = "resource_summary"

var resourceSummaryDao *ResourceSummaryDao

type ResourceSummaryDao struct {
}

var ResourceSummaryDaoOnce sync.Once

func NewResourceSummaryDao() *ResourceSummaryDao {
	ResourceDaoOnce.Do(func() {
		resourceSummaryDao = &ResourceSummaryDao{}
	})
	return resourceSummaryDao
}

func (d *ResourceSummaryDao) FindByEntryTypeAndEntryID(ctx context.Context, entryType int, entryID string) ([]*ResourceSummary, error) {
	var res []*ResourceSummary
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"entry_type": entryType},
		{"entry_id": entryID},
	}}
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: 1}})
	cur, err := pluginCollection.Collection(TableNameResourceSummary).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryTypeAndEntryID] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryTypeAndEntryID] mongo all error:%+v", err)
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

func (d *ResourceSummaryDao) FindByEntryTypeAndEntryIDAndSummaryType(ctx context.Context, entryType int, entryID string, summaryType int) (*ResourceSummary, error) {
	var res *ResourceSummary
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"entry_type": utils.TranslateSubscribeEntryType(entryType)},
		{"entry_id": entryID},
		{"summary_type": summaryType},
	}}
	err := pluginCollection.Collection(TableNameResourceSummary).FindOne(ctx, filter).Decode(&res)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryTypeAndEntryIDAndSummaryType] mongo find error:%+v", err)
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	res.CreateTime = res.CreateTime.Local()
	res.UpdateTime = res.UpdateTime.Local()
	res.EntryType = utils.TranslateEntryType(res.EntryType)
	return res, nil
}
