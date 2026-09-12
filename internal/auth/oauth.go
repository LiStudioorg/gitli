package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/oauth2"

	"gitli/internal/config"
	"gitli/internal/db"
)

// OAuthConfig 当前生效的 OAuth2 配置；未启用时为 nil。
var oauthCfg *config.OAuth2Config

// SetOAuthConfig 由 main 在启动时注入。
func SetOAuthConfig(cfg *config.OAuth2Config) {
	oauthCfg = cfg
}

// OAuthEnabled 是否已启用 OAuth2。
func OAuthEnabled() bool {
	return oauthCfg != nil && oauthCfg.Enabled
}

// OAuthEnabledCheck 供 handler 判断（未配置时给出明确错误）。
func OAuthConfigOrError() (*config.OAuth2Config, error) {
	if !OAuthEnabled() {
		return nil, errors.New("oauth2 not configured: set [oauth2] enabled=true in config.toml or GITLI_OAUTH_* env")
	}
	return oauthCfg, nil
}

func oauthEndpoint(c *config.OAuth2Config) oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  c.AuthURL,
		TokenURL: c.TokenURL,
	}
}

// OAuthAuthCodeURL 生成 provider 授权跳转 URL（state 防 CSRF）。
func OAuthAuthCodeURL(c *config.OAuth2Config, state string) string {
	conf := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     oauthEndpoint(c),
		Scopes:       c.Scopes,
		RedirectURL:  "", // callback 由 provider 配置/相对地址推导，一般直接配完整 URL
	}
	return conf.AuthCodeURL(state)
}

// OAuthExchange 用 code 换 token。
func OAuthExchange(ctx context.Context, c *config.OAuth2Config, code string) (*oauth2.Token, error) {
	conf := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     oauthEndpoint(c),
	}
	return conf.Exchange(ctx, code)
}

// OAuthUserinfo 请求 userinfo 端点，提取 sub / preferred_username / email。
func OAuthUserinfo(ctx context.Context, c *config.OAuth2Config, token *oauth2.Token) (sub, username, email string, err error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get(c.UserinfoURL)
	if err != nil {
		return "", "", "", fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("userinfo: status %d", resp.StatusCode)
	}
	// 手动解析（避免引入额外 JSON 依赖之外的东西，标准库足够）
	type userInfo struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Username          string `json:"username"`
		Login             string `json:"login"`
		Email             string `json:"email"`
	}
	var ui userInfo
	dec := newJSONDecoder(resp.Body)
	if err := dec.Decode(&ui); err != nil {
		return "", "", "", fmt.Errorf("userinfo decode: %w", err)
	}
	sub = ui.Sub
	username = ui.PreferredUsername
	if username == "" {
		username = ui.Username
	}
	if username == "" {
		username = ui.Login
	}
	email = ui.Email
	if sub == "" {
		return "", "", "", errors.New("userinfo: missing sub claim")
	}
	return sub, username, email, nil
}

// RandomState 生成防 CSRF 的随机 state。
func RandomState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// EnsureOAuthUser 取 username/email 对应的本地用户；不存在则自动注册（随机密码）。
func EnsureOAuthUser(ctx context.Context, q db.Querier, username, email string) (db.User, error) {
	if u, err := q.GetUserByUsername(ctx, NormalizeUsername(username)); err == nil {
		return u, nil
	}
	// 自动注册：随机 32 字节密码（用户不会用它登录，走 OAuth）
	pass := randomHex(32)
	return RegisterOAuthUser(ctx, q, username, email, pass)
}
