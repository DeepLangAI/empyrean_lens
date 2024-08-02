package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameSummary = "summary"

type Summary struct {
	EntryType         int       `bson:"entry_type" json:"entry_type"`
	CopyFromSummaryId string    `bson:"copy_from_summary_id" json:"copy_from_summary_id"`
	CreateTime        time.Time `bson:"create_time" json:"create_time"`
	UpdateTime        time.Time `bson:"update_time" json:"update_time"`
}

type SummaryDao struct{}

var summaryDao *SummaryDao
var summaryDaoInitOnce sync.Once

func NewSummaryModelDao() *SummaryDao {
	summaryDaoInitOnce.Do(func() {
		summaryDao = &SummaryDao{}
	})
	return summaryDao
}

func (self *SummaryDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]Summary, error) {
	var result []Summary
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	cur, err := lingoDatabase.
		Collection(TableNameSummary).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "Summary FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "Summary FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
