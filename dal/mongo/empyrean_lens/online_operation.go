package empyrean_lens

import (
	"context"
	consts2 "empyrean_lens/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

var TableNameOnlineOperation = ""

type OnlineOperationModel struct {
	Time    time.Time `bson:"time"`
	AppName string    `bson:"app_name"`

	Builder       string `bson:"builder"`
	BranchName    string `bson:"branch_name"`
	CommitMessage string `bson:"commit_message"`
	DomainName    string `bson:"domain_name"`
	RemoteUrl     string `bson:"remote_url"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type OnlineOperationDao struct{}

var onlineOperationDao *OnlineOperationDao

var onlineOperationDaoInitOnce sync.Once

func NewOnlineOperationModelDao() *OnlineOperationDao {
	onlineOperationDaoInitOnce.Do(func() {
		onlineOperationDao = &OnlineOperationDao{}
	})
	return onlineOperationDao
}

func (self *OnlineOperationDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameOnlineOperation).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "drop api cost table failed, err: %v", err)
		return err
	}
	return nil
}
func (self *OnlineOperationDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameOnlineOperation).
		DeleteMany(ctx, bson.M{"date": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete api cost model failed, err: %v", err)
		return err
	}
	return nil
}

func (u *OnlineOperationDao) UpsertMany(ctx context.Context, models []OnlineOperationModel) error {
	// 构建批量操作
	var operations []mongo.WriteModel

	for _, model := range models {

		// 构建查询条件，判断是否已经存在相同的记录
		filter := bson.D{
			{Key: "time", Value: model.Time},
			{Key: "app_name", Value: model.AppName},
		}

		// 构建更新操作
		update := bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "builder", Value: model.Builder},              // 只更新需要的字段
				{Key: "branch_name", Value: model.BranchName},       // 只更新需要的字段
				{Key: "commit_message", Value: model.CommitMessage}, // 只更新需要的字段
				{Key: "domain_name", Value: model.DomainName},       // 只更新需要的字段
				{Key: "remote_url", Value: model.RemoteUrl},         // 只更新需要的字段
				{Key: "status", Value: model.Status},                // 只更新需要的字段
				{Key: "update_time", Value: time.Now().Local()},     // 始终更新:更新时间
			}},
			{Key: "$setOnInsert", Value: bson.D{
				{Key: "create_time", Value: time.Now().Local()}, // 仅在插入时设置:创建时间
			}},
		}

		// 构建 upsert 操作
		updateOneModel := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)

		// 添加到操作集合
		operations = append(operations, updateOneModel)
	}
	if len(operations) == 0 {
		return nil
	}
	// 执行批量操作
	_, err := probeDatabase.Collection(TableNameOnlineOperation).BulkWrite(ctx, operations)
	if err != nil {
		hlog.CtxErrorf(ctx, "mongo onlineOperation upsertMany err: %v", err)
		return err
	}
	return nil
}

func (self *OnlineOperationDao) Save(ctx context.Context, model OnlineOperationModel) error {
	_, err := probeDatabase.Collection(TableNameOnlineOperation).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "save api cost model failed, err: %v", err)
		return err
	}
	return nil
}

func (self *OnlineOperationDao) FindTimespanModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]OnlineOperationModel, error) {
	var result []OnlineOperationModel
	cur, err := probeDatabase.
		Collection(TableNameOnlineOperation).
		Find(ctx, bson.M{"time": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts2.StatusValid})
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

func (self *OnlineOperationDao) CreateOrUpdate(ctx context.Context, update OnlineOperationModel) error {
	_, err := probeDatabase.Collection(TableNameOnlineOperation).
		UpdateOne(
			ctx,
			bson.M{"time": update.Time, "app_name": update.AppName, "status": consts2.StatusValid},
			bson.M{"$set": update},
			options.Update().SetUpsert(true),
		)
	if err != nil {
		hlog.CtxErrorf(ctx, "update api cost model failed, err: %v", err)
		return err
	}
	return nil
}
