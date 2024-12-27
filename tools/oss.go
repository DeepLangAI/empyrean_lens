package tools

import (
	"archive/zip"
	"bytes"
	"context"
	"empyrean_lens/conf"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type OssOperator struct {
	ctx       context.Context
	ossClient *oss.Client
}

var ossOperator *OssOperator

func GetOssOperator(ctx context.Context) *OssOperator {
	if ossOperator == nil {
		ossOperator = &OssOperator{
			ctx: ctx,
		}
		ossOperator.initOssClient()
	}
	return ossOperator
}

func (o *OssOperator) initOssClient() error {
	ossConfig := conf.GetConfig().Oss
	os.Setenv("OSS_ACCESS_KEY_ID", ossConfig.AccessKey)
	os.Setenv("OSS_ACCESS_KEY_SECRET", ossConfig.AccessSecret)
	os.Setenv("OSS_SESSION_TOKEN", "")

	provider, err := oss.NewEnvironmentVariableCredentialsProvider()
	if err != nil {
		log.Fatalf("Failed to create credentials provider: %v", err)
		return err
	}

	// 创建OSSClient实例。
	// yourEndpoint填写Bucket对应的Endpoint，以华东1（杭州）为例，填写为https://oss-cn-hangzhou.aliyuncs.com。其它Region请按实际情况填写。
	// yourRegion填写Bucket所在地域，以华东1（杭州）为例，填写为cn-hangzhou。其它Region请按实际情况填写。
	clientOptions := []oss.ClientOption{oss.SetCredentialsProvider(&provider)}
	clientOptions = append(clientOptions, oss.Region("cn-zhangjiakou"))
	// 设置签名版本
	clientOptions = append(clientOptions, oss.AuthVersion(oss.AuthV4))
	client, err := oss.New(ossConfig.Endpoint, "", "", clientOptions...)
	if err != nil {
		log.Fatalf("Failed to create OSS client: %v", err)
		return err
	}
	o.ossClient = client
	return nil
}

type WcdModel struct {
	RawHtml          string `json:"raw_html"`
	ParsedHtml       string `json:"parsed_html"`
	TextParserLabels string `json:"text_parser_labels"`
	Conclusion       string `json:"conclusion"`
}

func (o *OssOperator) getWcdZipReader(bucketName, objectKey string) (io.ReadCloser, error) {
	// 填写存储空间名称，例如examplebucket。
	//bucketName := "wcd-html-bucket-prod" // 请替换为实际的Bucket名称
	bucket, err := o.ossClient.Bucket(bucketName)
	if err != nil {
		hlog.CtxErrorf(o.ctx, "Failed to get bucket: %v", err)
		return nil, err
	}

	// 依次填写Object的完整路径（例如exampledir/exampleobject.txt）和本地文件的完整路径（例如D:\\localpath\\examplefile.txt）。
	//localFilePath := "./output.zip"                                                                           // 请替换为实际的本地文件路径
	//err = bucket.GetObjectToFile(objectKey, localFilePath)
	reader, err := bucket.GetObject(objectKey)
	if err != nil {
		hlog.CtxErrorf(o.ctx, "Failed to put object from file: %v", err)
		return nil, err
	}
	return reader, nil

}

func (o *OssOperator) DownloadWcdOssFile(bucketName, objectKey string) (*WcdModel, error) {
	reader, err := o.getWcdZipReader(bucketName, objectKey)
	if err != nil {
		hlog.CtxErrorf(o.ctx, "Failed to get wcd zip reader: %v", err)
		return nil, err
	}
	defer reader.Close()
	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		hlog.CtxErrorf(o.ctx, "Failed to read object into buffer: %v", err)
		return nil, err
	}

	// 解压缩 ZIP 文件
	r, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		hlog.CtxErrorf(o.ctx, "Failed to open zip file: %v", err)
		return nil, err
	}

	wcdModel := &WcdModel{}
	// 遍历 ZIP 文件中的每个文件
	for _, f := range r.File {
		//fmt.Printf("Extracting %s\n", f.Name)

		rc, err := f.Open()
		if err != nil {
			hlog.CtxErrorf(o.ctx, "Failed to open file %s: %v", f.Name, err)
			return nil, err
		}

		// 读取文件内容
		var fileBuffer bytes.Buffer
		if _, err := io.Copy(&fileBuffer, rc); err != nil {
			hlog.CtxErrorf(o.ctx, "Failed to read file %s: %v", f.Name, err)
			return nil, err
		}
		rc.Close()

		if strings.Contains(f.Name, "distill") {
			wcdModel.Conclusion = strings.TrimSpace(fileBuffer.String())
		} else if strings.Contains(f.Name, "readable") {
			wcdModel.ParsedHtml = strings.TrimSpace(fileBuffer.String())
		} else if strings.Contains(f.Name, "raw.html") {
			wcdModel.RawHtml = strings.TrimSpace(fileBuffer.String())
		} else if strings.Contains(f.Name, "model_result.json") {
			wcdModel.TextParserLabels = strings.TrimSpace(fileBuffer.String())
		}
	}
	return wcdModel, nil
}
