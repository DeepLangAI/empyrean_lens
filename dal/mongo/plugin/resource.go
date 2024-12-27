package plugin

import (
	"context"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Resource struct {
	ID          primitive.ObjectID `bson:"_id" json:"_id"`
	EntryType   int                `bson:"entry_type" json:"entry_type"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	AuthorName  string             `bson:"author_name" json:"author_name"`
	UserID      string             `bson:"user_id" json:"user_id"`
	OrigUrl     string             `bson:"orig_url" json:"orig_url"`
	NovelFormID string             `bson:"novel_form_id" json:"novel_form_id"`
	IsSafe      bool               `bson:"is_safe" json:"is_safe"`
	Status      int                `bson:"status" json:"status"`
	Source      int                `bson:"source" json:"source"`
	PubTime     time.Time          `bson:"pub_time" json:"pub_time" default:"false"`
	CreateTime  time.Time          `bson:"create_time" json:"create_time"`
	UpdateTime  time.Time          `bson:"update_time" json:"update_time"`

	// 不从数据库查询的字段
	ArticleList []ArticleEntry `bson:"article_list" json:"article_list"`
}

const TableNameResource = "resource"

var resourceDao *ResourceDao

type ResourceDao struct {
}

var ResourceDaoOnce sync.Once

func NewResourceDao() *ResourceDao {
	ResourceDaoOnce.Do(func() {
		resourceDao = &ResourceDao{}
	})
	return resourceDao
}

func (d *ResourceDao) FindResourceById(ctx context.Context, id string) (*Resource, error) {
	var res []*Resource

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"_id": _id},
	}}
	cur, err := pluginCollection.Collection(TableNameResource).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceById] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceById] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	res[0].CreateTime = res[0].CreateTime.Local()
	res[0].UpdateTime = res[0].UpdateTime.Local()
	res[0].EntryType = TranslateEntryType(res[0].EntryType)
	return res[0], nil
}

func (d *ResourceDao) FindResourceByIds(ctx context.Context, ids []string) (map[string]*Resource, error) {
	var res []*Resource

	_ids := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		_id, _ := primitive.ObjectIDFromHex(id)
		_ids = append(_ids, _id)
	}
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"_id": bson.M{"$in": _ids}},
	}}
	cur, err := pluginCollection.Collection(TableNameResource).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceByIds] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceByIds] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return map[string]*Resource{}, nil
	}
	sourceMapping := make(map[string]*Resource)
	for _, v := range res {
		v.CreateTime = v.CreateTime.Local()
		v.UpdateTime = v.UpdateTime.Local()
		v.EntryType = TranslateEntryType(v.EntryType)
		sourceMapping[v.ID.Hex()] = v
	}
	return sourceMapping, nil
}

func (d *ResourceDao) FindResourceByTimeRange(ctx context.Context, startTime, endTime time.Time, skip, limit int64) ([]*Resource, error) {
	var res []*Resource

	filter := bson.M{
		//"is_delete": false,
		"create_time": bson.M{"$gte": startTime, "$lt": endTime},
		"status":      consts.SubscribeSuccessStatus,
	}
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameResource).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceByTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceByTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
		res[i].EntryType = TranslateEntryType(res[i].EntryType)
	}
	return res, nil
}

func (d *ResourceDao) FindResourceByTimeRangeForSave(ctx context.Context, entryType int, startTime, endTime time.Time) ([]*Resource, error) {
	var res []*Resource

	filter := bson.M{
		//"is_delete": false,
		"create_time": bson.M{"$gte": startTime, "$lt": endTime},
		"entry_type":  TranslateSubscribeEntryType(entryType),
		"status":      consts.SubscribeSuccessStatus,
	}
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}})
	cur, err := pluginCollection.Collection(TableNameResource).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceByTimeRangeForSave] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindResourceByTimeRangeForSave] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
		res[i].EntryType = TranslateEntryType(res[i].EntryType)
	}
	return res, nil
}

func (d *Resource) TranslateEntryInfo() *bi.EntryInfo {
	multiArticles := []bi.MultiArticles{}
	for _, article := range d.ArticleList {
		multiArticles = append(multiArticles, bi.MultiArticles{
			EntryId:   article.EntryId,
			EntryType: int(article.EntryType),
		})
	}
	return &bi.EntryInfo{
		ID:              primitive.NewObjectID(),
		EntryID:         d.ID.Hex(),
		EntryType:       d.EntryType,
		EntrySource:     consts.EntryInfoEntrySourceSubscribe,
		SourceTable:     TableNameResource,
		DataType:        consts.EntryInfoDataTypeSubscribe,
		MultiArticles:   multiArticles,
		Title:           d.Title,
		EntryURL:        d.OrigUrl,
		Status:          int(utils.GetActionStatus(empyrean_lens.EntryTypeEnum(d.EntryType), d.Status, 0, 0)),
		EntryCreateTime: d.CreateTime,
		EntryUpdateTime: d.UpdateTime,
		CreateTime:      time.Now(),
	}
}

// 转换 基础类型 到 订阅类型
func TranslateEntryType(entryType int) int {
	switch entryType {
	case int(empyrean_lens.EntryTypeEnum_WEB):
		return int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB)
	case int(empyrean_lens.EntryTypeEnum_FILE):
		return int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE)
	case int(empyrean_lens.EntryTypeEnum_MULTI):
		return int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI)
	}
	return entryType
}

// 转换 订阅类型 到 基础类型
func TranslateSubscribeEntryType(entryType int) int {
	switch entryType {
	case int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB):
		return int(empyrean_lens.EntryTypeEnum_WEB)
	case int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE):
		return int(empyrean_lens.EntryTypeEnum_FILE)
	case int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI):
		return int(empyrean_lens.EntryTypeEnum_MULTI)
	}
	return entryType
}
