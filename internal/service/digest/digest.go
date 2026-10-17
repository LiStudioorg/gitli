// SPDX-License-Identifier: AGPL-3.0
// Package digest aggregates unread notifications into periodic email digests.
package digest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Frequency values for email digests.
const (
	Realtime = "realtime"
	Daily    = "daily"
	Weekly   = "weekly"
	Off      = "off"
)

// ErrNotFound indicates missing preferences.
var ErrNotFound = errors.New("digest: not found")

// Service manages digest preferences and aggregation.
type Service struct {
	st *store.Store
}

// New builds a digest Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// GetFrequency returns the user's digest frequency (default realtime).
func (s *Service) GetFrequency(ctx context.Context, userID int64) (string, error) {
	var freq string
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT frequency FROM user_notification_prefs WHERE user_id = ?`), userID).Scan(&freq)
	if errors.Is(err, sql.ErrNoRows) {
		return Realtime, nil
	}
	if err != nil {
		return "", err
	}
	return freq, nil
}

// SetFrequency updates the user's digest frequency.
func (s *Service) SetFrequency(ctx context.Context, userID int64, freq string) error {
	if !validFrequency(freq) {
		return fmt.Errorf("digest: invalid frequency %q", freq)
	}
	q := s.st.Rebind(`INSERT INTO user_notification_prefs (user_id, frequency, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET frequency = excluded.frequency, updated_at = excluded.updated_at`)
	if _, err := s.st.DB.ExecContext(ctx, q, userID, freq, time.Now().UTC()); err != nil {
		if _, err2 := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE user_notification_prefs SET frequency = ?, updated_at = ? WHERE user_id = ?`), freq, time.Now().UTC(), userID); err2 != nil {
			return err
		}
	}
	return nil
}

func validFrequency(f string) bool {
	switch f {
	case Realtime, Daily, Weekly, Off:
		return true
	}
	return false
}

// Interval returns the minimum time between digests for a frequency.
func Interval(freq string) time.Duration {
	switch freq {
	case Daily:
		return 24 * time.Hour
	case Weekly:
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}

// Due reports whether a digest is due given the frequency and last send time.
func Due(freq string, lastSent *time.Time, now time.Time) bool {
	if freq != Daily && freq != Weekly {
		return false
	}
	if lastSent == nil {
		return true
	}
	return now.Sub(*lastSent) >= Interval(freq)
}

// Candidate is a user eligible for a digest.
type Candidate struct {
	UserID    int64
	Email     string
	Frequency string
	LastSent  *time.Time
}

// DueCandidates returns users whose digest is due.
func (s *Service) DueCandidates(ctx context.Context, now time.Time) ([]Candidate, error) {
	q := s.st.Rebind(`SELECT p.user_id, u.email, p.frequency, p.last_sent_at
		FROM user_notification_prefs p JOIN users u ON u.id = p.user_id
		WHERE p.frequency IN (?, ?)`)
	rows, err := s.st.DB.QueryContext(ctx, q, Daily, Weekly)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		var last sql.NullTime
		if err := rows.Scan(&c.UserID, &c.Email, &c.Frequency, &last); err != nil {
			return nil, err
		}
		if last.Valid {
			t := last.Time
			c.LastSent = &t
		}
		if c.Email != "" && Due(c.Frequency, c.LastSent, now) {
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

// MarkSent records that a digest was sent to a user.
func (s *Service) MarkSent(ctx context.Context, userID int64, at time.Time) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE user_notification_prefs SET last_sent_at = ? WHERE user_id = ?`), at.UTC(), userID)
	return err
}

// Notification is an unread notification included in a digest.
type Notification struct {
	Title string
	URL   string
}

// Unread returns a user's unread notifications (bounded).
func (s *Service) Unread(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT title, url FROM notifications WHERE user_id = ? AND is_read = ? ORDER BY id DESC LIMIT ?`), userID, false, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.Title, &n.URL); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// RenderHTML builds a simple HTML digest email body.
func RenderHTML(username, baseURL string, items []Notification, prefsURL string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><body style="font-family:sans-serif">`)
	b.WriteString("<h2>gitli 通知摘要</h2>")
	if username != "" {
		fmt.Fprintf(&b, "<p>你好，%s：</p>", username)
	}
	if len(items) == 0 {
		b.WriteString("<p>没有新的未读通知。</p>")
	} else {
		b.WriteString("<ul>")
		for _, n := range items {
			link := n.URL
			if link != "" && baseURL != "" && !strings.HasPrefix(link, "http") {
				link = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(link, "/")
			}
			if link != "" {
				fmt.Fprintf(&b, `<li><a href="%s">%s</a></li>`, link, n.Title)
			} else {
				fmt.Fprintf(&b, "<li>%s</li>", n.Title)
			}
		}
		b.WriteString("</ul>")
	}
	if prefsURL != "" {
		fmt.Fprintf(&b, `<p style="color:#888;font-size:12px">修改通知频率或退订：<a href="%s">通知设置</a></p>`, prefsURL)
	}
	b.WriteString("</body></html>")
	return b.String()
}
