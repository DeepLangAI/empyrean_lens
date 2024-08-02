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

const TableNameApiFailure = "api_failure"

type ApiFailureModel struct {
	Date          time.Time `bson:"date"`
	ApiName       string    `bson:"api_name"`
	HostName      string    `bson:"host_name"`
	FailCnt       int32     `bson:"fail_cnt"`
	TotalCnt      int32     `bson:"total_cnt"`
	ErrCode3xxCnt int32     `bson:"err_code_3xx_cnt"`
	ErrCode4xxCnt int32     `bson:"err_code_4xx_cnt"`
	ErrCode5xxCnt int32     `bson:"err_code_5xx_cnt"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type ApiFailureDao struct{}

var apiFailureDao *ApiFailureDao

var apiFailureDaoInitOnce sync.Once

func NewApifailureModelDao() *ApiFailureDao {
	apiFailureDaoInitOnce.Do(func() {
		apiFailureDao = &ApiFailureDao{}
	})
	return apiFailureDao
}

func (self *ApiFailureDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameApiFailure).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "drop api failure table failed, err: %v", err)
		return err
	}
	return nil
}

func (self *ApiFailureDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameApiFailure).
		DeleteMany(ctx, bson.M{"date": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete api failure model failed, err: %v", err)
		return err
	}
	return nil

}

func (self *ApiFailureDao) Save(ctx context.Context, model ApiFailureModel) error {
	_, err := probeDatabase.Collection(TableNameApiFailure).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "save api failure model failed, err: %v", err)
		return err
	}
	return nil
}

func (self *ApiFailureDao) FindTimespanFailure(ctx context.Context, timeBegin, timeEnd time.Time) ([]ApiFailureModel, error) {
	var result []ApiFailureModel
	cur, err := probeDatabase.
		Collection(TableNameApiFailure).
		Find(ctx, bson.M{"date": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts.StatusValid})
	if err != nil {
		hlog.CtxErrorf(ctx, "find api failure model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find api failure model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}

func (self *ApiFailureDao) CreateOrUpdate(ctx context.Context, date time.Time, apiName string, update ApiFailureModel) error {
	_, err := probeDatabase.Collection(TableNameApiFailure).
		UpdateOne(ctx, bson.M{"date": date, "api_name": apiName, "status": consts.StatusValid}, bson.M{"$set": update}, options.Update().SetUpsert(true))
	if err != nil {
		hlog.CtxErrorf(ctx, "update api failure model failed, err: %v", err)
		return err
	}
	return nil
}
