package web

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"gitli/internal/auth"
	"gitli/internal/db"
)

// ---- PAT 管理 ----

type patView struct {
	ID      int64
	Name    string
	Scope   string
	Created int64
	LastUse int64
}

func (s *Server) showTokens(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.renderError(w, http.StatusNotFound, "需要登录")
		return
	}
	pats, _ := s.q.ListPATsFull(r.Context(), user.ID)
	views := make([]patView, 0, len(pats))
	for _, p := range pats {
		var lu int64
		if p.LastUsedAt.Valid {
			lu = p.LastUsedAt.Int64
		}
		views = append(views, patView{ID: p.ID, Name: p.Name, Scope: p.Scope, Created: p.CreatedAt, LastUse: lu})
	}
	s.render(w, http.StatusOK, "tokens", pageData{
		Title: "令牌", User: user, CSRF: CSRFFromContext(r.Context()),
		Data: map[string]any{"Pats": views, "NewToken": "", "Flash": Flash{}},
	})
}

func (s *Server) doCreateToken(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.renderError(w, http.StatusNotFound, "需要登录")
		return
	}
	name := strings.TrimSpace(r.PostFormValue("name"))
	scope := r.PostFormValue("scope") // "" / "all" / "repo:<id>:<r|w>"
	if name == "" {
		name = "token"
	}
	if scope == "" {
		scope = "all"
	}
	if scope != "all" && !strings.HasPrefix(scope, "repo:") {
		s.renderError(w, http.StatusBadRequest, "无效作用域")
		return
	}
	var expiresAt sql.NullInt64
	if v := r.PostFormValue("expires_days"); v != "" {
		days := 0
		for _, c := range v {
			if c < '0' || c > '9' {
				days = -1
				break
			}
			days = days*10 + int(c-'0')
		}
		if days > 0 && days <= 3650 {
			expiresAt = sql.NullInt64{Int64: time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Unix(), Valid: true}
		}
	}
	token := auth.NewToken()
	if _, err := s.q.CreateScopedPAT(r.Context(), db.CreateScopedPATParams{
		UserID: user.ID, Name: name, TokenHash: auth.HashTokenForStore(token),
		Scope: scope, CreatedAt: time.Now().UTC().Unix(), ExpiresAt: expiresAt,
	}); err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	pats, _ := s.q.ListPATsFull(r.Context(), user.ID)
	views := make([]patView, 0, len(pats))
	for _, p := range pats {
		var lu int64
		if p.LastUsedAt.Valid {
			lu = p.LastUsedAt.Int64
		}
		views = append(views, patView{ID: p.ID, Name: p.Name, Scope: p.Scope, Created: p.CreatedAt, LastUse: lu})
	}
	s.render(w, http.StatusOK, "tokens", pageData{
		Title: "令牌", User: user, CSRF: CSRFFromContext(r.Context()),
		Data: map[string]any{"Pats": views, "NewToken": token, "Flash": Flash{}},
	})
}

func (s *Server) doDeleteToken(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.renderError(w, http.StatusNotFound, "需要登录")
		return
	}
	id := pathInt64(r.PathValue("id"))
	_ = s.q.DeletePAT(r.Context(), db.DeletePATParams{ID: id, UserID: user.ID})
	http.Redirect(w, r, "/settings/tokens", http.StatusSeeOther)
}

// ---- TOTP 两步验证 ----

func (s *Server) showTotpSetup(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.renderError(w, http.StatusNotFound, "需要登录")
		return
	}
	if user.TotpEnabled == 1 {
		http.Redirect(w, r, "/settings/security", http.StatusSeeOther)
		return
	}
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// 暂存 secret 到 session 关联的临时存储（简单起见放 form hidden，校验时要求与提交一致）
	uri := auth.TOTPProvisioningURI(secret, user.Username)
	s.render(w, http.StatusOK, "totp_setup", pageData{
		Title: "启用两步验证", User: user, CSRF: CSRFFromContext(r.Context()),
		Data: map[string]any{"Secret": secret, "URI": uri, "Flash": Flash{}},
	})
}

func (s *Server) doTotpEnable(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.renderError(w, http.StatusNotFound, "需要登录")
		return
	}
	secret := strings.TrimSpace(r.PostFormValue("secret"))
	code := strings.TrimSpace(r.PostFormValue("code"))
	if secret == "" || !auth.VerifyTOTP(r.Context(), s.q, user.ID, secret, code) {
		uri := auth.TOTPProvisioningURI(secret, user.Username)
		s.render(w, http.StatusBadRequest, "totp_setup", pageData{
			Title: "启用两步验证", User: user, CSRF: CSRFFromContext(r.Context()),
			Data: map[string]any{"Secret": secret, "URI": uri,
				"Flash": Flash{Kind: "error", Message: "验证码错误，请重试"}},
		})
		return
	}
	_ = s.q.SetTOTPSecret(r.Context(), db.SetTOTPSecretParams{TotpSecret: secret, TotpEnabled: 1, ID: user.ID})
	codes, err := auth.GenerateRecoveryCodes(r.Context(), s.q, user.ID)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.render(w, http.StatusOK, "totp_recovery", pageData{
		Title: "备用恢复码", User: user, CSRF: CSRFFromContext(r.Context()),
		Data: map[string]any{"Codes": codes},
	})
}

func (s *Server) doTotpDisable(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.renderError(w, http.StatusNotFound, "需要登录")
		return
	}
	code := strings.TrimSpace(r.PostFormValue("code"))
	if user.TotpEnabled == 1 && !auth.VerifyTOTP(r.Context(), s.q, user.ID, user.TotpSecret, code) &&
		!auth.UseRecoveryCode(r.Context(), s.q, user.ID, code) {
		s.render(w, http.StatusBadRequest, "security", pageData{
			Title: "安全设置", User: user, CSRF: CSRFFromContext(r.Context()),
			Data: map[string]any{"Flash": Flash{Kind: "error", Message: "验证码错误"}},
		})
		return
	}
	_ = s.q.DisableTOTP(r.Context(), user.ID)
	_ = s.q.DeleteRecoveryCodes(r.Context(), user.ID)
	http.Redirect(w, r, "/settings/security", http.StatusSeeOther)
}

func (s *Server) showSecurity(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		s.renderError(w, http.StatusNotFound, "需要登录")
		return
	}
	s.render(w, http.StatusOK, "security", pageData{
		Title: "安全设置", User: user, CSRF: CSRFFromContext(r.Context()),
		Data: map[string]any{"Flash": Flash{}},
	})
}

// ---- TOTP 登录二次验证 ----

// pending2FA 登录第一阶段通过但待 TOTP 验证的会话（内存态，重启即失效可接受）。
type pending2FA struct {
	UserID    int64
	ExpiresAt time.Time
}

var twoFAPending = map[string]pending2FA{} // key: 临时 token

func newPendingToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// handleLogin2FA 密码校验通过但用户启用了 TOTP：进入二次验证（不建正式 session）。
func (s *Server) handleLogin2FA(w http.ResponseWriter, r *http.Request, userID int64, _ func(sid string)) {
	pt := newPendingToken()
	twoFAPending[pt] = pending2FA{UserID: userID, ExpiresAt: time.Now().Add(5 * time.Minute)}
	s.render(w, http.StatusOK, "totp_verify", pageData{
		Title: "两步验证", User: nil,
		Data: map[string]any{"PendingToken": pt, "Flash": Flash{}},
	})
}

func (s *Server) doTotpVerify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest, "invalid form")
		return
	}
	pt := r.PostFormValue("pending_token")
	code := strings.TrimSpace(r.PostFormValue("code"))
	useRecovery := r.PostFormValue("recovery") == "1"

	p, ok := twoFAPending[pt]
	if !ok {
		s.renderError(w, http.StatusUnauthorized, "会话过期，请重新登录")
		return
	}
	if time.Now().After(p.ExpiresAt) {
		delete(twoFAPending, pt)
		s.renderError(w, http.StatusUnauthorized, "会话过期，请重新登录")
		return
	}
	user, err := s.users.GetByID(r.Context(), p.UserID)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	okCode := false
	if useRecovery {
		okCode = auth.UseRecoveryCode(r.Context(), s.q, user.ID, code)
	} else {
		okCode = auth.VerifyTOTP(r.Context(), s.q, user.ID, user.TotpSecret, code)
	}
	if !okCode {
		s.render(w, http.StatusUnauthorized, "totp_verify", pageData{
			Title: "两步验证",
			Data: map[string]any{"PendingToken": pt,
				"Flash": Flash{Kind: "error", Message: "验证码错误"}},
		})
		return
	}
	delete(twoFAPending, pt)
	// 建立 session
	sid, _, err := auth.SessionForUser(r.Context(), s.q, user)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: sid, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 14 * 24 * 3600,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var errNoPending = errors.New("no pending 2fa session")

func pathInt64(s string) int64 {
	n := int64(0)
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
	}
	return n
}
