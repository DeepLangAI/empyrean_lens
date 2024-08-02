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

const TableNameScene = "scene"

type SceneModel struct {
	Date       time.Time `bson:"date"`
	Scene      string    `bson:"scene"`
	TotalCnt   int32     `bson:"total_cnt"`
	FailCnt    int32     `bson:"fail_cnt"`
	SlowCnt    int32     `bson:"slow_cnt"`
	FailReason string    `bson:"fail_reason"`

	Status     int32     `json:"status" bson:"status"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type SceneDao struct{}

var sceneDao *SceneDao

var sceneDaoInitOnce sync.Once

func NewSceneModelDao() *SceneDao {
	sceneDaoInitOnce.Do(func() {
		sceneDao = &SceneDao{}
	})
	return sceneDao
}

func (self *SceneDao) DropTable(ctx context.Context) error {
	err := probeDatabase.Collection(TableNameScene).Drop(ctx)
	if err != nil {
		hlog.CtxErrorf(ctx, "drop scene table failed, err: %v", err)
		return err
	}
	return nil
}

func (self *SceneDao) RmRecentDays(ctx context.Context, days int) error {
	anchorDay := time.Now().AddDate(0, 0, -days)
	day := time.Date(anchorDay.Year(), anchorDay.Month(), anchorDay.Day(), 0, 0, 0, 0, time.Local)
	_, err := probeDatabase.
		Collection(TableNameApiFailure).
		DeleteMany(ctx, bson.M{"date": bson.M{"$gte": day}})
	if err != nil {
		hlog.CtxErrorf(ctx, "delete scene model failed, err: %v", err)
		return err
	}
	return nil

}

func (self *SceneDao) Save(ctx context.Context, model SceneModel) error {
	_, err := probeDatabase.Collection(TableNameScene).InsertOne(ctx, model)
	if err != nil {
		hlog.CtxErrorf(ctx, "save scene model failed, err: %v", err)
		return err
	}
	return nil
}

func (self *SceneDao) FindTimespanScene(ctx context.Context, timeBegin, timeEnd time.Time) ([]SceneModel, error) {
	var result []SceneModel
	cur, err := probeDatabase.
		Collection(TableNameScene).
		Find(ctx, bson.M{"date": bson.M{"$gte": timeBegin, "$lt": timeEnd}, "status": consts.StatusValid})
	if err != nil {
		hlog.CtxErrorf(ctx, "find scene model failed, err: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "find scene model failed, err: %v", err)
		return nil, err
	}
	return result, nil
}

func (self *SceneDao) CreateOrUpdate(ctx context.Context, date time.Time, scene string, update SceneModel) error {
	_, err := probeDatabase.Collection(TableNameScene).
		UpdateOne(ctx, bson.M{"date": date, "scene": scene, "status": consts.StatusValid}, bson.M{"$set": update}, options.Update().SetUpsert(true))
	if err != nil {
		hlog.CtxErrorf(ctx, "update scene model failed, err: %v", err)
		return err
	}
	return nil
}
