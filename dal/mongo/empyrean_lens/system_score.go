package empyrean_lens

import (
	"context"
	"empyrean_lens/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

const TableNameSystemScore = "system_score"

type SystemScoreModel struct {
	Date          time.Time `bson:"date"`
	Score         float64   `bson:"score"`
	TotalReq      int32     `bson:"total_req"`
	FailReq       int32     `bson:"fail_req"`
	SlowReq       int32     `bson:"slow_req"`
	FailRate      float64   `bson:"fail_rate"`
	SlowRate      float64   `bson:"slow_rate"`
	ProbeFailReq  int32     `bson:"probe_fail_req"`
	ProbeTotalReq int32     `bson:"probe_total_req"`
	ProbeFailRate float64   `bson:"probe_fail_rate"`
	AvgRespCost   float64   `bson:"avg_resp_cost"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type SystemScoreDao struct{}

var systemScoreDao *SystemScoreDao

var systemScoreDaoInitOnce sync.Once

func NewSystemScoreDao() *SystemScoreDao {
	systemScoreDaoInitOnce.Do(func() {
		systemScoreDao = &SystemScoreDao{}
	})
	return systemScoreDao
}

func (self *SystemScoreDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameSystemScore).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "drop system score table failed, err: %v", err)
		return err
	}
	return nil
}

func (self *SystemScoreDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameSystemScore).
		DeleteMany(ctx, bson.M{"date": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete system score model failed, err: %v", err)
		return err
	}
	return nil

}

func (self *SystemScoreDao) Save(ctx context.Context, model SystemScoreModel) error {
	_, err := probeDatabase.Collection(TableNameSystemScore).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "save system score model failed, err: %v", err)
		return err
	}
	return nil
}

func (self *SystemScoreDao) FindTimespanScore(ctx context.Context, timeBegin, timeEnd time.Time) ([]SystemScoreModel, error) {
	var result []SystemScoreModel
	cur, err := probeDatabase.
		Collection(TableNameSystemScore).
		Find(ctx, bson.M{"date": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts.StatusValid})
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
func (self *SystemScoreDao) FindScoreByTime(ctx context.Context, date time.Time) (*SystemScoreModel, error) {
	result, err := self.FindTimespanScore(ctx, date, date.Add(24*time.Hour))
	if err != nil {
		return nil, err
	}
	if len(result) != 1 {

		return nil, nil
	}
	return &result[0], nil
}

func (self *SystemScoreDao) CreateOrUpdate(ctx context.Context, date time.Time, update SystemScoreModel) error {
	_, err := probeDatabase.Collection(TableNameSystemScore).
		UpdateOne(ctx, bson.M{"date": date, "status": consts.StatusValid}, bson.M{"$set": update}, options.Update().SetUpsert(true))
	if err != nil {
		hlog.CtxErrorf(ctx, "update system score model failed, err: %v", err)
		return err
	}
	return nil
}
