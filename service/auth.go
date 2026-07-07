package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	auth_model "empyrean_lens/biz/model/empyrean_lens/auth"
	"empyrean_lens/conf"
	dal_redis "empyrean_lens/dal/redis"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol"
	hconsts "github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/jwt"
)

const (
	cookieOAuthState    = "_oauth_state"
	cookieOAuthVerifier = "_oauth_verifier"
	cookieOAuthClientID = "_oauth_client_id"
	cookieOAuthRedirURI = "_oauth_redir_uri"
	cookieOAuthClientSt = "_oauth_client_st"
	CookieAuth          = "auth_token"
	IdentityKey         = "user" // JWT middleware IdentityKey，c.Get(IdentityKey) 读取当前用户

	codeKeyPrefix = "sso:code:"
	codeTTL       = 60 * time.Second
	tokenTTL      = 7 * 24 * time.Hour // JWT 有效期，auth cookie 和 Token 端点共用

	// OAuth2 错误码（RFC 6749），错误拼写会导致客户端无法正确处理
	errInvalidClient = "invalid_client"
	errServerError   = "server_error"
)

// AuthUser 用户信息，存入 JWT payload
type AuthUser struct {
	OpenID    string `json:"open_id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// ── JWT Middleware 单例 ────────────────────────────────────────────────────

var (
	authMwOnce sync.Once
	authMw     *jwt.HertzJWTMiddleware
)

// AuthMiddleware 返回全局 JWT middleware 单例。
// 首次调用时从 conf 读取密钥完成初始化，之后复用同一实例。
func AuthMiddleware() *jwt.HertzJWTMiddleware {
	authMwOnce.Do(func() {
		cfg := conf.GetConfig().Feishu
		svc := &AuthService{}
		var err error
		authMw, err = jwt.New(&jwt.HertzJWTMiddleware{
			Realm:       "empyrean-lens",
			Key:         []byte(cfg.JWTSecret),
			Timeout:     tokenTTL,
			IdentityKey: IdentityKey,

			// 同时支持 Cookie（浏览器同域）和 Authorization header（跨域服务）
			TokenLookup: "cookie: " + CookieAuth + ", header: Authorization",

			SendCookie:     true,
			CookieName:     CookieAuth,
			CookieHTTPOnly: true,
			CookieMaxAge:   tokenTTL,
			CookieDomain:   cfg.CookieDomain, // 生产填 ".lingowhale.com" 实现跨子域共享
			SecureCookie:   false,            // 生产环境改为 true（HTTPS）

			Authenticator: svc.FeishuAuthenticator,

			PayloadFunc: func(data interface{}) jwt.MapClaims {
				u, ok := data.(*AuthUser)
				if !ok {
					return jwt.MapClaims{}
				}
				return jwt.MapClaims{
					"open_id":    u.OpenID,
					"name":       u.Name,
					"avatar_url": u.AvatarURL,
				}
			},

			IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
				claims := jwt.ExtractClaims(ctx, c)
				return &AuthUser{
					OpenID:    stringClaim(claims, "open_id"),
					Name:      stringClaim(claims, "name"),
					AvatarURL: stringClaim(claims, "avatar_url"),
				}
			},

			LoginResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
				clientID, _ := url.QueryUnescape(string(c.Cookie(cookieOAuthClientID)))
				redirURI, _ := url.QueryUnescape(string(c.Cookie(cookieOAuthRedirURI)))
				clientSt, _ := url.QueryUnescape(string(c.Cookie(cookieOAuthClientSt)))
				clearTempCookies(c)

				if clientID == "" || redirURI == "" {
					// 直接访问 SSO（不带 client_id），登录后回首页
					c.Redirect(hconsts.StatusFound, []byte("/"))
					return
				}

				authCode, _ := randomHex(16)
				if err := storeCode(ctx, authCode, token, clientID); err != nil {
					hlog.CtxErrorf(ctx, "[auth] store code failed: %v", err)
					c.JSON(hconsts.StatusInternalServerError, map[string]string{"error": errServerError})
					return
				}

				hlog.CtxInfof(ctx, "[auth] code issued: client=%s redirect=%s", clientID, redirURI)
				target, _ := url.Parse(redirURI)
				q := target.Query()
				q.Set("code", authCode)
				if clientSt != "" {
					q.Set("state", clientSt)
				}
				target.RawQuery = q.Encode()
				c.Redirect(hconsts.StatusFound, []byte(target.String()))
			},

			Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
				hlog.CtxErrorf(ctx, "[auth] unauthorized: code=%d msg=%s path=%s", code, message, c.Path())
				accept := string(c.GetHeader("Accept"))
				if strings.Contains(accept, "text/html") {
					c.Redirect(hconsts.StatusFound, []byte("/auth/login"))
					return
				}
				c.JSON(code, &auth_model.BaseResp{Code: int32(code), Msg: message})
			},
		})
		if err != nil {
			panic("auth middleware init failed: " + err.Error())
		}
	})
	return authMw
}

// ── AuthService ────────────────────────────────────────────────────────────

type AuthService struct{}

func NewAuthService() *AuthService { return &AuthService{} }

// Login 发起飞书 OAuth 授权，支持两种模式：
//   - client_id 为空：第一方 web 登录，飞书授权后直接写 JWT cookie 并跳转 /（适合自己的前端）
//   - client_id 非空：标准 OAuth2 code exchange，飞书授权后带 code 跳回 redirect_uri（适合外部服务接入）
func (s *AuthService) Login(ctx context.Context, c *app.RequestContext, req *auth_model.LoginReq) {
	if req.ClientID != "" {
		// 标准 OAuth2 流程：校验已注册的 client
		client := getClient(req.ClientID)
		if client == nil || !validateRedirectURI(client, req.RedirectURI) {
			hlog.CtxWarnf(ctx, "[auth] invalid client or redirect_uri: client=%s uri=%s", req.ClientID, req.RedirectURI)
			c.JSON(hconsts.StatusBadRequest, map[string]string{"error": errInvalidClient})
			return
		}
		setTempCookie(c, cookieOAuthClientID, req.ClientID, 5*60)
		setTempCookie(c, cookieOAuthRedirURI, req.RedirectURI, 5*60)
		if req.State != "" {
			setTempCookie(c, cookieOAuthClientSt, req.State, 5*60)
		}
		hlog.CtxInfof(ctx, "[auth] oauth2 login: client=%s redirect=%s", req.ClientID, req.RedirectURI)
	} else {
		// 第一方 web 登录：无需 client_id，LoginResponse 直接写 JWT cookie 并跳转 /
		hlog.CtxInfof(ctx, "[auth] first-party web login")
	}
	s.startOAuth(ctx, c)
}

// Me 返回当前登录用户信息（需先经过 AuthMiddleware().MiddlewareFunc() 保护）。
func (s *AuthService) Me(_ context.Context, c *app.RequestContext) {
	user, _ := c.Get(IdentityKey)
	u, _ := user.(*AuthUser)
	resp := &auth_model.MeResp{Code: 0}
	if u != nil {
		resp.Data = &auth_model.UserInfo{
			OpenID:    u.OpenID,
			Name:      u.Name,
			AvatarURL: u.AvatarURL,
		}
	}
	c.JSON(hconsts.StatusOK, resp)
}

// startOAuth 生成 PKCE state/verifier，写入临时 cookie，重定向至飞书授权页。
func (s *AuthService) startOAuth(ctx context.Context, c *app.RequestContext) {
	state, err := randomHex(16)
	if err != nil {
		c.JSON(hconsts.StatusInternalServerError, &auth_model.BaseResp{Code: 500, Msg: "internal error"})
		return
	}
	verifier, challenge, err := newPKCE()
	if err != nil {
		c.JSON(hconsts.StatusInternalServerError, &auth_model.BaseResp{Code: 500, Msg: "internal error"})
		return
	}

	setTempCookie(c, cookieOAuthState, state, 5*60)
	setTempCookie(c, cookieOAuthVerifier, verifier, 5*60)

	cfg := conf.GetConfig().Feishu
	authURL := fmt.Sprintf(
		"https://accounts.feishu.cn/open-apis/authen/v1/authorize?"+
			"client_id=%s&response_type=code&redirect_uri=%s&state=%s"+
			"&code_challenge=%s&code_challenge_method=S256",
		url.QueryEscape(cfg.AppID),
		url.QueryEscape(cfg.RedirectURL),
		url.QueryEscape(state),
		url.QueryEscape(challenge),
	)
	c.Redirect(hconsts.StatusFound, []byte(authURL))
}

// FeishuAuthenticator 是 jwt.HertzJWTMiddleware.Authenticator，由 LoginHandler 在 /auth/callback 调用。
// 校验 state 防 CSRF，用 code 换取 access_token，获取用户信息。
func (s *AuthService) FeishuAuthenticator(ctx context.Context, c *app.RequestContext) (interface{}, error) {
	stateParam := string(c.Query("state"))
	stateCookie := string(c.Cookie(cookieOAuthState))
	if stateParam == "" || stateParam != stateCookie {
		hlog.CtxWarnf(ctx, "[auth] state mismatch: param=%s cookie=%s", stateParam, stateCookie)
		return nil, fmt.Errorf("invalid state")
	}

	code := string(c.Query("code"))
	if code == "" {
		hlog.CtxWarnf(ctx, "[auth] missing code, error=%s", string(c.Query("error_description")))
		return nil, fmt.Errorf("missing code")
	}

	verifier := string(c.Cookie(cookieOAuthVerifier))

	accessToken, err := s.exchangeCode(ctx, code, verifier)
	if err != nil {
		hlog.CtxErrorf(ctx, "[auth] exchange code failed: %v", err)
		return nil, fmt.Errorf("exchange code failed")
	}

	user, err := s.fetchUserInfo(ctx, accessToken)
	if err != nil {
		hlog.CtxErrorf(ctx, "[auth] fetch user info failed: %v", err)
		return nil, fmt.Errorf("fetch user info failed")
	}

	hlog.CtxInfof(ctx, "[auth] login ok: open_id=%s name=%s", user.OpenID, user.Name)
	return user, nil
}

// exchangeCode 向飞书 OAuth token 端点换取 user_access_token。
func (s *AuthService) exchangeCode(ctx context.Context, code, verifier string) (string, error) {
	cfg := conf.GetConfig().Feishu
	body, _ := sonic.Marshal(map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     cfg.AppID,
		"client_secret": cfg.AppSecret,
		"code":          code,
		"redirect_uri":  cfg.RedirectURL,
		"code_verifier": verifier,
	})

	var result struct {
		Code        int    `json:"code"`
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrDesc     string `json:"error_description"`
	}
	_, err := httplib.Do(ctx,
		"https://open.feishu.cn/open-apis/authen/v2/oauth/token",
		map[string]string{hconsts.HeaderContentType: hconsts.MIMEApplicationJSON},
		body, &result)
	if err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("feishu error %d: %s %s", result.Code, result.Error, result.ErrDesc)
	}
	return result.AccessToken, nil
}

// fetchUserInfo 用 user_access_token 调飞书获取用户信息（GET 接口）。
func (s *AuthService) fetchUserInfo(ctx context.Context, accessToken string) (*AuthUser, error) {
	var result struct {
		Code int `json:"code"`
		Data struct {
			OpenID    string `json:"open_id"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"data"`
	}
	_, err := httplib.Do(ctx,
		"https://open.feishu.cn/open-apis/authen/v1/user_info",
		map[string]string{hconsts.HeaderAuthorization: "Bearer " + accessToken},
		nil, &result,
		httplib.WithHttpMethod("GET"),
	)
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("feishu user_info error code=%d", result.Code)
	}
	return &AuthUser{
		OpenID:    result.Data.OpenID,
		Name:      result.Data.Name,
		AvatarURL: result.Data.AvatarURL,
	}, nil
}

func newPKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func setTempCookie(c *app.RequestContext, name, value string, maxAge int) {
	c.SetCookie(name, value, maxAge, "/", "", protocol.CookieSameSiteLaxMode, false, true)
}

func clearTempCookies(c *app.RequestContext) {
	setTempCookie(c, cookieOAuthState, "", -1)
	setTempCookie(c, cookieOAuthVerifier, "", -1)
	setTempCookie(c, cookieOAuthClientID, "", -1)
	setTempCookie(c, cookieOAuthRedirURI, "", -1)
	setTempCookie(c, cookieOAuthClientSt, "", -1)
}

// ── OAuth2 client helpers ─────────────────────────────────────────────────

func getClient(clientID string) *conf.OAuthClient {
	clients := conf.GetConfig().Feishu.OAuthClients
	for i := range clients {
		if clients[i].ClientID == clientID {
			return &clients[i]
		}
	}
	return nil
}

func validateRedirectURI(client *conf.OAuthClient, uri string) bool {
	for _, allowed := range client.RedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// ── Authorization code（Redis）────────────────────────────────────────────

type codePayload struct {
	Token    string `json:"token"`
	ClientID string `json:"client_id"`
}

func storeCode(ctx context.Context, code, token, clientID string) error {
	payload, _ := sonic.MarshalString(codePayload{Token: token, ClientID: clientID})
	return dal_redis.KeySet(ctx, codeKeyPrefix+code, payload, codeTTL)
}

func consumeCode(ctx context.Context, code string) (token, clientID string, ok bool) {
	cmd := dal_redis.GetVal(ctx, codeKeyPrefix+code)
	val := cmd.Val()
	if val == "" {
		return
	}
	_ = dal_redis.DelKey(ctx, codeKeyPrefix+code)
	var p codePayload
	if err := sonic.UnmarshalString(val, &p); err != nil || p.Token == "" {
		return
	}
	return p.Token, p.ClientID, true
}

func stringClaim(claims jwt.MapClaims, key string) string {
	v, _ := claims[key].(string)
	return v
}
