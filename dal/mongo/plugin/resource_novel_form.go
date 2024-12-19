package plugin

import (
	"context"
	"sync"
	"time"

	"empyrean_lens/consts"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResourceNovelForm struct {
	ID           primitive.ObjectID `bson:"_id" json:"_id"`
	UserID       string             `bson:"user_id" json:"user_id"`
	Title        string             `bson:"title" json:"title"`
	Description  string             `bson:"description" json:"description"`
	SurfaceImg   string             `bson:"surface_img" json:"surface_img"`
	BindEntry    ArticleEntry       `bson:"bind_entry" json:"bind_entry"`
	RelatedEntry []ArticleEntry     `bson:"related_entry" json:"related_entry"`
	Status       int                `bson:"status" json:"status"`
	CreateTime   time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime   time.Time          `bson:"update_time" json:"update_time"`
}

const TableNameResourceNovelForm = "resource_novel_form"

var resourceNovelFormDao *ResourceNovelFormDao

type ResourceNovelFormDao struct {
}

var ResourceNovelFormDaoOnce sync.Once

func NewResourceNovelFormDao() *ResourceNovelFormDao {
	ResourceNovelFormDaoOnce.Do(func() {
		resourceNovelFormDao = &ResourceNovelFormDao{}
	})
	return resourceNovelFormDao
}

func (d *ResourceNovelFormDao) FindResourceNovelFormById(ctx context.Context, id string) (*ResourceNovelForm, error) {
	var res []*ResourceNovelForm

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"_id": _id},
	}}
	cur, err := pluginCollection.Collection(TableNameResourceNovelForm).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceNovelFormById] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceNovelFormById] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	res[0].CreateTime = res[0].CreateTime.Local()
	res[0].UpdateTime = res[0].UpdateTime.Local()
	for idx := range res[0].RelatedEntry {
		entryType := res[0].RelatedEntry[idx].EntryType
		res[0].RelatedEntry[idx].EntryType = consts.EntryType(TranslateEntryType(int(entryType)))
	}
	return res[0], nil
}
