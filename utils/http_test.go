package utils

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/url"
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
