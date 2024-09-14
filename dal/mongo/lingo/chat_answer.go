package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameChatAnswer = "chat_answer"

type ChatAnswer struct {
	CreateTime time.Time `bson:"create_time" json:"create_time"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

type ChatAnswerDao struct{}

var chatAnswerDao *ChatAnswerDao
var chatAnswerDaoInitOnce sync.Once

func NewChatAnswerModelDao() *ChatAnswerDao {
	chatAnswerDaoInitOnce.Do(func() {
		chatAnswerDao = &ChatAnswerDao{}
	})
	return chatAnswerDao
}

func (self *ChatAnswerDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]ChatAnswer, error) {
	var result []ChatAnswer
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	cur, err := lingoDatabase.
		Collection(TableNameChatAnswer).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "ChatAnswer FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "ChatAnswer FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
