package lingo

import (
	"context"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"sync"
	"time"
)

const TableNameFile = "file"

type File struct {
	MultiId        string    `bson:"multi_id" json:"multi_id"`
	CopyFromFileId string    `bson:"copy_from_file_id" json:"copy_from_file_id"`
	CreateTime     time.Time `bson:"create_time" json:"create_time"`
	UpdateTime     time.Time `bson:"update_time" json:"update_time"`
}

type FileDao struct{}

var fileDao *FileDao
var fileDaoInitOnce sync.Once

func NewFileModelDao() *FileDao {
	fileDaoInitOnce.Do(func() {
		fileDao = &FileDao{}
	})
	return fileDao
}

func (self *FileDao) FindModels(ctx context.Context, timeBegin, timeEnd time.Time) ([]File, error) {
	var result []File
	filter := bson.M{"create_time": bson.M{"$gte": timeBegin, "$lt": timeEnd}}
	cur, err := lingoDatabase.
		Collection(TableNameFile).
		Find(ctx, filter)
	if err != nil {
		hlog.CtxErrorf(ctx, "File FindModels error: %v", err)
		return nil, err
	}
	defer cur.Close(ctx)
	if err = cur.All(ctx, &result); err != nil {
		hlog.CtxErrorf(ctx, "File FindModels error: %v", err)
		return nil, err
	}
	return result, nil
}
