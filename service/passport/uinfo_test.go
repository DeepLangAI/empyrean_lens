package passport

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"testing"
)

func TestGetUidByPhone(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	dal.Init()

	phoneToUid := map[string]string{
		"15801172522": "63e0713930c33a167f79d5d8",
		"18710797707": "e31b474f1ef540dbae86194e622ecec5",
	}
	phones := utils.KeysOfMap(phoneToUid)
	uids, err := GetUidByPhone(ctx, phones)
	if err != nil {
		t.Fatal(err)
	}
	for phone, uid := range uids {
		assert.True(t, phoneToUid[phone] == uid)
	}
}

func TestGetPhoneByUid(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	dal.Init()

	uidToPhone := map[string]string{
		"63e0713930c33a167f79d5d8":         "15801172522",
		"e31b474f1ef540dbae86194e622ecec5": "18710797707",
	}
	uids := utils.KeysOfMap(uidToPhone)
	phones, err := GetPhoneByUid(ctx, uids)
	if err != nil {
		t.Fatal(err)
	}
	for uid, phone := range phones {
		assert.True(t, uidToPhone[uid] == phone)
	}
}

func TestGetUserInfo(t *testing.T) {
	ctx := context.Background()
	conf.TestInit()
	dal.Init()

	t.Run("从手机号查用户ID", func(t *testing.T) {
		UID := "63e0713930c33a167f79d5d8"
		req := empyrean_lens.UInfoReq{
			Phone: "15801172522",
		}
		info, err := GetUserInfo(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, info.UID == UID)
	})
	t.Run("从用户ID查手机号", func(t *testing.T) {
		Phone := "15801172522"
		req := empyrean_lens.UInfoReq{
			UID: "63e0713930c33a167f79d5d8",
		}
		info, err := GetUserInfo(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, info.Phone == Phone)
	})
}
