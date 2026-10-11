// SPDX-License-Identifier: AGPL-3.0
// Package audit records security-relevant actions.
package audit

import (
	"context"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Entry describes an audit record.
type Entry struct {
	ActorID    *int64
	Action     string
	TargetType string
	TargetID   *int64
	IP         string
	Detail     string
}

// Service writes audit logs.
type Service struct {
	st *store.Store
}

// New builds an audit Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Record writes an audit entry; failures are non-fatal to callers but returned.
func (s *Service) Record(ctx context.Context, e Entry) error {
	q := s.st.Rebind(`INSERT INTO audit_logs (actor_id, action, target_type, target_id, ip, detail) VALUES (?, ?, ?, ?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, e.ActorID, e.Action, e.TargetType, e.TargetID, e.IP, e.Detail)
	return err
}

// List returns recent audit logs.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.st.Rebind(`SELECT id, actor_id, action, target_type, target_id, ip, detail, created_at FROM audit_logs ORDER BY id DESC LIMIT ? OFFSET ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		if err := rows.Scan(&a.ID, &a.ActorID, &a.Action, &a.TargetType, &a.TargetID, &a.IP, &a.Detail, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}
