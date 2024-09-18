package passport

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/utils"

	"github.com/golang-jwt/jwt/v5"
	larkauthen "github.com/larksuite/oapi-sdk-go/v3/service/authen/v1"
)

// Auth 鉴权，有权限返回true，没有权限返回false
func Auth(ctx context.Context, req *empyrean_lens.AuthReq) (bool, string, *consts.BizCode) {
	userAccessToken, err := utils.GetUserAccessToken(ctx, conf.GetLark().AppId, conf.GetLark().AppSecret, req.Code)
	if err != nil {
		return false, "", &consts.BizCode{Code: consts.LarkAuthError.Code, Msg: consts.LarkAuthError.Msg}
	}
	userInfo, err := utils.GetUserInfo(ctx, conf.GetLark().AppId, conf.GetLark().AppSecret, userAccessToken)
	if err != nil || userInfo == nil {
		return false, "", &consts.BizCode{Code: consts.LarkAuthError.Code, Msg: consts.LarkAuthError.Msg}
	}

	if CheckUserInfo(ctx, userInfo) {
		token, _ := utils.GenerateJWT(GetJWTData(ctx, userInfo),
			consts.LARK_AUTH_EXPIRE_TIME, conf.GetLark().JwtSecret)
		return true, token, nil
	}
	return false, "", nil
}

func CheckUserInfo(ctx context.Context, userInfo *larkauthen.GetUserInfoRespData) bool {
	auth := conf.GetLark().Auth
	return (userInfo.Name != nil && utils.InSlice(*userInfo.Name, auth.Names)) ||
		(userInfo.Email != nil && utils.InSlice(*userInfo.Email, auth.Emails)) ||
		(userInfo.Mobile != nil && utils.InSlice(*userInfo.Mobile, auth.Mobiles)) ||
		(userInfo.EmployeeNo != nil && utils.InSlice(*userInfo.EmployeeNo, auth.EmployeeNos))
}

func CheckCookie(ctx context.Context, claim jwt.MapClaims) bool {
	resp := false
	auth := conf.GetLark().Auth
	if username, ok := claim[consts.LARK_USERNAME].(string); ok {
		resp = resp || utils.InSlice(username, auth.Names)
	}
	if email, ok := claim[consts.LARK_EMAIL].(string); ok {
		resp = resp || utils.InSlice(email, auth.Emails)
	}
	if mobile, ok := claim[consts.LARK_MOBILE].(string); ok {
		resp = resp || utils.InSlice(mobile, auth.Mobiles)
	}
	if employeeNo, ok := claim[consts.LARK_EMPLOYEE_NO].(string); ok {
		resp = resp || utils.InSlice(employeeNo, auth.EmployeeNos)
	}
	return resp
}

func GetJWTData(ctx context.Context, userInfo *larkauthen.GetUserInfoRespData) map[string]any {
	resp := map[string]any{}
	if userInfo.Name != nil {
		resp[consts.LARK_USERNAME] = *userInfo.Name
	}
	if userInfo.Email != nil {
		resp[consts.LARK_EMAIL] = *userInfo.Email
	}
	if userInfo.Mobile != nil {
		resp[consts.LARK_MOBILE] = *userInfo.Mobile
	}
	if userInfo.EmployeeNo != nil {
		resp[consts.LARK_EMPLOYEE_NO] = *userInfo.EmployeeNo
	}
	return resp
}
