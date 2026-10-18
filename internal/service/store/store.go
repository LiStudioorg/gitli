// SPDX-License-Identifier: AGPL-3.0
// Package store provides shared database access for the service layer.
package store

import (
	"context"
	"fmt"
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

// ExecAffected runs a write guarded by ownership predicates (`WHERE id = ?
// AND user_id = ?`), and reports how many rows it actually touched.
//
// 归属类写操作必须区分「改到了」和「什么都没匹配」：如果调用方只检查 err，
// 越权或已删除的请求会得到 200「成功」，既误导用户，也让越权探测失去信号。
// 这里把 RowsAffected 一并返回，由 service 转成各自的 ErrNotFound。
func (s *Store) ExecAffected(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: rows affected: %w", err)
	}
	return n, nil
}

// ExecAffectedOr runs ExecAffected and returns notFound when no row matched.
func (s *Store) ExecAffectedOr(ctx context.Context, notFound error, query string, args ...any) error {
	n, err := s.ExecAffected(ctx, query, args...)
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}
	return nil
}

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
