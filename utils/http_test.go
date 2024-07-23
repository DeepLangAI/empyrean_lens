package utils

import (
	"context"
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
