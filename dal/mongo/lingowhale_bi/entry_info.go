package bi

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/utils"

	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MultiArticles struct {
	EntryId   string `json:"entry_id" bson:"entry_id"`
	EntryType int    `json:"entry_type" bson:"entry_type"`
}

type SourceEntryInfo struct {
	EntryID       string          `json:"entry_id" bson:"entry_id"`
	EntryType     int             `json:"entry_type" bson:"entry_type"`
	MultiID       string          `json:"multi_id" bson:"multi_id"`
	MultiArticles []MultiArticles `json:"multi_articles" bson:"multi_articles"`
}

type EntryInfo struct {
	ID              primitive.ObjectID `bson:"_id" json:"_id"`
	EntryID         string             `json:"entry_id" bson:"entry_id"`
	EntryType       int                `json:"entry_type" bson:"entry_type"`
	EntrySource     int                `json:"entry_source" bson:"entry_source"`
	SourceTable     string             `json:"source_table" bson:"source_table"`
	DataType        int                `json:"data_type" bson:"data_type"`
	MultiID         string             `json:"multi_id" bson:"multi_id"`
	MultiArticles   []MultiArticles    `json:"multi_articles" bson:"multi_articles"`
	SourceEntryInfo SourceEntryInfo    `json:"source_entry_info" bson:"source_entry_info"`
	WebSite         string             `json:"web_site" bson:"web_site"`
	ActionName      string             `json:"action_name" bson:"action_name"`
	Language        string             `json:"language" bson:"language"`
	UserID          string             `json:"user_id" bson:"user_id"`
	UserType        int                `json:"user_type" bson:"user_type"`
	Title           string             `json:"title" bson:"title"`
	ChannelType     int                `json:"channel_type" bson:"channel_type"`
	OutlineType     int                `json:"outline_type" bson:"outline_type"`
	ParentEntryID   string             `json:"parent_entry_id" bson:"parent_entry_id"`
	ParentEntryType int                `json:"parent_entry_type" bson:"parent_entry_type"`
	EntryURL        string             `json:"entry_url" bson:"entry_url"`
	Status          int                `json:"status" bson:"status"`
	LinkStatus      int                `json:"link_status" bson:"link_status"`
	FailedAction    string             `json:"failed_action" bson:"failed_action"`
	Cost            int                `json:"cost" bson:"cost"`
	ContentSize     int                `json:"content_size" bson:"content_size"`
	EntryCreateTime time.Time          `json:"entry_create_time" bson:"entry_create_time"`
	EntryUpdateTime time.Time          `json:"entry_update_time" bson:"entry_update_time"`
	CreateTime      time.Time          `json:"create_time" bson:"create_time"`
}

var entryInfoDao *EntryInfoDao

type EntryInfoDao struct {
}

var EntryInfoDaoOnce sync.Once

func NewEntryInfoDao() *EntryInfoDao {
	EntryInfoDaoOnce.Do(func() {
		entryInfoDao = &EntryInfoDao{}
	})
	return entryInfoDao
}

func TableNameEntryInfo() string {
	env := os.Getenv(constslib.ModeEnvName)
	if env == "" {
		env = "test"
	}
	if env == "test" {
		return "entry_info_test"
	}
	return "entry_info"
}

func (d *EntryInfoDao) SaveEntryInfo(ctx context.Context, entryInfo *EntryInfo) error {
	// 是否存在
	info, err := d.FindByEntryIDAndEntryType(ctx, entryInfo.EntryID, entryInfo.EntryType)
	if err != nil {
		hlog.CtxErrorf(ctx, "db error, method:Save EntryInfo, err:%+v", err)
		return err
	}
	// 存在，upload
	if info != nil {
		filter := bson.M{"entry_id": entryInfo.EntryID, "entry_type": entryInfo.EntryType}
		update := bson.M{"action_name": entryInfo.ActionName, "web_site": entryInfo.WebSite, "title": entryInfo.Title, "multi_id": entryInfo.MultiID, "status": entryInfo.Status, "outline_type": entryInfo.OutlineType, "link_status": entryInfo.LinkStatus, "entry_url": entryInfo.EntryURL, "failed_action": entryInfo.FailedAction, "cost": entryInfo.Cost, "multi_articles": entryInfo.MultiArticles, "parent_entry_id": entryInfo.ParentEntryID, "source_entry_info": entryInfo.SourceEntryInfo, "entry_create_time": entryInfo.EntryCreateTime}
		_, err := biCollection.Collection(TableNameEntryInfo()).UpdateOne(ctx, filter, bson.M{"$set": update})
		if err != nil {
			hlog.CtxErrorf(ctx, "db error, method:Save EntryInfo, err:%+v", err)
			return err
		}
		return nil
	}
	// 不存在，插入
	_, err = biCollection.Collection(TableNameEntryInfo()).InsertOne(ctx, entryInfo)
	if err != nil {
		hlog.CtxErrorf(ctx, "db error, method:Save EntryInfo, err:%+v", err)
		return err
	}
	return err
}

func (d *EntryInfoDao) FindByEntryIDs(ctx context.Context, entryIDs []string) ([]*EntryInfo, error) {
	var entryInfos []*EntryInfo
	cur, err := biCollection.Collection(TableNameEntryInfo()).Find(ctx, bson.M{"entry_id": bson.M{"$in": entryIDs}})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByEntryIDs, err:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &entryInfos); err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryIDs] mongo all error:%+v", err)
		return nil, err
	}
	return entryInfos, nil
}

func (d *EntryInfoDao) FindByTimeRange(ctx context.Context, status []int32, webSites, actionNames []string, onlyOuter bool, startTime, endTime time.Time, skip, limit int64) ([]*EntryInfo, error) {
	var entryInfos []*EntryInfo
	filter := bson.M{
		"entry_create_time": bson.M{"$gte": startTime, "$lt": endTime},
	}
	if len(status) > 0 {
		filter["link_status"] = bson.M{"$in": status}
	}
	if len(webSites) > 0 {
		filter["web_site"] = bson.M{"$in": webSites}
	}
	if len(actionNames) > 0 {
		filter["action_name"] = bson.M{"$in": actionNames}
	}
	if onlyOuter {
		filter["user_type"] = 1
	}
	options := options.Find().SetSort(bson.D{{Key: "entry_create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := biCollection.Collection(TableNameEntryInfo()).Find(ctx, filter, options)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByTimeRange, err:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &entryInfos); err != nil {
		hlog.CtxErrorf(ctx, "[FindByTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	return entryInfos, nil
}

func (d *EntryInfoDao) FindByQueryAndTimeRange(ctx context.Context, query string, status []int32, webSites, actionNames []string, onlyOuter bool, startTime, endTime time.Time, skip, limit int64) ([]*EntryInfo, error) {
	var entryInfos []*EntryInfo
	queryFilter := bson.M{"$or": []bson.M{{"title": bson.M{"$regex": query, "$options": "i"}}, {"user_id": query}, {"entry_url": query}, {"multi_articles.entry_id": query}, {"entry_id": query}, {"source_entry_info.entry_id": query}}}
	filter := bson.M{"$and": []bson.M{
		queryFilter,
		{"entry_create_time": bson.M{"$gte": startTime, "$lt": endTime}},
	}}
	if len(status) > 0 {
		filter["link_status"] = bson.M{"$in": status}
	}
	if len(webSites) > 0 {
		filter["web_site"] = bson.M{"$in": webSites}
	}
	if len(actionNames) > 0 {
		filter["action_name"] = bson.M{"$in": actionNames}
	}
	if onlyOuter {
		filter["user_type"] = 1
	}
	options := options.Find().SetSort(bson.D{{Key: "entry_create_time", Value: -1}}).SetLimit(limit).SetSkip(skip)
	cur, err := biCollection.Collection(TableNameEntryInfo()).Find(ctx, filter, options)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByQueryAndTimeRange, err:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &entryInfos); err != nil {
		hlog.CtxErrorf(ctx, "[FindByQueryAndTimeRange] mongo all error:%+v", err)
		return nil, err
	}
	return entryInfos, nil
}

func (d *EntryInfoDao) FindByEntryIDsWithoutCopy(ctx context.Context, entryIDs []string) ([]*EntryInfo, error) {
	var entryInfos []*EntryInfo
	filter := bson.M{"$and": []bson.M{
		{"parent_entry_id": ""},
		{"entry_id": bson.M{"$in": entryIDs}},
	}}
	cur, err := biCollection.Collection(TableNameEntryInfo()).Find(ctx, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByEntryIDsWithoutCopy, err:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &entryInfos); err != nil {
		hlog.CtxErrorf(ctx, "[FindByEntryIDsWithoutCopy] mongo all error:%+v", err)
		return nil, err
	}
	return entryInfos, nil
}

func (d *EntryInfoDao) FindByEntryIDAndEntryType(ctx context.Context, entryID string, entryType int) (*EntryInfo, error) {
	entryInfo := &EntryInfo{}
	err := biCollection.Collection(TableNameEntryInfo()).FindOne(ctx, bson.M{"entry_id": entryID, "entry_type": entryType}).Decode(entryInfo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		hlog.CtxErrorf(ctx, "db error, method:FindByEntryIDAndSourceTable, err:%+v", err)
		return nil, err
	}
	hlog.CtxInfof(ctx, "FindByEntryIDAndSourceTable, entryInfo exist entryId:%s", entryID)
	return entryInfo, nil
}

func (d *EntryInfo) TranslateUserActionRow() *empyrean_lens.UserActionRespRow {
	resources := []*empyrean_lens.ResourceInfo{}
	for _, v := range d.MultiArticles {
		resources = append(resources, &empyrean_lens.ResourceInfo{
			EntryID:   v.EntryId,
			EntryType: empyrean_lens.EntryTypeEnum(v.EntryType),
		})
	}
	if d.SourceEntryInfo.EntryID != "" {
		resources = append(resources, &empyrean_lens.ResourceInfo{
			EntryID:   d.SourceEntryInfo.EntryID,
			EntryType: empyrean_lens.EntryTypeEnum(d.SourceEntryInfo.EntryType),
		})
	}
	if len(resources) == 0 {
		resources = append(resources, &empyrean_lens.ResourceInfo{
			EntryID:   d.EntryID,
			EntryType: empyrean_lens.EntryTypeEnum(d.EntryType),
		})
	}
	// 状态转换，未执行认为是失败
	status := empyrean_lens.ActionStatusEnum(d.LinkStatus)
	if status == empyrean_lens.ActionStatusEnum_UNREACHEAD {
		status = empyrean_lens.ActionStatusEnum_FAIL
	}
	// user id
	if d.UserID == "" {
		d.UserID = "resource_server"
	}
	// 订阅ID
	copyFromResourceID := ""
	if d.ParentEntryID != "" && utils.IsSubscribe(d.ParentEntryType) {
		copyFromResourceID = d.ParentEntryID
	}
	return &empyrean_lens.UserActionRespRow{
		UserID:     d.UserID,
		EntryID:    d.EntryID,
		EntryType:  empyrean_lens.EntryTypeEnum(d.EntryType),
		Channel:    utils.ChannelIntToString(d.ChannelType),
		Title:      d.Title,
		Resources:  resources,
		Cost:       float64(d.Cost) / 1000,
		Status:     empyrean_lens.ActionStatusEnum(d.LinkStatus),
		ActionName: utils.GetActionName(d.EntryType, d.MultiID, copyFromResourceID, d.Language, d.OutlineType),
		CreateTime: d.EntryCreateTime.Local().Format(consts.DateTimeTemplate),
	}
}
