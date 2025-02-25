package empyrean_lens

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sync"
	"time"
)

var TableNameModelAutomation = "model_automation"

type ModelAutomationModel struct {
	Id               primitive.ObjectID     `bson:"_id"`
	TestCaseVersion  string                 `bson:"test_case_version"`
	CaseType         string                 `bson:"case_type"`
	ModelVersion     string                 `bson:"model_version"`
	FileEntryIds     []string               `bson:"file_entry_ids"`
	ResultCollection map[string]interface{} `bson:"result_collection"`
	IsUsing          bool                   `bson:"is_using"`
	Extra            map[string]interface{} `bson:"extra"`
	CreateTime       time.Time              `bson:"create_time"`
	UpdateTime       time.Time              `bson:"update_time"`
}

type ModelAutomationDao struct{}

var modelAutomationDao *ModelAutomationDao
var modelAutomationInitOnce sync.Once

func NewModelAutomationDao() *ModelAutomationDao {
	modelAutomationInitOnce.Do(func() {
		modelAutomationDao = &ModelAutomationDao{}
	})
	return modelAutomationDao
}

func (self *ModelAutomationDao) GetModelAutomationInfoByTime(ctx context.Context, beginTime, endTime time.Time) ([]ModelAutomationModel, error) {
	var result []ModelAutomationModel
	cur, err := probeDatabase.
		Collection(TableNameModelAutomation).
		Find(ctx, bson.M{
			"update_time": bson.M{
				"$gte": beginTime,
				"$lt":  endTime,
			},
			"is_using": true,
		})
	if err != nil {
		hlog.CtxErrorf(ctx, "find model_automation model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find model_automation model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}
