package utils

import (
	"context"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/url"
	"os"
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
