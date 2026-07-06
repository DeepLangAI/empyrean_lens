namespace go empyrean_lens.auth

// ── 公共 struct ──────────────────────────────────────────────────────────────

struct BaseResp {
    1: i32 code
    2: string msg
}

struct UserInfo {
    1: string open_id
    2: string name
    3: string avatar_url
}

// ── AuthService struct ───────────────────────────────────────────────────────

// LoginReq 标准 OAuth2 授权请求
struct LoginReq {
    1: string client_id    (api.query="client_id")
    2: string redirect_uri (api.query="redirect_uri")
    3: string state        (api.query="state")   // 客户端生成的随机串，原样带回
}

struct LoginCallbackReq {
    1: string code  (api.query="code")
    2: string state (api.query="state")
}

struct MeResp {
    1: i32 code
    2: string msg
    3: UserInfo data
}

// ── OidcService struct ───────────────────────────────────────────────────────

// IntrospectReq RFC 7662 token introspection 请求
struct IntrospectReq {
    1: string token (api.form="token")
}

// IntrospectResp RFC 7662 标准响应
struct IntrospectResp {
    1: bool   active
    2: string sub
    3: string name
    4: string picture
    5: i64    exp
    6: i64    iat
}

// TokenReq RFC 6749 token 端点请求（Authorization Code Grant）
struct TokenReq {
    1: string grant_type    (api.form="grant_type")    // 固定 "authorization_code"
    2: string client_id     (api.form="client_id")
    3: string client_secret (api.form="client_secret")
    4: string code          (api.form="code")
    5: string redirect_uri  (api.form="redirect_uri")  // 须与授权时一致
}

// TokenResp RFC 6749 token 端点响应
struct TokenResp {
    1: string access_token
    2: string token_type    // "Bearer"
    3: i64    expires_in    // 秒
}

// ── Services ─────────────────────────────────────────────────────────────────

service AuthService {
    // 发起飞书 OAuth 登录（重定向到飞书授权页）
    // 需提供已注册的 client_id + redirect_uri，登录成功后带 code 跳回 redirect_uri
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
}

service OidcService {
    // OIDC Discovery 文档，接入方通过此端点自动发现 SSO 能力（RFC 8414）
    BaseResp Configuration()(
        api.get="/.well-known/openid-configuration"
    )

    // 标准 OIDC UserInfo 端点（OIDC Core §5.3）
    // 需携带 Authorization: Bearer <token>，返回标准字段名（sub/picture）
    BaseResp UserInfo()(
        api.get="/userinfo"
    )

    // 标准 Token Introspection 端点（RFC 7662）
    // token 在 POST body form 里，返回 {active, sub, exp, ...}
    IntrospectResp Introspect(1: IntrospectReq req)(
        api.post="/introspect"
    )

    // 标准 OAuth2 Token 端点（RFC 6749 §4.1.3）
    // 接入方用 authorization_code 换取 access_token，需提供 client_id + client_secret
    TokenResp Token(1: TokenReq req)(
        api.post="/token"
    )
}
