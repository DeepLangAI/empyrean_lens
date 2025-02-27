package empyrean_lens

import (
	"context"
	"encoding/json"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
	"sync"
	"time"
)

var TableNameModelCaseResult = "model_case_result"

type ModelCaseResultModel struct {
	Id             primitive.ObjectID     `bson:"_id"`
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
			"create_time": bson.M{
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
		"create_time": bson.M{
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
		"create_time": bson.M{
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

func (self *ModelCaseResultDao) SaveModelCaseResultInfo(ctx context.Context, req map[string]string) (primitive.ObjectID, error) {
	caseId := req["case_id"]
	fileEntryId := req["file_entry_id"]
	fileTypeDetail := make(map[string]interface{})
	if ft, ok := req["file_type_detail"]; ok {
		if err := json.Unmarshal([]byte(ft), &fileTypeDetail); err != nil {
			hlog.CtxErrorf(ctx, "parse file_type_detail error in SaveModelCaseResultInfo :%v", err)
			return primitive.ObjectID{}, err
		}
	}
	caseResult := req["case_result"]
	hlog.CtxErrorf(ctx, "caseResult = %v", caseResult)
	caseType := req["case_type"]
	failureDetails := make(map[string]interface{})
	if fd, ok := req["failure_details"]; ok {
		if err := json.Unmarshal([]byte(fd), &failureDetails); err != nil {
			hlog.CtxErrorf(ctx, "parse failure_details error in SaveModelCaseResultInfo :%v", err)
			return primitive.ObjectID{}, err
		}
	}
	testDuration := req["test_duration"]
	testDurationFloat, err := strconv.ParseFloat(testDuration, 64)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse test_duration error in SaveModelCaseResultInfo :%v", err)
		return primitive.ObjectID{}, err
	}
	errorLog := req["error_log"]
	extra := make(map[string]interface{})
	if ex, ok := req["extra"]; ok {
		if err := json.Unmarshal([]byte(ex), &extra); err != nil {
			hlog.CtxErrorf(ctx, "parse extra error in SaveModelCaseResultInfo :%v", err)
			return primitive.ObjectID{}, err
		}
	}
	id := primitive.NewObjectID()
	model := ModelCaseResultModel{
		Id:             id,
		CaseId:         caseId,
		FileEntryId:    fileEntryId,
		FileTypeDetail: fileTypeDetail,
		CaseResult:     caseResult == "True",
		CaseType:       caseType,
		FailureDetails: failureDetails,
		TestDuration:   testDurationFloat,
		ErrorLog:       errorLog,
		IsUsing:        true,
		Extra:          extra,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}
	hlog.CtxErrorf(ctx, "model = %+v", model)
	_, err = probeDatabase.
		Collection(TableNameModelCaseResult).
		InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "insert model_case_result model failed, err: %v", err)
		return primitive.ObjectID{}, err
	}
	return id, nil
}
