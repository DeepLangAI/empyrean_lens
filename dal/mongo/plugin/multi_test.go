package plugin

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"empyrean_lens/conf"
	"empyrean_lens/consts"
)

func TestMultiDao_FindMultiByQueryAndStatusAndTimeRange(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	t1, _ := time.Parse(consts.DateHourMinuteTemplate, "2023-07-01 00:00:00")
	t2, _ := time.Parse(consts.DateHourMinuteTemplate, "2024-10-30 00:00:00")
	data, err := NewMultiDao().FindMultiByQueryAndStatusAndTimeRange(ctx, "cuter", []int32{2, 3, 4}, t1, t2, 0, 10)
	assert.Nil(t, err)
	for _, x := range data {
		fmt.Printf("%+v\n", x)
	}
}
