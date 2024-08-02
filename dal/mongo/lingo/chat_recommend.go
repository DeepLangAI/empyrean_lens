package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameChatRecommend = "chat_recommend"

type ChatRecommend struct {
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type ChatRecommendDao struct{}

var chatRecommendDao *ChatRecommendDao
var chatRecommendDaoInitOnce sync.Once

func NewChatRecommendModelDao() *ChatRecommendDao {
	chatRecommendDaoInitOnce.Do(func() {
		chatRecommendDao = &ChatRecommendDao{}
	})
	return chatRecommendDao
}

func (self *ChatRecommendDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]ChatRecommend, error) {
	var result []ChatRecommend
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	cur, err := lingoDatabase.
		Collection(TableNameChatRecommend).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "ChatRecommend FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "ChatRecommend FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
