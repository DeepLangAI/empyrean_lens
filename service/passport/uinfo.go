package passport

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/utils"
	"github.com/pkg/errors"
	"net/url"
	"strings"
	"sync"
)

type GetUidsReq struct {
	Phones string `json:"phones"`
}
type GetUidsResp struct {
	Code         int32             `thrift:"code,1" form:"code" json:"code" query:"code"`
	Msg          string            `thrift:"msg,2" form:"msg" json:"msg" query:"msg"`
	PhoneUIDMaps map[string]string `thrift:"phone_uid_maps,3" form:"phone_uid_maps" json:"phone_uid_maps" query:"phone_uid_maps"`
}

type GetPhoneReq struct {
	Uid string `json:"uid"`
}
type GetPhoneResp struct {
	Code  int32  `thrift:"code,1" form:"code" json:"code" query:"code"`
	Msg   string `thrift:"msg,2" form:"msg" json:"msg" query:"msg"`
	Phone string `thrift:"phone,3" form:"phone" json:"phone" query:"phone"`
}

func GetUserInfo(ctx context.Context, req empyrean_lens.UInfoReq) (*empyrean_lens.UInfoRespData, error) {
	if req.UID == "" && req.Phone == "" {
		return nil, errors.New("uid and phone both empty")
	}
	data := &empyrean_lens.UInfoRespData{
		Phone:    req.Phone,
		UID:      req.UID,
		Nickname: "",
	}
	if req.UID == "" {
		phoneToUid, err := GetUidByPhone(ctx, []string{req.Phone})
		if err != nil {
			return nil, err
		}
		data.UID = phoneToUid[req.Phone]
	} else if req.Phone == "" {
		uidToPhone, err := GetPhoneByUid(ctx, []string{req.UID})
		if err != nil {
			return nil, err
		}
		data.Phone = uidToPhone[req.UID]
	}
	return data, nil
}

func GetUidByPhone(ctx context.Context, phones []string) (map[string]string, error) {
	req := GetUidsReq{Phones: strings.Join(phones, ",")}
	resp := GetUidsResp{}

	err := utils.DoPost(ctx, "http://39.99.129.223:18005/get_uids", nil, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.PhoneUIDMaps, nil
}

func GetPhoneByUid(ctx context.Context, uids []string) (map[string]string, error) {
	result := map[string]string{}
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	for _, uid := range uids {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()
			resp := GetPhoneResp{}
			params := url.Values{}
			params.Set("uid", uid)
			err := utils.DoGet(ctx, "http://39.99.129.223:18005/get_phone", params, nil, &resp)
			if err != nil {
				return
			}
			mu.Lock()
			result[uid] = resp.Phone
			mu.Unlock()
		}(uid)
	}
	wg.Wait()
	return result, nil
}
