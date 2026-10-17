// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/service/oauth"
)

// handleOAuthToken implements POST /oauth/token (authorization_code grant).
func (s *Server) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if s.deps.OAuth == nil {
		writeError(w, http.StatusNotFound, "oauth disabled")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid form")
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		writeError(w, http.StatusBadRequest, "unsupported grant_type")
		return
	}
	clientID := r.Form.Get("client_id")
	secret := r.Form.Get("client_secret")
	if u, p, ok := r.BasicAuth(); ok {
		clientID, secret = u, p
	}
	resp, err := s.deps.OAuth.ExchangeCode(r.Context(), clientID, secret, r.Form.Get("code"), r.Form.Get("redirect_uri"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, resp)
}

// handleOAuthUserInfo implements GET /oauth/userinfo (Bearer token).
func (s *Server) handleOAuthUserInfo(w http.ResponseWriter, r *http.Request) {
	if s.deps.OAuth == nil {
		writeError(w, http.StatusNotFound, "oauth disabled")
		return
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	userID, scopes, err := s.deps.OAuth.TokenUser(r.Context(), strings.TrimPrefix(auth, "Bearer "))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	u, err := s.deps.Users.ByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        u.ID,
		"username":  u.Username,
		"email":     u.Email,
		"full_name": u.FullName,
		"scopes":    scopes,
	})
}

// handleOAuthAuthorizeInfo returns app metadata for the consent screen.
func (s *Server) handleOAuthAuthorizeInfo(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	if s.deps.OAuth == nil {
		writeError(w, http.StatusNotFound, "oauth disabled")
		return
	}
	app, err := s.deps.OAuth.AppByClientID(r.Context(), r.URL.Query().Get("client_id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "application not found")
		return
	}
	redirect := r.URL.Query().Get("redirect_uri")
	if !oauth.ValidateRedirectURI(app, redirect) {
		writeError(w, http.StatusBadRequest, "redirect_uri not registered")
		return
	}
	scopes := oauth.FilterScopes(strings.Fields(r.URL.Query().Get("scope")))
	writeJSON(w, http.StatusOK, map[string]any{
		"app":          map[string]any{"name": app.Name, "client_id": app.ClientID},
		"redirect_uri": redirect,
		"scopes":       scopes,
		"state":        r.URL.Query().Get("state"),
	})
}

type oauthApproveReq struct {
	ClientID    string   `json:"client_id"`
	RedirectURI string   `json:"redirect_uri"`
	Scopes      []string `json:"scopes"`
	State       string   `json:"state"`
}

// handleOAuthApprove grants consent and issues an authorization code.
func (s *Server) handleOAuthApprove(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	if s.deps.OAuth == nil {
		writeError(w, http.StatusNotFound, "oauth disabled")
		return
	}
	var req oauthApproveReq
	if !decodeJSON(w, r, &req) {
		return
	}
	app, err := s.deps.OAuth.AppByClientID(r.Context(), req.ClientID)
	if err != nil {
		writeError(w, http.StatusNotFound, "application not found")
		return
	}
	if !oauth.ValidateRedirectURI(app, req.RedirectURI) {
		writeError(w, http.StatusBadRequest, "redirect_uri not registered")
		return
	}
	code, err := s.deps.OAuth.IssueCode(r.Context(), oauth.AuthorizeParams{
		AppID: app.ID, UserID: u.ID, RedirectURI: req.RedirectURI,
		Scopes: oauth.FilterScopes(req.Scopes),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	redirect := req.RedirectURI + "?code=" + code
	if req.State != "" {
		redirect += "&state=" + req.State
	}
	writeJSON(w, http.StatusOK, map[string]any{"redirect": redirect})
}

// --- app management ---

type createOAuthAppReq struct {
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

func (s *Server) handleListOAuthApps(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	apps, err := s.deps.OAuth.ListAppsByOwner(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *Server) handleCreateOAuthApp(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createOAuthAppReq
	if !decodeJSON(w, r, &req) {
		return
	}
	app, secret, err := s.deps.OAuth.CreateApp(r.Context(), u.ID, req.Name, req.RedirectURIs, oauth.FilterScopes(req.Scopes))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "oauth_app.create", "success")
	writeJSON(w, http.StatusCreated, map[string]any{"app": app, "client_secret": secret})
}

func (s *Server) handleDeleteOAuthApp(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.OAuth.DeleteApp(r.Context(), u.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "oauth_app.delete", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListOAuthAuthorizations(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	auths, err := s.deps.OAuth.ListAuthorizations(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, auths)
}

func (s *Server) handleRevokeOAuthAuthorization(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	appID, err := pathInt(r, "appId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid app id")
		return
	}
	if err := s.deps.OAuth.RevokeAuthorization(r.Context(), u.ID, appID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "oauth_app.revoke", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
