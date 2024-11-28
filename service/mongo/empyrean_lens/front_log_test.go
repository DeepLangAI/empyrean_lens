package empyrean_lens

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_userError_GeneralErrorList(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	s := UserErrorService
	req := empyrean_lens.UserErrorListReq{}
	list, err := s.GeneralErrorList(ctx, req)
	assert.Nil(t, err)
	for _, v := range list {
		fmt.Println(v)
	}
}

func Test_userError_UploadErrorList(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	s := UserErrorService
	var req empyrean_lens.UserErrorUploadReq
	req.Date = "2024-11-17"
	list, err := s.UploadErrorList(ctx, req)
	assert.Nil(t, err)
	for _, v := range list {
		fmt.Println(v)
	}
}

func Test_userError_GenerateErrorList(t *testing.T) {

}
