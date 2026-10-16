// SPDX-License-Identifier: AGPL-3.0
package audit

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// AlertRule triggers when an event exceeds a threshold within a window.
type AlertRule struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Event       string     `json:"event"`
	Threshold   int        `json:"threshold"`
	WindowSec   int        `json:"window_seconds"`
	Channel     string     `json:"channel"` // email | webhook
	Target      string     `json:"target"`
	Enabled     bool       `json:"enabled"`
	LastFiredAt *time.Time `json:"last_fired_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ErrAlertNotFound indicates a missing alert rule.
var ErrAlertNotFound = errors.New("audit: alert rule not found")

// CreateAlertRule stores a new alert rule.
func (s *Service) CreateAlertRule(ctx context.Context, r AlertRule) (*AlertRule, error) {
	if r.Channel == "" {
		r.Channel = "email"
	}
	if r.Threshold <= 0 {
		r.Threshold = 10
	}
	if r.WindowSec <= 0 {
		r.WindowSec = 300
	}
	q := s.st.Rebind(`INSERT INTO audit_alert_rules (name, event, threshold, window_seconds, channel, target, enabled) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, r.Name, r.Event, r.Threshold, r.WindowSec, r.Channel, r.Target, r.Enabled)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.AlertRuleByID(ctx, id)
}

// ListAlertRules returns all alert rules.
func (s *Service) ListAlertRules(ctx context.Context) ([]*AlertRule, error) {
	q := s.st.Rebind(`SELECT id, name, event, threshold, window_seconds, channel, target, enabled, last_fired_at, created_at FROM audit_alert_rules ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*AlertRule
	for rows.Next() {
		var r AlertRule
		var last sql.NullTime
		if err := rows.Scan(&r.ID, &r.Name, &r.Event, &r.Threshold, &r.WindowSec, &r.Channel, &r.Target, &r.Enabled, &last, &r.CreatedAt); err != nil {
			return nil, err
		}
		if last.Valid {
			t := last.Time
			r.LastFiredAt = &t
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// AlertRuleByID loads a rule by id.
func (s *Service) AlertRuleByID(ctx context.Context, id int64) (*AlertRule, error) {
	q := s.st.Rebind(`SELECT id, name, event, threshold, window_seconds, channel, target, enabled, last_fired_at, created_at FROM audit_alert_rules WHERE id = ?`)
	var r AlertRule
	var last sql.NullTime
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&r.ID, &r.Name, &r.Event, &r.Threshold, &r.WindowSec, &r.Channel, &r.Target, &r.Enabled, &last, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAlertNotFound
	}
	if err != nil {
		return nil, err
	}
	if last.Valid {
		t := last.Time
		r.LastFiredAt = &t
	}
	return &r, nil
}

// UpdateAlertRule updates mutable fields of a rule.
func (s *Service) UpdateAlertRule(ctx context.Context, id int64, name, event string, threshold, windowSec int, channel, target string, enabled bool) error {
	sets := []string{}
	args := []any{}
	if name != "" {
		sets = append(sets, "name = ?")
		args = append(args, name)
	}
	if event != "" {
		sets = append(sets, "event = ?")
		args = append(args, event)
	}
	if threshold > 0 {
		sets = append(sets, "threshold = ?")
		args = append(args, threshold)
	}
	if windowSec > 0 {
		sets = append(sets, "window_seconds = ?")
		args = append(args, windowSec)
	}
	if channel != "" {
		sets = append(sets, "channel = ?")
		args = append(args, channel)
	}
	if target != "" {
		sets = append(sets, "target = ?")
		args = append(args, target)
	}
	sets = append(sets, "enabled = ?")
	args = append(args, enabled)
	args = append(args, id)
	q := s.st.Rebind(`UPDATE audit_alert_rules SET ` + strings.Join(sets, ", ") + ` WHERE id = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, args...)
	return err
}

// DeleteAlertRule removes a rule.
func (s *Service) DeleteAlertRule(ctx context.Context, id int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM audit_alert_rules WHERE id = ?`), id)
	return err
}

// AlertFiring is a rule that has exceeded its threshold.
type AlertFiring struct {
	Rule  *AlertRule
	Count int
}

// EvaluateAlerts checks all enabled rules and returns those that fired.
func (s *Service) EvaluateAlerts(ctx context.Context) ([]AlertFiring, error) {
	rules, err := s.ListAlertRules(ctx)
	if err != nil {
		return nil, err
	}
	var fired []AlertFiring
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		since := time.Now().Add(-time.Duration(r.WindowSec) * time.Second)
		q := s.st.Rebind(`SELECT COUNT(*) FROM audit_logs WHERE event LIKE ? AND created_at >= ?`)
		var n int
		if err := s.st.DB.QueryRowContext(ctx, q, r.Event+"%", dbTime(since)).Scan(&n); err != nil {
			return nil, err
		}
		if n >= r.Threshold {
			fired = append(fired, AlertFiring{Rule: r, Count: n})
		}
	}
	return fired, nil
}

// MarkAlertFired updates a rule's last_fired_at timestamp.
func (s *Service) MarkAlertFired(ctx context.Context, id int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE audit_alert_rules SET last_fired_at = ? WHERE id = ?`), time.Now().UTC(), id)
	return err
}
