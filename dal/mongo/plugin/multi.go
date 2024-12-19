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

const TableNameMulti = "multi"

type ArticleEntry struct {
	EntryId   string           `json:"entry_id" bson:"entry_id"`
	EntryType consts.EntryType `json:"entry_type" bson:"entry_type"`
	Title     string           `json:"title" bson:"title"`
}

type MultiModel struct {
	ID                 primitive.ObjectID `bson:"_id" json:"id"`
	UserID             string             `json:"user_id" bson:"user_id" validate:"required"`
	ArticleList        []ArticleEntry     `json:"article_list" bson:"article_list"`
	Title              string             `json:"title" bson:"title"`
	CopyFromMultiID    string             `bson:"copy_from_multi_id" json:"copy_from_multi_id"`
	CopyFromResourceID string             `bson:"copy_from_resource_id" json:"copy_from_resource_id"`
	AnalysisStatus     int                `json:"analysis_status" bson:"analysis_status"`
	MergeStatus        int                `json:"merge_status" bson:"merge_status"`
	SummaryStatus      int                `json:"summary_status" bson:"summary_status"`
	ChannelType        int                `bson:"channel_type" json:"channel_type"`
	CreateTime         time.Time          `json:"create_time" bson:"create_time"`
	UpdateTime         time.Time          `json:"update_time" bson:"update_time"`
	IsDeleted          bool               `json:"is_deleted" bson:"is_deleted"`
}

var multiDao *MultiDao

type MultiDao struct {
}

var MultiDaoOnce sync.Once

func NewMultiDao() *MultiDao {
	MultiDaoOnce.Do(func() {
		multiDao = &MultiDao{}
	})
	return multiDao
}

func (d *MultiDao) FindMultiById(ctx context.Context, id string) (*MultiModel, error) {
	var res []*MultiModel

	_id, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"$and": []bson.M{
		// {"is_deleted": false},
		{"_id": _id}},
	}
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiById] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiById] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	res[0].CreateTime = res[0].CreateTime.Local()
	res[0].UpdateTime = res[0].UpdateTime.Local()
	return res[0], nil
}

func (d *MultiDao) FindMultiByIds(ctx context.Context, ids []string) ([]*MultiModel, error) {
	var res []*MultiModel

	objIDs := []primitive.ObjectID{}
	for _, id := range ids {
		_id, _ := primitive.ObjectIDFromHex(id)
		objIDs = append(objIDs, _id)
	}
	filter := bson.M{"$and": []bson.M{
		// {"is_deleted": false},
		{"_id": bson.M{"$in": objIDs}}},
	}
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiById] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiById] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	for _, r := range res {
		r.CreateTime = r.CreateTime.Local()
		r.UpdateTime = r.UpdateTime.Local()
	}
	return res, nil
}

func (d *MultiDao) FindMultiByQueryAndStatusAndTimeRange(ctx context.Context, query string, startTime, endTime time.Time, skip, limit int64) ([]*MultiModel, error) {
	var res []*MultiModel
	queryFilter := []bson.M{}
	if query != "" {
		_id, _ := primitive.ObjectIDFromHex(query)
		queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"title": bson.M{"$regex": query, "$options": "i"}}, {"user_id": query}, {"article_list.entry_id": query}, {"_id": _id}}})
	}
	queryFilter = append(queryFilter, bson.M{"create_time": bson.M{"$gte": startTime, "$lt": endTime}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_multi_id": bson.M{"$exists": false}}, {"copy_from_multi_id": ""}}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_resource_id": bson.M{"$exists": false}}, {"copy_from_resource_id": ""}}})
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, bson.M{"$and": queryFilter}, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *MultiDao) FindFailMultiByQueryAndStatusAndTimeRange(ctx context.Context, query string, startTime, endTime time.Time, skip, limit int64) ([]*MultiModel, error) {
	var res []*MultiModel
	queryFilter := []bson.M{}
	if query != "" {
		_id, _ := primitive.ObjectIDFromHex(query)
		queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"title": bson.M{"$regex": query, "$options": "i"}}, {"user_id": query}, {"article_list.entry_id": query}, {"_id": _id}}})
	}
	queryFilter = append(queryFilter, bson.M{"create_time": bson.M{"$gte": startTime, "$lt": endTime}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_multi_id": bson.M{"$exists": false}}, {"copy_from_multi_id": ""}}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_resource_id": bson.M{"$exists": false}}, {"copy_from_resource_id": ""}}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{
		{"analysis_status": bson.M{"$in": []int{0, 1, 3, 4}}},
		{"merge_status": bson.M{"$in": []int{0, 1, 3, 4}}},
		{"summary_status": bson.M{"$in": []int{0, 1, 3, 4}}},
	}})
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, bson.M{"$and": queryFilter}, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *MultiDao) FindSuccessMultiByQueryAndStatusAndTimeRange(ctx context.Context, query string, startTime, endTime time.Time, skip, limit int64) ([]*MultiModel, error) {
	var res []*MultiModel
	queryFilter := []bson.M{}
	if query != "" {
		_id, _ := primitive.ObjectIDFromHex(query)
		queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"title": bson.M{"$regex": query, "$options": "i"}}, {"user_id": query}, {"article_list.entry_id": query}, {"_id": _id}}})
	}
	queryFilter = append(queryFilter, bson.M{"create_time": bson.M{"$gte": startTime, "$lt": endTime}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_multi_id": bson.M{"$exists": false}}, {"copy_from_multi_id": ""}}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_resource_id": bson.M{"$exists": false}}, {"copy_from_resource_id": ""}}})
	queryFilter = append(queryFilter, bson.M{"$and": []bson.M{
		{"analysis_status": bson.M{"$in": []int{2}}},
		{"merge_status": bson.M{"$in": []int{2}}},
		{"summary_status": bson.M{"$in": []int{2}}},
	}})
	options := options.Find().SetSort(bson.D{{Key: "create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, bson.M{"$and": queryFilter}, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByQueryAndTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *MultiDao) FindMultiByTimeRangeForSave(ctx context.Context, startTime, endTime time.Time) ([]*MultiModel, error) {
	var res []*MultiModel
	queryFilter := []bson.M{}
	queryFilter = append(queryFilter, bson.M{"update_time": bson.M{"$gte": startTime, "$lt": endTime}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_multi_id": bson.M{"$exists": false}}, {"copy_from_multi_id": ""}}})
	queryFilter = append(queryFilter, bson.M{"$or": []bson.M{{"copy_from_resource_id": bson.M{"$exists": false}}, {"copy_from_resource_id": ""}}})
	options := options.Find().SetSort(bson.D{{Key: "update_time", Value: -1}})
	cur, err := pluginCollection.Collection(TableNameMulti).Find(ctx, bson.M{"$and": queryFilter}, options)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByTimeRangeForSave] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindMultiByTimeRangeForSave] mongo all error:%+v", err)
		return nil, err
	}
	for i := 0; i < len(res); i++ {
		res[i].CreateTime = res[i].CreateTime.Local()
		res[i].UpdateTime = res[i].UpdateTime.Local()
	}
	return res, nil
}

func (d *MultiModel) TranslateEntryInfo() *bi.EntryInfo {
	multiArticles := []bi.MultiArticles{}
	for _, article := range d.ArticleList {
		multiArticles = append(multiArticles, bi.MultiArticles{
			EntryId:   article.EntryId,
			EntryType: int(article.EntryType),
		})
	}
	parentEntryID := ""
	parentEntryType := int(empyrean_lens.EntryTypeEnum_MULTI)
	if d.CopyFromMultiID != "" {
		parentEntryID = d.CopyFromMultiID
		parentEntryType = int(empyrean_lens.EntryTypeEnum_MULTI)
	}
	if d.CopyFromResourceID != "" {
		parentEntryID = d.CopyFromResourceID
		parentEntryType = int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI)
	}
	return &bi.EntryInfo{
		ID:              primitive.NewObjectID(),
		EntryID:         d.ID.Hex(),
		EntryType:       int(empyrean_lens.EntryTypeEnum_MULTI),
		EntrySource:     utils.GetEntrySource(d.UserID, d.CopyFromResourceID, d.CopyFromMultiID),
		SourceTable:     TableNameFile,
		DataType:        utils.GetDataType("", d.CopyFromResourceID, empyrean_lens.EntryTypeEnum_FILE),
		MultiArticles:   multiArticles,
		UserID:          d.UserID,
		Title:           d.Title,
		ChannelType:     d.ChannelType,
		ParentEntryID:   parentEntryID,
		ParentEntryType: parentEntryType,
		Status:          int(utils.GetActionStatus(empyrean_lens.EntryTypeEnum_MULTI, d.AnalysisStatus, d.SummaryStatus, d.MergeStatus)),
		Cost:            0, // TODO
		EntryCreateTime: d.CreateTime,
		EntryUpdateTime: d.UpdateTime,
		CreateTime:      time.Now(),
	}
}
