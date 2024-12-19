package plugin

import (
	"context"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MultiAigc struct {
	ID         primitive.ObjectID       `bson:"_id" json:"_id"`
	MultiID    string                   `bson:"multi_id" json:"multi_id"`
	AigcType   int                      `bson:"aigc_type" json:"aigc_type"`
	AigcResult map[string][]interface{} `bson:"aigc_result" json:"aigc_result"`
	IsDeleted  bool                     `bson:"is_deleted" json:"is_deleted"`
	IsSafe     bool                     `bson:"is_safe" json:"is_safe"`
	CreateTime time.Time                `bson:"create_time" json:"create_time"`
	UpdateTime time.Time                `bson:"update_time" json:"update_time"`

	// 不是从数据库获取的
	ArticleList        []ArticleEntry `json:"article_list" bson:"article_list"`
	UserID             string         `json:"user_id" bson:"user_id"`
	Title              string         `json:"title" bson:"title"`
	ChannelType        int            `json:"channel_type" bson:"channel_type"`
	CopyFromResourceID string         `json:"copy_from_resource_id" bson:"copy_from_resource_id"`
}

const TableNameMultiAigc = "multi_aigc"

var multiAigcDao *MultiAigcDao

type MultiAigcDao struct {
}

var MultiAigcDaoOnce sync.Once

func NewMultiAigcDao() *MultiAigcDao {
	MultiAigcDaoOnce.Do(func() {
		multiAigcDao = &MultiAigcDao{}
	})
	return multiAigcDao
}

func (d *MultiAigcDao) QueryByID(ctx context.Context, multiAigcID string) (*MultiAigc, error) {
	var res []*MultiAigc
	_id, _ := primitive.ObjectIDFromHex(multiAigcID)
	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"_id": _id},
	}}
	cur, err := pluginCollection.Collection(TableNameMultiAigc).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[QueryByID] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[QueryByID] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	if len(res) == 0 {
		return nil, nil
	}
	return res[0], nil
}

func (d *MultiAigcDao) FindByTimeRangeForSave(ctx context.Context, startTime, endTime time.Time) ([]*MultiAigc, error) {
	var res []*MultiAigc

	filter := bson.M{
		//"is_delete": false,
		"create_time": bson.M{"$gte": startTime, "$lt": endTime},
		"aigc_type":   3,
	}
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}})
	cur, err := pluginCollection.Collection(TableNameMultiAigc).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindByTimeRangeForSave] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindByTimeRangeForSave] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *MultiAigcDao) QueryFirstAigc(ctx context.Context, multiID, theme string) (*MultiAigc, error) {
	var res []*MultiAigc

	filter := bson.M{
		//"is_delete": false,
		"aigc_type":            3,
		"multi_id":             multiID,
		"aigc_result." + theme: bson.M{"$exists": true},
	}

	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: 1}}).SetLimit(1)
	cur, err := pluginCollection.Collection(TableNameMultiAigc).Find(ctx, filter, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[QueryFirstAigc] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[QueryFirstAigc] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	if len(res) != 0 {
		return res[0], nil
	}
	return nil, nil
}

func (d *MultiAigc) TranslateEntryInfo() *bi.EntryInfo {
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
		EntryType:       int(empyrean_lens.EntryTypeEnum_MULTI_OUTLINE),
		EntrySource:     consts.EntryInfoEntrySourceSummary,
		SourceTable:     TableNameMultiAigc,
		MultiArticles:   multiArticles,
		DataType:        consts.EntryInfoDataTypeSummary,
		UserID:          d.UserID,
		Title:           d.Title,
		ChannelType:     d.ChannelType,
		EntryCreateTime: d.CreateTime,
		EntryUpdateTime: d.UpdateTime,
		CreateTime:      time.Now(),
	}
}
