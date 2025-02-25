package empyrean_lens

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

var TableNameModelCaseResult = "model_case_result"

type ModelCaseResultModel struct {
	CaseId         string                 `bson:"case_id"`
	FileEntryId    string                 `bson:"file_entry_id"`
	FileTypeDetail map[string]interface{} `bson:"file_type_detail"`
	CaseResult     bool                   `bson:"case_result"`
	CaseType       string                 `bson:"case_type"`
	FailureDetails map[string]interface{} `bson:"failure_details"`
	TestDuration   float64                `bson:"test_duration"`
	ErrorLog       string                 `bson:"error_log"`
	IsUsing        bool                   `bson:"is_using"`
	Extra          map[string]interface{} `bson:"extra"`
	CreateTime     time.Time              `bson:"create_time"`
	UpdateTime     time.Time              `bson:"update_time"`
}

type ModelCaseResultDao struct{}

var modelCaseResultDao *ModelCaseResultDao
var modelCaseResultInitOnce sync.Once

func NewModelCaseResultDao() *ModelCaseResultDao {
	modelCaseResultInitOnce.Do(func() {
		modelCaseResultDao = &ModelCaseResultDao{}
	})
	return modelCaseResultDao
}

func (self *ModelCaseResultDao) GetModelCaseResultsByTime(ctx context.Context, beginTime, endTime time.Time) ([]ModelCaseResultModel, error) {
	var result []ModelCaseResultModel
	cur, err := probeDatabase.
		Collection(TableNameModelCaseResult).
		Find(ctx, bson.M{
			"update_time": bson.M{
				"$gte": beginTime,
				"$lt":  endTime,
			},
			"is_using": true,
		})
	if err != nil {
		hlog.CtxErrorf(ctx, "find model_case_result model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find model_case_result model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}

func (self *ModelCaseResultDao) GetModelCaseResultsByTimeAndEntryType(ctx context.Context, beginTime, endTime time.Time, entryType int64, failOrTotal bool) ([]ModelCaseResultModel, error) {
	var result []ModelCaseResultModel
	q := bson.M{
		"update_time": bson.M{
			"$gte": beginTime,
			"$lt":  endTime,
		},
		"is_using":                    true,
		"file_type_detail.entry_type": entryType,
	}
	//if failOrTotal {
	//	q["case_result"] = false
	//}
	cur, err := probeDatabase.
		Collection(TableNameModelCaseResult).
		Find(ctx, q)
	if err != nil {
		hlog.CtxErrorf(ctx, "find model_case_result model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find model_case_result model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}

func (self *ModelCaseResultDao) GetModelCaseResultsByTimeAndEntryTypeAndCaseType(ctx context.Context, beginTime, endTime time.Time, entryType int64, caseType string, failOrTotal bool) ([]ModelCaseResultModel, error) {
	var result []ModelCaseResultModel
	q := bson.M{
		"update_time": bson.M{
			"$gte": beginTime,
			"$lt":  endTime,
		},
		"is_using":                    true,
		"file_type_detail.entry_type": entryType,
		"case_type":                   caseType,
	}
	if failOrTotal {
		q["case_result"] = false
	}
	cur, err := probeDatabase.
		Collection(TableNameModelCaseResult).
		Find(ctx, q)
	if err != nil {
		hlog.CtxErrorf(ctx, "find model_case_result model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find model_case_result model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}
