package passport

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetUserActionTimeline(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	dal.Init()

	req := empyrean_lens.UserActionInfoReq{
		Uids: []string{"fa8b889879214fddac3985ba2c30df28"},
	}
	timeline, err := GetUserActionTimeline(ctx, req)
	assert.Nil(t, err)
	fmt.Println(timeline)
}
