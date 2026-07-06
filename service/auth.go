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

	"empyrean_lens/conf"

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
	cookieOAuthRedirect = "_oauth_redirect"
	CookieAuth          = "auth_token"
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
			Timeout:     7 * 24 * time.Hour,
			IdentityKey: "user",

			// 同时支持 Cookie（浏览器同域）和 Authorization header（跨域服务）
			TokenLookup: "cookie: " + CookieAuth + ", header: Authorization",

			SendCookie:     true,
			CookieName:     CookieAuth,
			CookieHTTPOnly: true,
			CookieMaxAge:   7 * 24 * time.Hour,
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
				redirectURL := string(c.Cookie(cookieOAuthRedirect))
				hlog.CtxInfof(ctx, "[auth] login success, redirect_cookie=%q allowed=%v", redirectURL, isAllowedRedirect(redirectURL))
				clearTempCookies(c)
				if redirectURL != "" && isAllowedRedirect(redirectURL) {
					// 跨域登录：带 token 跳回接入方服务
					hlog.CtxInfof(ctx, "[auth] cross-domain redirect to %s", redirectURL)
					target, _ := url.Parse(redirectURL)
					q := target.Query()
					q.Set("token", token)
					target.RawQuery = q.Encode()
					c.Redirect(hconsts.StatusFound, []byte(target.String()))
					return
				}
				c.Redirect(hconsts.StatusFound, []byte("/"))
			},

			Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
				hlog.CtxErrorf(ctx, "[auth] unauthorized: code=%d msg=%s path=%s", code, message, c.Path())
				accept := string(c.GetHeader("Accept"))
				if strings.Contains(accept, "text/html") {
					c.Redirect(hconsts.StatusFound, []byte("/auth/login"))
					return
				}
				c.JSON(code, map[string]interface{}{"code": code, "msg": message})
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

// Login 发起飞书 OAuth，生成 state/verifier 写 cookie，跳转飞书授权页。
// 支持 redirect 参数：跨域接入方传入登录成功后的回调地址（需在白名单中）。
func (s *AuthService) Login(ctx context.Context, c *app.RequestContext) {
	redirect := string(c.Query("redirect"))
	if redirect != "" {
		if isAllowedRedirect(redirect) {
			hlog.CtxInfof(ctx, "[auth] cross-domain login, redirect=%s", redirect)
			setTempCookie(c, cookieOAuthRedirect, redirect, 5*60)
		} else {
			hlog.CtxWarnf(ctx, "[auth] redirect not allowed: %s", redirect)
		}
	}
	s.startOAuth(ctx, c)
}

// Me 返回当前登录用户信息（需先经过 AuthMiddleware().MiddlewareFunc() 保护）。
func (s *AuthService) Me(_ context.Context, c *app.RequestContext) {
	user, _ := c.Get("user")
	c.JSON(hconsts.StatusOK, map[string]interface{}{"code": 0, "data": user})
}

// Verify 供其他服务调用，通过 Authorization: Bearer <token> 验证身份并返回用户信息。
// 路由已被 _authMw() 保护，到达此处时 token 已验证通过，直接读 context 即可。
func (s *AuthService) Verify(_ context.Context, c *app.RequestContext) {
	user, _ := c.Get("user")
	c.JSON(hconsts.StatusOK, map[string]interface{}{"code": 0, "data": user})
}

// startOAuth 生成 PKCE state/verifier，写入临时 cookie，重定向至飞书授权页。
func (s *AuthService) startOAuth(ctx context.Context, c *app.RequestContext) {
	state, err := randomHex(16)
	if err != nil {
		c.JSON(hconsts.StatusInternalServerError, map[string]string{"msg": "internal error"})
		return
	}
	verifier, challenge, err := newPKCE()
	if err != nil {
		c.JSON(hconsts.StatusInternalServerError, map[string]string{"msg": "internal error"})
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

// ── helpers ────────────────────────────────────────────────────────────────

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
	setTempCookie(c, cookieOAuthRedirect, "", -1)
}

// isAllowedRedirect 校验 redirect URL 是否在配置白名单中，防止 token 被骗走。
// 白名单项可以是精确 host（如 "localhost:8888"）或父域名（如 "lingowhale.com"）。
// 父域名匹配会同时允许该域名本身及其所有子域名。
func isAllowedRedirect(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Host
	for _, allowed := range conf.GetConfig().Feishu.AllowedRedirects {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

func stringClaim(claims jwt.MapClaims, key string) string {
	v, _ := claims[key].(string)
	return v
}
