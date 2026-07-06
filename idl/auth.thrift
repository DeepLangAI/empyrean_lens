namespace go empyrean_lens.auth

// LoginReq 登录发起请求，redirect 可选，跨域接入时填写目标回调地址
struct LoginReq {
    1: string redirect (api.query="redirect")
}

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

// VerifyResp 供其他服务调用 /api/auth/verify 时使用
struct VerifyResp {
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
    // redirect 可选：跨域服务填登录成功后的回调地址，认证中心会带 token 跳回
    BaseResp Login(1: LoginReq req)(
        api.get="/auth/login"
    )

    // 飞书 OAuth 回调（用 code 换 token，写 cookie，跳转首页）
    BaseResp Callback(1: LoginCallbackReq req)(
        api.get="/auth/callback"
    )

    // 返回当前登录用户信息（需携带 auth_token cookie）
    MeResp Me()(
        api.get="/api/auth/me"
    )

    // 登出（清除 cookie）
    BaseResp Logout()(
        api.post="/api/auth/logout"
    )

    // Token 验证接口：供其他服务通过 Authorization: Bearer <token> 验证身份
    // 200 → 返回用户信息；401 → token 无效或已过期
    VerifyResp Verify()(
        api.get="/api/auth/verify"
    )
}
