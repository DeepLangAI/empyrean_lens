package service

import (
	"context"
	"fmt"
	"time"

	auth_model "empyrean_lens/biz/model/empyrean_lens/auth"
	"empyrean_lens/conf"

	"github.com/cloudwego/hertz/pkg/app"
	hconsts "github.com/cloudwego/hertz/pkg/protocol/consts"
	jwtv4 "github.com/golang-jwt/jwt/v4"
	"github.com/hertz-contrib/jwt"
)

type OidcService struct{}

func NewOidcService() *OidcService { return &OidcService{} }

const (
	// OAuth2 / OIDC 错误码（RFC 6749 / RFC 7662），错误拼写会导致客户端无法正确处理
	errUnsupportedGrantType = "unsupported_grant_type"
	errInvalidGrant         = "invalid_grant"
	errInvalidTokenOIDC     = "invalid_token"

	tokenTypeBearer = "Bearer" // RFC 6749 §7.1
)

// Configuration 返回 OIDC Discovery 文档（RFC 8414）。
// 仅声明当前实际实现的能力，未实现的端点标为 null，避免误导接入方。
func (s *OidcService) Configuration(_ context.Context, c *app.RequestContext) {
	issuer := conf.GetConfig().Feishu.Issuer
	if issuer == "" {
		issuer = "https://" + string(c.Host())
	}
	c.JSON(hconsts.StatusOK, map[string]interface{}{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/auth/login",
		"token_endpoint":                        issuer + "/token",
		"userinfo_endpoint":                     issuer + "/userinfo",
		"introspection_endpoint":                issuer + "/introspect",
		"end_session_endpoint":                  issuer + "/api/auth/logout",
		"jwks_uri":                              nil, // 暂未实现（当前使用 HMAC 对称签名）
		"scopes_supported":                      []string{"openid"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"HS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post"},
	})
}

// UserInfo 标准 OIDC UserInfo 端点（OIDC Core §5.3）。
// 路由受 _userinfoMw() 保护，token 已验证，直接从 context 取用户信息返回标准字段名。
func (s *OidcService) UserInfo(_ context.Context, c *app.RequestContext) {
	user, _ := c.Get(IdentityKey)
	u, ok := user.(*AuthUser)
	if !ok {
		c.JSON(hconsts.StatusUnauthorized, map[string]string{"error": errInvalidTokenOIDC})
		return
	}
	c.JSON(hconsts.StatusOK, map[string]interface{}{
		"sub":     u.OpenID, // 标准字段：用户唯一标识
		"name":    u.Name,
		"picture": u.AvatarURL, // 标准字段：头像 URL
	})
}

// Introspect 标准 Token Introspection 端点（RFC 7662）。
// req.Token 由 handler 通过 BindAndValidate 绑定，手动验签，返回 {active, sub, exp, ...}。
func (s *OidcService) Introspect(_ context.Context, c *app.RequestContext, req *auth_model.IntrospectReq) {
	inactive := &auth_model.IntrospectResp{Active: false}

	if req.Token == "" {
		c.JSON(hconsts.StatusOK, inactive)
		return
	}

	cfg := conf.GetConfig().Feishu
	token, err := jwtv4.Parse(req.Token, func(t *jwtv4.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtv4.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(hconsts.StatusOK, inactive)
		return
	}

	claims, ok := token.Claims.(jwtv4.MapClaims)
	if !ok {
		c.JSON(hconsts.StatusOK, inactive)
		return
	}

	exp, _ := claims["exp"].(float64)
	iat, _ := claims["orig_iat"].(float64)
	c.JSON(hconsts.StatusOK, &auth_model.IntrospectResp{
		Active:  true,
		Sub:     stringClaim(jwt.MapClaims(claims), "open_id"),
		Name:    stringClaim(jwt.MapClaims(claims), "name"),
		Picture: stringClaim(jwt.MapClaims(claims), "avatar_url"),
		Exp:     int64(exp),
		Iat:     int64(iat),
	})
}

// Token 标准 OAuth2 Token 端点（RFC 6749 §4.1.3）。
// req 由 handler 通过 BindAndValidate 绑定，须携带 client_id + client_secret 验证身份。
func (s *OidcService) Token(ctx context.Context, c *app.RequestContext, req *auth_model.TokenReq) {
	if req.GrantType != "authorization_code" {
		c.JSON(hconsts.StatusBadRequest, map[string]string{"error": errUnsupportedGrantType})
		return
	}

	client := getClient(req.ClientID)
	if client == nil || client.ClientSecret != req.ClientSecret {
		c.JSON(hconsts.StatusUnauthorized, map[string]string{"error": errInvalidClient})
		return
	}

	token, codeClientID, ok := consumeCode(ctx, req.Code)
	if !ok || codeClientID != req.ClientID {
		c.JSON(hconsts.StatusBadRequest, map[string]string{"error": errInvalidGrant})
		return
	}

	c.JSON(hconsts.StatusOK, &auth_model.TokenResp{
		AccessToken: token,
		TokenType:   tokenTypeBearer,
		ExpiresIn:   int64(tokenTTL / time.Second),
	})
}
