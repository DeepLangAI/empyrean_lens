package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameChat = "chat"

type Chat struct {
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type ChatDao struct{}

var chatDao *ChatDao
var chatDaoInitOnce sync.Once

func NewChatModelDao() *ChatDao {
	chatDaoInitOnce.Do(func() {
		chatDao = &ChatDao{}
	})
	return chatDao
}

func (self *ChatDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]Chat, error) {
	var result []Chat
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	cur, err := lingoDatabase.
		Collection(TableNameChat).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "Chat FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "Chat FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
