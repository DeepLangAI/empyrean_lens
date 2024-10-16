package plugin

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type File struct {
	ID     primitive.ObjectID `bson:"_id" json:"_id"`
	UserID string             `bson:"user_id" json:"user_id"`
	// ParsingResult string             `bson:"parsing_result" json:"parsing_result" default:""`
	// HTMLContent   string             `bson:"html_content" json:"html_content"`
	Name        string    `bson:"name" json:"name"`
	SafeStatus  bool      `bson:"safe_status" json:"safe_status" default:"true"`
	IsDelete    bool      `bson:"is_delete" json:"is_delete" default:"false"`
	IsGenerated bool      `bson:"is_generated" json:"is_generated" default:"false"`
	CreateTime  time.Time `bson:"create_time" json:"create_time"`
	UpdateTime  time.Time `bson:"update_time" json:"update_time"`
}

const TableNameFile = "file"

var fileDao *FileDao

type FileDao struct {
}

var FileDaoOnce sync.Once

func NewFileDao() *FileDao {
	FileDaoOnce.Do(func() {
		fileDao = &FileDao{}
	})
	return fileDao
}

func (d *FileDao) FindFileByUserIdAndCreateTime(ctx context.Context, userId string, startTime, endTime time.Time) ([]File, error) {
	var result []File

	filter := bson.M{"is_delete": false}
	if userId != "" {
		filter["user_id"] = userId
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		filter["create_time"] = bson.M{"$gte": startTime, "$lt": endTime}
	}
	cur, err := pluginCollection.Collection(TableNameFile).Find(ctx, filter)

	if err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByUserIdAndCreateTime] mongo find error:%+v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "[FindFileByUserIdAndCreateTime] mongo all error:%+v", err)
		return nil, err
	}
	return result, nil
}
