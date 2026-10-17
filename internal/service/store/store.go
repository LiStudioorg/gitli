// SPDX-License-Identifier: AGPL-3.0
// Package store provides shared database access for the service layer.
package store

import (
	"strings"

	"github.com/LiStudioorg/gitli/internal/db"
)

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

// EscapeLike escapes SQL LIKE wildcards so user input is matched literally.
// Callers still use a parameterised query; this only neutralises % and _.
// The backslash is escaped first because it is the escape character itself.
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

// ContainsPattern builds a "%value%" pattern with wildcards escaped.
func ContainsPattern(s string) string { return "%" + EscapeLike(s) + "%" }

// PrefixPattern builds a "value%" pattern with wildcards escaped.
func PrefixPattern(s string) string { return EscapeLike(s) + "%" }
