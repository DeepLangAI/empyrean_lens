package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"fmt"
	"os"
	"testing"
	"time"

	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"go.mongodb.org/mongo-driver/bson"
)

func TestSceneDao_RmRecentDays(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	err := NewSceneModelDao().RmRecentDays(ctx, 7)
	if err != nil {
		t.Error(err)
	}
}

func TestSceneDao_Modify(t *testing.T) {
	os.Setenv(constslib.ModeEnvName, "pre")
	ctx := context.Background()
	conf.InitConfig()
	// 写入环境变量
	Init(ctx)

	date, _ := time.Parse(consts.DateTemplate, "2024-10-31")
	filter := bson.M{
		"scene": "单文档：智能大纲",
		"date":  date,
	}

	result := probeDatabase.Collection(TableNameScene).FindOneAndUpdate(
		ctx,
		filter,
		bson.M{"$set": bson.M{"slow_cnt": 0}},
	)
	if result.Err() == nil {
		t.Error("modify scene failed")
	}
	var sceneModel SceneModel
	result.Decode(&sceneModel)
	fmt.Printf("%+v\n", sceneModel)
}
