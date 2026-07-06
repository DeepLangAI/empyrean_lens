namespace go empyrean_lens.auth

struct LoginCallbackReq {
    1: string code  (api.query="code")
    2: string state (api.query="state")
}

struct UserInfo {
    1: string open_id
    2: string name
    3: string avatar_url
}

struct MeResp {
    1: i32 code
    2: string msg
    3: UserInfo data
}

struct BaseResp {
    1: i32 code
    2: string msg
}

service AuthService {
    // 发起飞书 OAuth 登录（重定向到飞书授权页）
    BaseResp Login()(
        api.get="/auth/login"
    )

    // 飞书 OAuth 回调（用 code 换 token，写 cookie，跳转首页）
    BaseResp Callback(1: LoginCallbackReq req)(
        api.get="/auth/callback"
    )

    // 返回当前登录用户信息
    MeResp Me()(
        api.get="/api/auth/me"
    )

    // 登出（清除 cookie）
    BaseResp Logout()(
        api.post="/api/auth/logout"
    )
}
