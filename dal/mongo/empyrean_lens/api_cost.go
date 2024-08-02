package empyrean_lens

import (
	"context"
	consts2 "empyrean_lens/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

const TableNameApiCost = "api_cost"

type ApiCostModel struct {
	Date                    time.Time `bson:"date"`
	ApiName                 string    `bson:"api_name"`
	ReqCnt                  int64     `bson:"req_cnt"`
	AvgCost                 float64   `bson:"avg_cost"`
	CostDistribution0_1     float64   `bson:"cost_distribution0_1"`
	CostDistribution1_3     float64   `bson:"cost_distribution1_3"`
	CostDistribution3_5     float64   `bson:"cost_distribution3_5"`
	CostDistribution5_10    float64   `bson:"cost_distribution5_10"`
	CostDistribution10_20   float64   `bson:"cost_distribution10_20"`
	CostDistribution20_30   float64   `bson:"cost_distribution20_30"`
	CostDistribution30_50   float64   `bson:"cost_distribution30_50"`
	CostDistribution50_100  float64   `bson:"cost_distribution50_100"`
	CostDistribution100_inf float64   `bson:"cost_distribution100_inf"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type ApiCostDao struct{}

var apiCostDao *ApiCostDao

var apiCostDaoInitOnce sync.Once

func NewApicostModelDao() *ApiCostDao {
	apiCostDaoInitOnce.Do(func() {
		apiCostDao = &ApiCostDao{}
	})
	return apiCostDao
}

func (self *ApiCostDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameApiCost).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "drop api cost table failed, err: %v", err)
		return err
	}
	return nil
}
func (self *ApiCostDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameApiCost).
		DeleteMany(ctx, bson.M{"date": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete api cost model failed, err: %v", err)
		return err
	}
	return nil

}

func (self *ApiCostDao) Save(ctx context.Context, model ApiCostModel) error {
	_, err := probeDatabase.Collection(TableNameApiCost).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "save api cost model failed, err: %v", err)
		return err
	}
	return nil
}

func (self *ApiCostDao) FindTimespanCost(ctx context.Context, timeBegin, timeEnd time.Time) ([]ApiCostModel, error) {
	var result []ApiCostModel
	cur, err := probeDatabase.
		Collection(TableNameApiCost).
		Find(ctx, bson.M{"date": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts2.StatusValid})
	if err != nil {
		hlog.CtxErrorf(ctx, "find api cost model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find api cost model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}

func (self *ApiCostDao) CreateOrUpdate(ctx context.Context, date time.Time, apiName string, update ApiCostModel) error {
	_, err := probeDatabase.Collection(TableNameApiCost).
		UpdateOne(ctx, bson.M{"date": date, "api_name": apiName, "status": consts2.StatusValid}, bson.M{"$set": update}, options.Update().SetUpsert(true))
	if err != nil {
		hlog.CtxErrorf(ctx, "update api cost model failed, err: %v", err)
		return err
	}
	return nil
}
