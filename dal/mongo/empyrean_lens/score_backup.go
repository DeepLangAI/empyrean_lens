package empyrean_lens

import (
	"context"
	"empyrean_lens/consts"
	"errors"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"sync"
	"time"
)

var TableNameScoreBackup = "score_backup"

type ScoreBackupModel struct {
	Time              time.Time `bson:"time"`
	Score             float64   `bson:"score"`
	ScoreDayOverDay   float64   `bson:"score_day_over_day"`
	ScoreWeekOverWeek float64   `bson:"score_week_over_week"`
	TotalReq          int32     `bson:"total_req"`
	FailReq           int32     `bson:"fail_req"`
	SlowReq           int32     `bson:"slow_req"`
	FailRate          float64   `bson:"fail_rate"`
	SlowRate          float64   `bson:"slow_rate"`
	ProbeFailReq      int32     `bson:"probe_fail_req"`
	ProbeTotalReq     int32     `bson:"probe_total_req"`
	ProbeFailRate     float64   `bson:"probe_fail_rate"`
	AvgRespCost       float64   `bson:"avg_resp_cost"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type ScoreBackupDao struct{}

var scoreBackupDao *ScoreBackupDao

var scoreBackupDaoInitOnce sync.Once

func NewScoreBackupDao() *ScoreBackupDao {
	scoreBackupDaoInitOnce.Do(func() {
		scoreBackupDao = &ScoreBackupDao{}
	})
	return scoreBackupDao
}

func (self *ScoreBackupDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameScoreBackup).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "drop system score table failed, err: %v", err)
		return err
	}
	return nil
}

func (self *ScoreBackupDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameScoreBackup).
		DeleteMany(ctx, bson.M{"time": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete system score model failed, err: %v", err)
		return err
	}
	return nil

}

func (self *ScoreBackupDao) Save(ctx context.Context, model ScoreBackupModel) error {
	_, err := probeDatabase.Collection(TableNameScoreBackup).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "save system score model failed, err: %v", err)
		return err
	}
	return nil
}

func (self *ScoreBackupDao) FindTimespanScore(ctx context.Context, timeBegin, timeEnd time.Time) ([]ScoreBackupModel, error) {
	var result []ScoreBackupModel
	cur, err := probeDatabase.
		Collection(TableNameScoreBackup).
		Find(ctx, bson.M{"time": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts.StatusValid})
	if err != nil {
		hlog.CtxErrorf(ctx, "find system score model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find system score model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}
func (self *ScoreBackupDao) FindScoreByTime(ctx context.Context, _time time.Time) (*ScoreBackupModel, error) {
	result, err := self.FindTimespanScore(ctx, _time, _time.Add(24*time.Hour))
	if err != nil {
		return nil, err
	}
	if len(result) != 1 {
		return nil, errors.New("system score backup model not found")
	}
	return &result[0], nil
}

func (self *ScoreBackupDao) convertToBsonM(model ScoreBackupModel) (bson.M, error) {
	result := bson.M{}
	v := reflect.ValueOf(model)

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if v.Field(i).IsZero() {
			continue
		}
		tag := ""
		// 取注解的bson值
		if bsonTag := field.Tag.Get("bson"); bsonTag != "" {
			tag = bsonTag
		} else {
			tag = field.Name
		}
		if tag == "" || tag == "create_time" || tag == "update_time" {
			continue
		}
		result[tag] = v.Field(i).Interface()
	}

	return result, nil
}

func (self *ScoreBackupDao) CreateOrUpdate(ctx context.Context, update ScoreBackupModel) error {
	updateModel, err := self.convertToBsonM(update)
	updateModel["update_time"] = time.Now() //始终更新update_time
	if err != nil {
		hlog.CtxErrorf(ctx, "convert to bsonM failed, err: %v", err)
		return err
	}
	_, err = probeDatabase.Collection(TableNameScoreBackup).
		UpdateOne(
			ctx,
			bson.M{"time": update.Time, "status": consts.StatusValid},
			bson.M{
				"$set": updateModel,
				"$setOnInsert": bson.M{ // 仅当第一次插入时，才设置这些字段
					"create_time": time.Now(),
				},
			},
			options.Update().SetUpsert(true),
		)
	if err != nil {
		hlog.CtxErrorf(ctx, "update system score model failed, err: %v", err)
		return err
	}
	return nil
}
