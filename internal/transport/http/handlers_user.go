// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/pat"
)

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	writeJSON(w, http.StatusOK, publicUser(u))
}

type updateProfileReq struct {
	FullName string `json:"full_name"`
	Bio      string `json:"bio"`
	Website  string `json:"website"`
	Location string `json:"location"`
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req updateProfileReq
	if !decodeJSON(w, r, &req) {
		return
	}
	u.FullName, u.Bio, u.Website, u.Location = req.FullName, req.Bio, req.Website, req.Location
	if err := s.deps.Users.UpdateProfile(r.Context(), u); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, publicUser(u))
}

type changePasswordReq struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	TOTPCode        string `json:"totp_code"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req changePasswordReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if !s.reauth(r, u, req.CurrentPassword, req.TOTPCode) {
		s.auditAction(r, u.ID, "user.password_change", "failure")
		writeError(w, http.StatusUnauthorized, "二次认证失败")
		return
	}
	if err := s.deps.Auth.UpdatePassword(r.Context(), u, req.NewPassword); err != nil {
		s.auditAction(r, u.ID, "user.password_change", "failure")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.password_change", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type renameReq struct {
	NewUsername string `json:"new_username"`
	Password    string `json:"password"`
	TOTPCode    string `json:"totp_code"`
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req renameReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateName("username", req.NewUsername); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !s.reauth(r, u, req.Password, req.TOTPCode) {
		s.auditAction(r, u.ID, "user.rename", "failure")
		writeError(w, http.StatusUnauthorized, "二次认证失败")
		return
	}
	if err := s.deps.Users.Rename(r.Context(), u.ID, req.NewUsername); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.rename", "success")
	writeJSON(w, http.StatusOK, map[string]any{"username": req.NewUsername})
}

func (s *Server) handleTOTPBegin(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	secret, uri, err := s.deps.Auth.BeginTOTP(r.Context(), u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"secret": secret, "otpauth_url": uri})
}

type totpCodeReq struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (s *Server) handleTOTPEnable(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req totpCodeReq
	if !decodeJSON(w, r, &req) {
		return
	}
	codes, err := s.deps.Auth.EnableTOTP(r.Context(), u, req.Code)
	if err != nil {
		s.auditAction(r, u.ID, "user.totp_enable", "failure")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.totp_enable", "success")
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (s *Server) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req totpCodeReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if !s.reauth(r, u, req.Password, req.Code) {
		s.auditAction(r, u.ID, "user.totp_disable", "failure")
		writeError(w, http.StatusUnauthorized, "二次认证失败")
		return
	}
	if err := s.deps.Auth.DisableTOTP(r.Context(), u, req.Code); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.totp_disable", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type addKeyReq struct {
	Title           string `json:"title"`
	PublicKey       string `json:"public_key"`
	CanAuthenticate *bool  `json:"can_authenticate"`
	CanSign         *bool  `json:"can_sign"`
}

func (s *Server) handleListSSHKeys(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	keys, err := s.deps.Users.ListSSHKeys(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleAddSSHKey(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req addKeyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	canAuth := true // default: authentication enabled
	if req.CanAuthenticate != nil {
		canAuth = *req.CanAuthenticate
	}
	canSign := false // default: signing disabled
	if req.CanSign != nil {
		canSign = *req.CanSign
	}
	key, err := s.deps.Users.AddSSHKey(r.Context(), u.ID, req.Title, req.PublicKey, canAuth, canSign)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.sshkey_add",
		fmt.Sprintf("fingerprint=%s authenticate=%v sign=%v", key.Fingerprint, canAuth, canSign))
	writeJSON(w, http.StatusCreated, key)
}

type updateKeyReq struct {
	CanAuthenticate bool `json:"can_authenticate"`
	CanSign         bool `json:"can_sign"`
}

func (s *Server) handleUpdateSSHKey(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req updateKeyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Users.UpdateSSHKeyPurposes(r.Context(), u.ID, id, req.CanAuthenticate, req.CanSign); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.sshkey_update",
		fmt.Sprintf("id=%d authenticate=%v sign=%v", id, req.CanAuthenticate, req.CanSign))
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleDeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.Users.DeleteSSHKey(r.Context(), u.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.sshkey_delete", fmt.Sprintf("id=%d", id))
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// --- GPG / signing keys ---

type addSigningKeyReq struct {
	Title     string `json:"title"`
	PublicKey string `json:"public_key"`
	KeyType   string `json:"key_type"`
}

func (s *Server) handleListSigningKeys(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	keys, err := s.deps.Users.ListSigningKeys(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) handleAddSigningKey(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req addSigningKeyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	key, err := s.deps.Users.AddSigningKey(r.Context(), u.ID, req.KeyType, req.Title, req.PublicKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.gpgkey_add", fmt.Sprintf("fingerprint=%s", key.Fingerprint))
	writeJSON(w, http.StatusCreated, key)
}

func (s *Server) handleDeleteSigningKey(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.Users.DeleteSigningKey(r.Context(), u.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.gpgkey_delete", fmt.Sprintf("id=%d", id))
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type createTokenReq struct {
	Name   string          `json:"name"`
	Kind   string          `json:"kind"`
	Scopes []string        `json:"scopes"`
	Grants []tokenGrantReq `json:"repo_grants"`
}

type tokenGrantReq struct {
	RepoID     int64  `json:"repo_id"`
	Permission string `json:"permission"`
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	tokens, err := s.deps.Tokens.List(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, t := range tokens {
		t.TokenHash = ""
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createTokenReq
	if !decodeJSON(w, r, &req) {
		return
	}
	kind := domain.TokenClassic
	if req.Kind == string(domain.TokenFineGrained) {
		kind = domain.TokenFineGrained
	}
	if kind == domain.TokenClassic && len(req.Scopes) == 0 && len(req.Grants) == 0 {
		req.Scopes = nil
	}
	var grants []domain.TokenRepoGrant
	for _, g := range req.Grants {
		perm := g.Permission
		if perm == "" {
			perm = "read"
		}
		grants = append(grants, domain.TokenRepoGrant{RepoID: g.RepoID, Permission: perm})
	}
	created, err := s.deps.Tokens.Create(r.Context(), pat.CreateParams{
		UserID: u.ID, Name: req.Name, Kind: kind, Scopes: req.Scopes, Grants: grants,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.pat_create", "success")
	writeJSON(w, http.StatusCreated, map[string]any{"id": created.Token.ID, "token": created.Raw, "name": created.Token.Name})
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.Tokens.Revoke(r.Context(), u.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "user.pat_revoke", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) auditAction(r *http.Request, userID int64, action, result string) {
	if s.deps.Audit == nil {
		return
	}
	uid := userID
	resource := action
	if i := strings.IndexByte(action, '.'); i > 0 {
		resource = action[:i]
	}
	actorType := "user"
	if u := userFrom(r.Context()); u != nil && u.IsAdmin {
		actorType = "admin"
	}
	s.deps.Audit.Log(audit.Entry{
		ActorID:    &uid,
		ActorType:  actorType,
		Action:     action,
		Event:      action,
		Resource:   resource,
		TargetType: resource,
		Result:     result,
		IP:         clientIP(r),
		UserAgent:  r.UserAgent(),
		RequestID:  requestIDFrom(r),
	})
}

// requestIDFrom extracts (or is empty for) the request correlation id.
func requestIDFrom(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}

func publicUser(u *domain.User) map[string]any {
	return map[string]any{
		"id":           u.ID,
		"username":     u.Username,
		"email":        u.Email,
		"full_name":    u.FullName,
		"bio":          u.Bio,
		"avatar_url":   u.AvatarURL,
		"website":      u.Website,
		"location":     u.Location,
		"is_admin":     u.IsAdmin,
		"totp_enabled": u.TOTPEnabled,
	}
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	raw := ""
	if c, err := r.Cookie("gitli_session"); err == nil {
		raw = c.Value
	}
	sessions, err := s.deps.Auth.ListSessions(r.Context(), u.ID, raw)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.Auth.RevokeSession(r.Context(), u.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "auth.session.revoke", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleRevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	raw := ""
	if c, err := r.Cookie("gitli_session"); err == nil {
		raw = c.Value
	}
	if err := s.deps.Auth.RevokeOtherSessions(r.Context(), u.ID, raw); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "auth.session.revoke", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
