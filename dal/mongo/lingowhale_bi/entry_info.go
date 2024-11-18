package bi

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MultiArticles struct {
	EntryId   string `json:"entry_id" bson:"entry_id"`
	EntryType int    `json:"entry_type" bson:"entry_type"`
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
	UserID          string             `json:"user_id" bson:"user_id"`
	UserType        int                `json:"user_type" bson:"user_type"`
	Title           string             `json:"title" bson:"title"`
	ChannelType     int                `json:"channel_type" bson:"channel_type"`
	ParentEntryID   string             `json:"parent_entry_id" bson:"parent_entry_id"`
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

const TableNameEntryInfo = "entry_info"

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

func (d *EntryInfoDao) SaveEntryInfo(ctx context.Context, entryInfo *EntryInfo) error {
	_, err := biCollection.Collection(TableNameEntryInfo).InsertOne(ctx, entryInfo)
	if err != nil {
		hlog.CtxErrorf(ctx, "db error, method:Save EntryInfo, err:%+v", err)
		return err
	}
	return err
}

func (d *EntryInfoDao) FindByEntryIDs(ctx context.Context, entryIDs []string) ([]*EntryInfo, error) {
	var entryInfos []*EntryInfo
	cur, err := biCollection.Collection(TableNameEntryInfo).Find(ctx, bson.M{"entry_id": bson.M{"$in": entryIDs}})
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

func (d *EntryInfoDao) FindByEntryIDAndEntryType(ctx context.Context, entryID string, entryType int) (*EntryInfo, error) {
	entryInfo := &EntryInfo{}
	err := biCollection.Collection(TableNameEntryInfo).FindOne(ctx, bson.M{"entry_id": entryID, "entry_type": entryType}).Decode(entryInfo)
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

func (d *EntryInfoDao) EntryInfoExist(ctx context.Context, entryID string, entryType int) bool {
	info, err := d.FindByEntryIDAndEntryType(ctx, entryID, entryType)
	if err != nil {
		return true
	}
	// 查询不到返回true
	return info != nil
}

func (d *EntryInfo) TranslateUserActionRow() *empyrean_lens.UserActionRespRow {
	resources := []*empyrean_lens.ResourceInfo{}
	for _, v := range d.MultiArticles {
		resources = append(resources, &empyrean_lens.ResourceInfo{
			EntryID:   v.EntryId,
			EntryType: empyrean_lens.EntryTypeEnum(v.EntryType),
		})
	}
	if len(resources) == 0 {
		resources = append(resources, &empyrean_lens.ResourceInfo{
			EntryID:   d.EntryID,
			EntryType: empyrean_lens.EntryTypeEnum(d.EntryType),
		})
	}
	return &empyrean_lens.UserActionRespRow{
		UserID:     d.UserID,
		EntryID:    d.EntryID,
		EntryType:  empyrean_lens.EntryTypeEnum(d.EntryType),
		Channel:    utils.ChannelIntToString(d.ChannelType),
		Title:      d.Title,
		Resources:  resources,
		Cost:       float64(d.Cost / 1000),
		Status:     empyrean_lens.ActionStatusEnum(d.Status),
		ActionName: utils.GetActionName(d.EntryType, d.MultiID),
		CreateTime: d.EntryCreateTime.Local().Format(consts.DateTimeTemplate),
	}
}
