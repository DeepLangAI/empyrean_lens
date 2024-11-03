package tools

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOssOperator_DownloadOssFile(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	op := GetOssOperator(ctx)
	//objectKey := "parsed/lingos-wmMMZBEQAA2Y7XivYlXU8P20fCeb7q6A/20241101143102_6724759eac5b464db956f32b.zip" // 请替换为实际的对象Key
	//bucketName := "wcd-html-bucket-prod"

	objectKey := "parsed/123/20241102190121_123.zip" // 请替换为实际的对象Key
	bucketName := "wcd-html-bucket-test"
	file, err := op.DownloadWcdOssFile(bucketName, objectKey)
	assert.Nil(t, err)
	fmt.Println(file)
}
