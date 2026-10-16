// SPDX-License-Identifier: AGPL-3.0
// Package org2fa manages per-organization two-factor authentication policies.
package org2fa

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service manages org 2FA policies.
type Service struct {
	st *store.Store
}

// New builds an org2fa Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Set upserts whether 2FA is required for an organization.
func (s *Service) Set(ctx context.Context, orgID int64, required bool) error {
	del := s.st.Rebind(`DELETE FROM org_2fa_policies WHERE org_id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, del, orgID); err != nil {
		return fmt.Errorf("org2fa: delete: %w", err)
	}
	q := s.st.Rebind(`INSERT INTO org_2fa_policies (org_id, required) VALUES (?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, orgID, required); err != nil {
		return fmt.Errorf("org2fa: insert: %w", err)
	}
	return nil
}

// Get reports whether 2FA is required for an organization.
func (s *Service) Get(ctx context.Context, orgID int64) (bool, error) {
	q := s.st.Rebind(`SELECT required FROM org_2fa_policies WHERE org_id = ?`)
	var required bool
	err := s.st.DB.QueryRowContext(ctx, q, orgID).Scan(&required)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("org2fa: get: %w", err)
	}
	return required, nil
}
