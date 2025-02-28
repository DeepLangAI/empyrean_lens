package empyrean_lens

import (
	"context"
	"encoding/json"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"sync"
	"time"
)

var TableNameModelAutomation = "model_automation"

type ModelAutomationModel struct {
	Id                primitive.ObjectID     `bson:"_id"`
	TestCaseVersion   string                 `bson:"test_case_version"`
	ModelVersion      string                 `bson:"model_version"`
	EnvironmentConfig map[string]interface{} `bson:"environment_config"`
	FileEntryIds      []string               `bson:"file_entry_ids"`
	ResultCollection  map[string]interface{} `bson:"result_collection"`
	IsUsing           bool                   `bson:"is_using"`
	Extra             map[string]interface{} `bson:"extra"`
	CreateTime        time.Time              `bson:"create_time"`
	UpdateTime        time.Time              `bson:"update_time"`
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

func (self *ModelAutomationDao) SaveModelAutomationInfo(ctx context.Context, req map[string]string) (primitive.ObjectID, error) {
	// 解析req, 构建Model
	testCaseVersion := req["test_case_version"]
	environmentConfig := make(map[string]interface{})
	if ec, ok := req["environment_config"]; ok {
		if err := json.Unmarshal([]byte(ec), &environmentConfig); err != nil {
			return primitive.ObjectID{}, err
		}
	}
	modelVersion := req["model_version"]
	fileEntryIds := make([]string, 0)
	if fe, ok := req["file_entry_ids"]; ok {
		if err := json.Unmarshal([]byte(fe), &fileEntryIds); err != nil {
			return primitive.ObjectID{}, err
		}
	}
	resultCollection := make(map[string]interface{})
	if rc, ok := req["result_collection"]; ok {
		if err := json.Unmarshal([]byte(rc), &resultCollection); err != nil {
			return primitive.ObjectID{}, err
		}
	}
	extra := make(map[string]interface{})
	if ex, ok := req["extra"]; ok {
		if err := json.Unmarshal([]byte(ex), &extra); err != nil {
			return primitive.ObjectID{}, err
		}
	}
	id := primitive.NewObjectID()
	model := ModelAutomationModel{
		Id:                id,
		TestCaseVersion:   testCaseVersion,
		ModelVersion:      modelVersion,
		EnvironmentConfig: environmentConfig,
		FileEntryIds:      fileEntryIds,
		ResultCollection:  resultCollection,
		IsUsing:           true,
		Extra:             extra,
		CreateTime:        time.Now().AddDate(0, 0, -1),
		UpdateTime:        time.Now().AddDate(0, 0, -1),
	}
	_, err := probeDatabase.Collection(TableNameModelAutomation).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "insert model_automation model failed, err: %v", err)
		return primitive.ObjectID{}, err
	}
	return id, nil
}
