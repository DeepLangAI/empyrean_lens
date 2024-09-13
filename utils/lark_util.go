package utils

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	"github.com/larksuite/oapi-sdk-go/v3/service/authen/v1"
)

// SDK 使用文档：https://github.com/larksuite/oapi-sdk-go/tree/v3_main
// 复制该 Demo 后, 需要将 "YOUR_APP_ID", "YOUR_APP_SECRET" 替换为自己应用的 APP_ID, APP_SECRET.
// 以下示例代码是根据 API 调试台参数自动生成，如果存在代码问题，请在 API 调试台填上相关必要参数后再使用
func GetAccessToken(ctx context.Context, appId, appSecret string) (string, error) {
	// 创建 Client
	client := lark.NewClient(appId, appSecret)
	// 创建请求对象
	req := larkauth.NewInternalAppAccessTokenReqBuilder().
		Body(larkauth.NewInternalAppAccessTokenReqBodyBuilder().
			AppId(appId).
			AppSecret(appSecret).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Auth.AppAccessToken.Internal(ctx, req)

	// 处理错误
	if err != nil {
		hlog.CtxErrorf(ctx, "get lark access token request failed, err: %+v", err)
		return "", err
	}

	// 服务端错误处理
	if !resp.Success() {
		errStr := fmt.Sprintf("get lark access token request failed, code: %d, msg: %s, log_id: %s", resp.Code, resp.Msg, resp.LogId())
		hlog.CtxErrorf(ctx, errStr)
		return "", errors.New(errStr)
	}

	// 业务处理
	var data map[string]any
	JSONUnMarshal(resp.RawBody, &data)
	return data["app_access_token"].(string), nil
}

func GetUserAccessToken(ctx context.Context, appId, appSecret, code string) (string, error) {
	// 创建 Client
	client := lark.NewClient(appId, appSecret)
	// 创建请求对象
	req := larkauthen.NewCreateOidcAccessTokenReqBuilder().
		Body(larkauthen.NewCreateOidcAccessTokenReqBodyBuilder().
			GrantType(`authorization_code`).
			Code(code).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Authen.OidcAccessToken.Create(ctx, req)

	// 处理错误
	if err != nil {
		hlog.CtxErrorf(ctx, "get lark user access token failed, err: %+v", err)
		return "", err
	}

	// 服务端错误处理
	if !resp.Success() {
		errStr := fmt.Sprintf("get lark user access token failed, code: %d, msg: %s, request_id: %s",
			resp.Code, resp.Msg, resp.RequestId())
		hlog.CtxErrorf(ctx, errStr)
		return "", errors.New(errStr)
	}

	return *resp.Data.AccessToken, nil
}

func GetUserInfo(ctx context.Context, appId, appSecret, userAccessToken string) (*larkauthen.GetUserInfoRespData, error) {
	// 创建 Client
	client := lark.NewClient(appId, appSecret)

	// 发起请求
	resp, err := client.Authen.UserInfo.Get(ctx, larkcore.WithUserAccessToken(userAccessToken))

	// 处理错误
	if err != nil {
		hlog.Errorf("get lark user info failed, err: %+v", err)
		return nil, err
	}

	// 服务端错误处理
	if !resp.Success() {
		errStr := fmt.Sprintf("get lark user info failed, code: %d, msg: %s, request_id: %s", resp.Code, resp.Msg, resp.RequestId())
		return nil, errors.New(errStr)
	}

	// 业务处理
	return resp.Data, nil
}
