package bi

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserTypeEnum int

const (
	InternalUser UserTypeEnum = 0
	ExternalUser UserTypeEnum = 1
)

type UserInfo struct {
	ID               primitive.ObjectID `bson:"_id" json:"_id"`
	UserID           string             `bson:"user_id" json:"user_id"`
	UserType         UserTypeEnum       `bson:"user_type" json:"user_type"`
	UserPhone        string             `bson:"user_phone" json:"user_phone"`
	UserSource       int                `bson:"user_source" json:"user_source"`
	PromotionChannel string             `bson:"promotion_channel" json:"promotion_channel"`
	RegisterTime     time.Time          `bson:"register_time" json:"register_time"`
	RegisterStatus   int                `bson:"register_status" json:"register_status"`
	IsTest           bool               `bson:"is_test" json:"is_test"`
	IsVip            bool               `bson:"is_vip" json:"is_vip"`
}

const TableNameUserInfo = "userInfo"

var userInfoDao *UserInfoDao

type UserInfoDao struct {
}

var UserInfoDaoOnce sync.Once

func NewUserInfoDao() *UserInfoDao {
	UserInfoDaoOnce.Do(func() {
		userInfoDao = &UserInfoDao{}
	})
	return userInfoDao
}

func (d *UserInfoDao) FindFileByUids(ctx context.Context, uids []string) ([]*UserInfo, error) {
	var res []*UserInfo

	filter := bson.M{"$and": []bson.M{
		//{"is_delete": false},
		{"user_id": bson.M{"$in": uids}},
	}}
	cur, err := biCollection.Collection(TableNameUserInfo).Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByUids] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)

	if err = cur.All(ctx, &res); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByUids] mongo all error:%+v", err)
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	for _, v := range res {
		v.RegisterTime = v.RegisterTime.Local()
	}
	return res, nil
}
