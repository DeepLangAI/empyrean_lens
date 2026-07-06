package http

import (
	"context"
	"empyrean_lens/conf"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSlsQuery(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	startTime := time.Now().Add(-time.Hour * 24)
	endTime := time.Now()
	logsResp, err := SlsQuery(ctx, LogStore_BusinessPod, "*", startTime, endTime, 10)
	assert.Nil(t, err)
	assert.True(t, logsResp.Code == 0)
	assert.True(t, len(logsResp.Logs) > 0)
}
