package passport

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
)

// Auth 鉴权，有权限返回true，没有权限返回false
func Auth(ctx context.Context, req *empyrean_lens.AuthReq) (bool, string, *consts.BizCode) {
	userAccessToken, err := utils.GetUserAccessToken(ctx, conf.GetLark().AppId, conf.GetLark().AppSecret, req.Code)
	if err != nil {
		return false, "", &consts.BizCode{Code: consts.LarkAuthError.Code, Msg: consts.LarkAuthError.Msg}
	}
	userInfo, err := utils.GetUserInfo(ctx, conf.GetLark().AppId, conf.GetLark().AppSecret, userAccessToken)
	if err != nil {
		return false, "", &consts.BizCode{Code: consts.LarkAuthError.Code, Msg: consts.LarkAuthError.Msg}
	}

	if utils.InSlice(*userInfo.Name, conf.GetLark().AuthNames) {
		token, _ := utils.GenerateJWT(map[string]any{
			consts.LARK_USERNAME: *userInfo.Name,
		}, consts.LARK_AUTH_EXPIRE_TIME, conf.GetLark().JwtSecret)
		return true, token, nil
	}
	return false, "", nil
}
