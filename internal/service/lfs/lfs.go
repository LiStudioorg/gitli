// SPDX-License-Identifier: AGPL-3.0
// Package lfs stores large objects on disk keyed by their SHA-256 OID.
package lfs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing object on disk or in the database.
var ErrNotFound = errors.New("lfs: not found")

// ErrBadOID indicates a malformed (non SHA-256 hex) object id.
var ErrBadOID = errors.New("lfs: invalid oid")

// Service stores large objects under root using streaming I/O.
type Service struct {
	st   *store.Store
	root string
}

// New builds an LFS Service rooted at root.
func New(st *store.Store, root string) *Service {
	return &Service{st: st, root: root}
}

// BatchObject is a single entry of an LFS batch response.
type BatchObject struct {
	Oid    string
	Size   int64
	Action string
	Href   string
}

// VerifyOID reports whether oid is a 64-character lowercase hex SHA-256.
func (s *Service) VerifyOID(oid string) bool {
	if len(oid) != 64 {
		return false
	}
	for i := 0; i < len(oid); i++ {
		c := oid[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}

// Path returns the sharded on-disk path for oid.
func (s *Service) Path(oid string) (string, error) {
	if !s.VerifyOID(oid) {
		return "", ErrBadOID
	}
	return filepath.Join(s.root, oid[:2], oid[2:4], oid), nil
}

// Exists reports whether the object file is present on disk.
func (s *Service) Exists(oid string) bool {
	p, err := s.Path(oid)
	if err != nil {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// Put streams r to disk and records the object, verifying the written size.
func (s *Service) Put(ctx context.Context, repoID int64, oid string, size int64, r io.Reader) error {
	p, err := s.Path(oid)
	if err != nil {
		return err
	}
	if !s.Exists(oid) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return fmt.Errorf("lfs: mkdir: %w", err)
		}
		tmp, err := os.CreateTemp(filepath.Dir(p), oid+".tmp-*")
		if err != nil {
			return fmt.Errorf("lfs: temp: %w", err)
		}
		tmpName := tmp.Name()
		written, cerr := io.Copy(tmp, r)
		if cerr == nil && written != size {
			cerr = fmt.Errorf("lfs: size mismatch: got %d want %d", written, size)
		}
		if cerr == nil {
			cerr = tmp.Sync()
		}
		if closeErr := tmp.Close(); cerr == nil {
			cerr = closeErr
		}
		if cerr != nil {
			_ = os.Remove(tmpName)
			return cerr
		}
		if written != size {
			_ = os.Remove(tmpName)
			return fmt.Errorf("lfs: size mismatch: got %d want %d", written, size)
		}
		if err := os.Rename(tmpName, p); err != nil {
			_ = os.Remove(tmpName)
			return fmt.Errorf("lfs: rename: %w", err)
		}
	}
	q := s.st.Rebind(`INSERT INTO lfs_objects (repo_id, oid, size_bytes, stored_path) VALUES (?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, repoID, oid, size, p); err != nil {
		return fmt.Errorf("lfs: insert: %w", err)
	}
	return nil
}

// Get opens the object for streaming and returns its recorded size.
func (s *Service) Get(ctx context.Context, repoID int64, oid string) (io.ReadCloser, int64, error) {
	if !s.VerifyOID(oid) {
		return nil, 0, ErrBadOID
	}
	q := s.st.Rebind(`SELECT size_bytes FROM lfs_objects WHERE repo_id = ? AND oid = ?`)
	var size int64
	err := s.st.DB.QueryRowContext(ctx, q, repoID, oid).Scan(&size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("lfs: query: %w", err)
	}
	p, err := s.Path(oid)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, fmt.Errorf("lfs: open: %w", err)
	}
	return f, size, nil
}

// Delete removes the object file and its database row.
func (s *Service) Delete(ctx context.Context, repoID int64, oid string) error {
	p, err := s.Path(oid)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("lfs: remove: %w", err)
	}
	q := s.st.Rebind(`DELETE FROM lfs_objects WHERE repo_id = ? AND oid = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, repoID, oid); err != nil {
		return fmt.Errorf("lfs: delete: %w", err)
	}
	return nil
}
