// SPDX-License-Identifier: AGPL-3.0
// Package webhook delivers repository webhooks with HMAC signing and SSRF guards.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service manages webhook configuration and delivery.
type Service struct {
	st      *store.Store
	client  *http.Client
	maxBody int64
}

// New builds a webhook Service.
func New(st *store.Store) *Service {
	return &Service{
		st: st,
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		maxBody: 1 << 20,
	}
}

// Create registers a webhook.
func (s *Service) Create(ctx context.Context, repoID *int64, orgID *int64, rawURL, secret string, events []string) (*domain.Webhook, error) {
	if err := validateURL(rawURL); err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO webhooks (repo_id, org_id, url, secret, events) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, orgID, rawURL, secret, strings.Join(events, ","))
	if err != nil {
		return nil, fmt.Errorf("webhook: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return &domain.Webhook{ID: id, RepoID: repoID, OrgID: orgID, URL: rawURL, Secret: secret, Events: events, Active: true}, nil
}

// ListForRepo lists webhooks for a repository.
func (s *Service) ListForRepo(ctx context.Context, repoID int64) ([]*domain.Webhook, error) {
	q := s.st.Rebind(`SELECT id, repo_id, org_id, url, secret, events, active, created_at FROM webhooks WHERE repo_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Webhook
	for rows.Next() {
		var w domain.Webhook
		var events string
		if err := rows.Scan(&w.ID, &w.RepoID, &w.OrgID, &w.URL, &w.Secret, &events, &w.Active, &w.CreatedAt); err != nil {
			return nil, err
		}
		w.Events = splitCSV(events)
		out = append(out, &w)
	}
	return out, rows.Err()
}

// ByID loads a webhook by id.
func (s *Service) ByID(ctx context.Context, id int64) (*domain.Webhook, error) {
	q := s.st.Rebind(`SELECT id, repo_id, org_id, url, secret, events, active, created_at FROM webhooks WHERE id = ?`)
	var w domain.Webhook
	var events string
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&w.ID, &w.RepoID, &w.OrgID, &w.URL, &w.Secret, &events, &w.Active, &w.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("webhook: not found")
	}
	if err != nil {
		return nil, err
	}
	w.Events = splitCSV(events)
	return &w, nil
}

// Deliver posts a JSON payload signed with HMAC-SHA256.
func (s *Service) Deliver(ctx context.Context, w *domain.Webhook, event string, payload any) (*domain.WebhookDelivery, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err := validateURL(w.URL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitli-Event", event)
	if w.Secret != "" {
		req.Header.Set("X-Gitli-Signature", "sha256="+sign(w.Secret, body))
	}
	delivery := &domain.WebhookDelivery{WebhookID: w.ID, Event: event, RequestBody: string(body), Attempts: 1}
	resp, err := s.client.Do(req)
	if err != nil {
		delivery.ResponseBody = err.Error()
		_ = s.record(ctx, delivery)
		return delivery, fmt.Errorf("webhook: deliver: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, s.maxBody))
	delivery.StatusCode = resp.StatusCode
	delivery.ResponseBody = string(respBody)
	now := time.Now().UTC()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.DeliveredAt = &now
	}
	if err := s.record(ctx, delivery); err != nil {
		return delivery, err
	}
	return delivery, nil
}

func (s *Service) record(ctx context.Context, d *domain.WebhookDelivery) error {
	q := s.st.Rebind(`INSERT INTO webhook_deliveries (webhook_id, event, status_code, request_body, response_body, attempts, delivered_at) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, d.WebhookID, d.Event, d.StatusCode, d.RequestBody, d.ResponseBody, d.Attempts, d.DeliveredAt)
	return err
}

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// validateURL rejects non-http schemes and private/metadata addresses (SSRF).
func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("webhook: invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook: unsupported scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("webhook: missing host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil // allow unresolved; delivery will error
	}
	for _, ip := range ips {
		if isPrivate(ip) {
			return fmt.Errorf("webhook: refusing private address %s", ip)
		}
	}
	return nil
}

func isPrivate(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	return false
}
