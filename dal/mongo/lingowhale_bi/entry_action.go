package bi

import (
	"context"
	"errors"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/utils"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/utillib"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ActionIO struct {
	TraceID      string `json:"trace_id" bson:"trace_id"`
	ActionInput  any    `json:"action_input" bson:"action_input"`
	ActionOutput any    `json:"action_output" bson:"action_output"`
	ActionError  []any  `json:"action_error" bson:"action_error"`
}

type EntryAction struct {
	ID              primitive.ObjectID `json:"_id,omitempty" bson:"_id"`
	EntryID         string             `json:"entry_id" bson:"entry_id"`
	ActionChannel   int                `json:"action_channel" bson:"action_channel"`
	ActionType      int                `json:"action_type" bson:"action_type"`
	ActionIOs       []*ActionIO        `json:"action_ios" bson:"action_ios"`
	ActionStatus    int                `json:"action_status" bson:"action_status"`
	DataVersion     string             `json:"data_version" bson:"data_version"`
	Cost            int                `json:"cost" bson:"cost"`
	ActionStartTime time.Time          `json:"action_start_time" bson:"action_start_time"`
	ActionEndTime   time.Time          `json:"action_end_time" bson:"action_end_time"`
	CreateTime      time.Time          `json:"create_time" bson:"create_time"`
}

const TableNameEntryAction = "entry_action"

type EntryActionDao struct{}

var (
	entryActionDao     *EntryActionDao
	entryActionDaoOnce sync.Once
)

func NewEntryActionDao() *EntryActionDao {
	entryActionDaoOnce.Do(func() {
		entryActionDao = &EntryActionDao{}
	})
	return entryActionDao
}

func (d *EntryActionDao) SaveEntryAction(ctx context.Context, entryAction *EntryAction) error {
	// 是否存在
	info, err := d.FindByEntryTypeEntryIDAndActionType(ctx, entryAction.ActionChannel, entryAction.EntryID, entryAction.ActionType)
	if err != nil {
		hlog.CtxErrorf(ctx, "db error, method:Save EntryAction, err:%+v", err)
		return err
	}
	// 存在，upload
	if info != nil {
		filter := bson.M{"entry_id": entryAction.EntryID, "action_channel": entryAction.ActionChannel, "action_type": entryAction.ActionType}
		update := bson.M{"action_ios": entryAction.ActionIOs, "action_status": entryAction.ActionStatus, "cost": entryAction.Cost}
		res, err := biCollection.Collection(TableNameEntryAction).UpdateOne(ctx, filter, bson.M{"$set": update})
		if err != nil {
			hlog.CtxErrorf(ctx, "db error, method:Save EntryAction, err:%+v", err)
			return err
		}
		hlog.CtxInfof(ctx, "db info, method:Save EntryAction, res:%+v", res)
		return nil
	}
	// 不存在，插入
	_, err = biCollection.Collection(TableNameEntryAction).InsertOne(ctx, entryAction)
	if err != nil {
		hlog.CtxErrorf(ctx, "db error, method:Save EntryAction, err:%+v", err)
		return err
	}
	return nil
}

func (d *EntryActionDao) SaveBatchEntryAction(ctx context.Context, entryActions []*EntryAction) error {
	for _, entryAction := range entryActions {
		err := d.SaveEntryAction(ctx, entryAction)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *EntryActionDao) FindByEntryTypeEntryID(ctx context.Context, entryType int, entryID string) ([]*EntryAction, error) {
	var entryActions []*EntryAction
	cur, err := biCollection.Collection(TableNameEntryAction).Find(ctx, bson.M{"entry_id": entryID, "action_channel": entryType})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByEntryTypeEntryID, err:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &entryActions); err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryTypeEntryID] mongo all error:%+v", err)
		return nil, err
	}
	return entryActions, nil
}

func (d *EntryActionDao) FindByTraceID(ctx context.Context, traceID string) ([]*EntryAction, error) {
	var entryActions []*EntryAction
	cur, err := biCollection.Collection(TableNameEntryAction).Find(ctx, bson.M{"action_ios.trace_id": traceID})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByTraceID, err:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &entryActions); err != nil {
		hlog.CtxErrorf(ctx, "[FindByTraceID] mongo all error:%+v", err)
		return nil, err
	}
	return entryActions, nil
}

func (d *EntryActionDao) FindByEntryTypeEntryIDAndActionType(ctx context.Context, entryType int, entryID string, actionType int) (*EntryAction, error) {
	entryAction := &EntryAction{}
	err := biCollection.Collection(TableNameEntryAction).FindOne(ctx, bson.M{"entry_id": entryID, "action_channel": entryType, "action_type": actionType}).Decode(entryAction)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByEntryTypeEntryIDAndActionType, err:%+v", err)
		return nil, err
	}
	return entryAction, nil
}

func (d *EntryActionDao) FindByEntryIDAndActionTypeAndTraceId(ctx context.Context, entryID string, actionType int, traceId string) (*EntryAction, error) {
	entryAction := &EntryAction{}
	err := biCollection.Collection(TableNameEntryAction).FindOne(ctx, bson.M{"entry_id": entryID, "action_type": actionType, "trace_id": traceId}).Decode(entryAction)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByEntryIDAndActionTypeAndTraceId, err:%+v", err)
		return nil, err
	}
	return entryAction, nil
}

func (d *EntryActionDao) EntryActionExist(ctx context.Context, entryType int, entryID string, actionType int) bool {
	info, err := d.FindByEntryTypeEntryIDAndActionType(ctx, entryType, entryID, actionType)
	if err != nil {
		return true
	}
	// 查询不到返回true
	return info != nil
}

func (d *EntryAction) TranslateGraphNode() *empyrean_lens.GraphNode {
	node := &empyrean_lens.GraphNode{
		ID:         d.ID.Hex(),
		Name:       consts.LinkNodeTypeName[empyrean_lens.LinkNodeTypeEnum(d.ActionType)],
		Type:       empyrean_lens.LinkNodeTypeEnum(d.ActionType),
		EnterTime:  d.ActionStartTime.Format(consts.DateTimeTemplate),
		FinishTime: d.ActionEndTime.Format(consts.DateTimeTemplate),
		Status:     empyrean_lens.ActionStatusEnum(d.ActionStatus),
	}
	if node.EnterTime == "0001-01-01 00:00:00" {
		node.EnterTime = ""
	}
	if node.Status == empyrean_lens.ActionStatusEnum_FAIL || node.FinishTime == "0001-01-01 00:00:00" {
		node.FinishTime = ""
	}
	return node
}

func (d *ActionIO) TranslateApiLogs(actionType int) []*empyrean_lens.ApiLog {
	logs := []*empyrean_lens.ApiLog{}
	for _, log := range d.ActionError {
		logs = append(logs, &empyrean_lens.ApiLog{
			TraceID:  d.TraceID,
			ErrorMsg: log.(string),
			HTTPCode: 500,
		})
	}
	input := d.ActionInput.(string)
	if newInput, err := utillib.DeStrGzip(input); err == nil {
		input = newInput
	}
	input = utils.TranslateJsonIO(input)

	output := d.ActionOutput.(string)
	if newOutput, err := utillib.DeStrGzip(output); err == nil {
		output = newOutput
	}
	output = utils.TranslateJsonIO(output)

	if d.ActionOutput != nil && d.ActionInput != nil {
		logs = append(logs, &empyrean_lens.ApiLog{
			TraceID:  d.TraceID,
			Input:    input,
			Output:   output,
			HTTPCode: 200,
		})
	}
	return logs
}
