package bi

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"sync"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const TableNameExcel = "link_trace_excel"

var excelDao *ExcelDao

type ExcelDao struct {
}

var ExcelDaoOnce sync.Once

func NewExcelDao() *ExcelDao {
	ExcelDaoOnce.Do(func() {
		excelDao = &ExcelDao{}
	})
	return excelDao
}

func (ed *ExcelDao) getGridfsBucket(collName string) *gridfs.Bucket {
	var bucket *gridfs.Bucket
	bucketOptions := options.GridFSBucket().SetName(collName)
	bucket, _ = gridfs.NewBucket(biCollection, bucketOptions)
	return bucket
}

// 上传文件
func (ed *ExcelDao) GridfsUpload(ctx context.Context, fileName string, fileContent []byte) error {
	// md5 获取文件ID
	fileID := fmt.Sprintf("%x", md5.Sum([]byte(fileName)))
	bucket := ed.getGridfsBucket(TableNameExcel)
	err := bucket.UploadFromStreamWithID(fileID, fileName, bytes.NewBuffer(fileContent))
	if err != nil {
		hlog.CtxErrorf(ctx, "GridfsUploadWithID error: %v", err)
		return err
	}
	return nil
}

// 下载文件
func (ed *ExcelDao) GridfsDownload(ctx context.Context, fileName string) (fileContent []byte, err error) {
	// md5 获取文件ID
	fileID := fmt.Sprintf("%x", md5.Sum([]byte(fileName)))
	bucket := ed.getGridfsBucket(TableNameExcel)
	fileBuffer := bytes.NewBuffer(nil)
	if _, err = bucket.DownloadToStream(fileID, fileBuffer); err != nil {
		hlog.CtxErrorf(ctx, "GridfsDownload error: %v", err)
		return nil, err
	}
	return fileBuffer.Bytes(), nil
}
