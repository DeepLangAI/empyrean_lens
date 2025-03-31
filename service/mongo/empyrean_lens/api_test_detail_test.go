package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	dal_mongo_empyrean_lens "empyrean_lens/dal/mongo/empyrean_lens"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestSaveApiTestDetail(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal_mongo_empyrean_lens.Init(ctx)

	// 准备测试数据
	model := dal_mongo_empyrean_lens.ApiTestDetailModel{
		Id:      primitive.NewObjectID(),
		TraceId: "test-entry-id1",
		EntryId: "test-entry-id1",
		RequestContent: dal_mongo_empyrean_lens.RequestContent{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    `{"test": "data"}`,
			Method:  "POST",
			Url:     "http://test.com/api",
		},
		ResponseContent: dal_mongo_empyrean_lens.ResponseContent{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"result": "success"}`,
			Error:      "",
		},
		CostTime:    100,
		ApiName:     "upload_pdf",
		CreatedTime: time.Now(),
	}

	// 测试保存
	err := SaveApiTestDetail(ctx, model)
	assert.NoError(t, err)
}
