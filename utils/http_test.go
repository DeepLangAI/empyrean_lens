package utils

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestDoGet(t *testing.T) {
	ctx := context.Background()
	uri := "https://baidu.com"
	params := url.Values{
		"sign": {"abc"},
	}
	DoGet(ctx, uri, params, nil, nil)
}

func Test_ipInRange(t *testing.T) {
	assert.True(t, ipInRange("172.16.0.1", "172.16.0.0/12"))
	assert.True(t, ipInRange("172.31.255.254", "172.16.0.0/12"))
}

func GetWcdZipReader() (io.ReadCloser, error) {
	endpoint := "oss-cn-zhangjiakou.aliyuncs.com"
	OSS_ACCESS_KEY_ID := "REDACTED"
	OSS_ACCESS_KEY_SECRET := "REDACTED"

	os.Setenv("OSS_ACCESS_KEY_ID", OSS_ACCESS_KEY_ID)
	os.Setenv("OSS_ACCESS_KEY_SECRET", OSS_ACCESS_KEY_SECRET)
	os.Setenv("OSS_SESSION_TOKEN", "")

	provider, err := oss.NewEnvironmentVariableCredentialsProvider()
	if err != nil {
		log.Fatalf("Failed to create credentials provider: %v", err)
	}

	// 创建OSSClient实例。
	// yourEndpoint填写Bucket对应的Endpoint，以华东1（杭州）为例，填写为https://oss-cn-hangzhou.aliyuncs.com。其它Region请按实际情况填写。
	// yourRegion填写Bucket所在地域，以华东1（杭州）为例，填写为cn-hangzhou。其它Region请按实际情况填写。
	clientOptions := []oss.ClientOption{oss.SetCredentialsProvider(&provider)}
	clientOptions = append(clientOptions, oss.Region("cn-zhangjiakou"))
	// 设置签名版本
	clientOptions = append(clientOptions, oss.AuthVersion(oss.AuthV4))
	client, err := oss.New(endpoint, "", "", clientOptions...)
	if err != nil {
		log.Fatalf("Failed to create OSS client: %v", err)
	}

	// 填写存储空间名称，例如examplebucket。
	bucketName := "wcd-html-bucket-prod" // 请替换为实际的Bucket名称
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		log.Fatalf("Failed to get bucket: %v", err)
	}

	// 依次填写Object的完整路径（例如exampledir/exampleobject.txt）和本地文件的完整路径（例如D:\\localpath\\examplefile.txt）。
	objectKey := "parsed/lingos-wmMMZBEQAA2Y7XivYlXU8P20fCeb7q6A/20241101143102_6724759eac5b464db956f32b.zip" // 请替换为实际的对象Key
	//localFilePath := "./output.zip"                                                                           // 请替换为实际的本地文件路径
	//err = bucket.GetObjectToFile(objectKey, localFilePath)
	reader, err := bucket.GetObject(objectKey)
	if err != nil {
		log.Fatalf("Failed to put object from file: %v", err)
	}

	return reader, nil
}

type WcdModel struct {
	RawHtml          string `json:"raw_html"`
	ParsedHtml       string `json:"parsed_html"`
	TextParserLabels string `json:"text_parser_labels"`
	Conclusion       string `json:"conclusion"`
}

func TestDownloadOssFile(t *testing.T) {
	reader, err := GetWcdZipReader()
	assert.Nil(t, err)
	defer reader.Close()
	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		log.Fatalf("Failed to read object into buffer: %v", err)
	}

	// 解压缩 ZIP 文件
	r, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		log.Fatalf("Failed to create zip reader: %v", err)
	}

	wcdModel := WcdModel{}
	// 遍历 ZIP 文件中的每个文件
	for _, f := range r.File {
		fmt.Printf("Extracting %s\n", f.Name)

		rc, err := f.Open()
		if err != nil {
			log.Fatalf("Failed to open file %s: %v", f.Name, err)
		}

		// 读取文件内容
		var fileBuffer bytes.Buffer
		if _, err := io.Copy(&fileBuffer, rc); err != nil {
			log.Fatalf("Failed to read file %s: %v", f.Name, err)
		}
		rc.Close()

		// 这里可以对 fileBuffer 做进一步处理
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
	marshalString, err := sonic.MarshalString(wcdModel)
	assert.Nil(t, err)
	fmt.Println(marshalString)
}
