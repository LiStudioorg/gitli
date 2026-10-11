// SPDX-License-Identifier: AGPL-3.0
// Package store provides shared database access for the service layer.
package store

import "github.com/LiStudioorg/gitli/internal/db"

// Store holds shared dependencies for services.
type Store struct {
	DB *db.DB
}

// New builds a Store.
func New(database *db.DB) *Store {
	return &Store{DB: database}
}

// Rebind rewrites placeholders for the active dialect.
func (s *Store) Rebind(q string) string { return s.DB.Rebind(q) }
