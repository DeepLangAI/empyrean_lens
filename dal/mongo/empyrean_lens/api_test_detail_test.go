package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestApiTestDetailDao_Save(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewApiTestDetailDao()

	// 准备测试数据
	model := ApiTestDetailModel{
		Id:      primitive.NewObjectID(),
		TraceId: "test-trace-id8",
		EntryId: "test-entry-id",
		RequestContent: RequestContent{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    `{"test": "data"}`,
			Method:  "POST",
			Url:     "http://test.com/api",
		},
		ResponseContent: ResponseContent{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"result": "success"}`,
			Error:      "",
		},
		CostTime:    100,
		ApiName:     "edu_output",
		CreatedTime: time.Now().AddDate(0, 0, -1),
	}

	// 测试保存
	err := dao.Save(ctx, model)
	assert.NoError(t, err)
}
