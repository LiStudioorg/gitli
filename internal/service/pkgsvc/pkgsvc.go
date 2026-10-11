// SPDX-License-Identifier: AGPL-3.0
// Package pkgsvc provides package registry services for the Generic, NPM,
// Go modules and Maven formats, layered on top of the content-addressed blob
// store.
package pkgsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/package"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing version or file.
var ErrNotFound = errors.New("pkgsvc: not found")

// ErrInvalidPackageType indicates an unsupported package format.
var ErrInvalidPackageType = errors.New("pkgsvc: invalid package type")

// validTypes is the set of supported package formats.
var validTypes = map[string]bool{
	"generic": true,
	"npm":     true,
	"go":      true,
	"maven":   true,
}

// Service manages package versions and their files.
type Service struct {
	st    *store.Store
	blobs *pkgstore.BlobStore
}

// New builds a package Service.
func New(st *store.Store, blobs *pkgstore.BlobStore) *Service {
	return &Service{st: st, blobs: blobs}
}

// Version is a single published package version.
type Version struct {
	ID          int64
	RepoID      int64
	PackageType string
	Name        string
	Version     string
	Metadata    string
}

// PublishVersion inserts a new package version and returns its id.
func (s *Service) PublishVersion(ctx context.Context, repoID int64, pkgType, name, version, metadata string) (int64, error) {
	if !validTypes[pkgType] {
		return 0, fmt.Errorf("%w: %q", ErrInvalidPackageType, pkgType)
	}
	q := s.st.Rebind(`INSERT INTO package_versions (repo_id, package_type, name, version, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, pkgType, name, version, metadata, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("pkgsvc: insert version: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("pkgsvc: version id: %w", err)
	}
	return id, nil
}

// GetVersion fetches a single version by repository, format, name and version.
func (s *Service) GetVersion(ctx context.Context, repoID int64, pkgType, name, version string) (*Version, error) {
	q := s.st.Rebind(`SELECT id, repo_id, package_type, name, version, metadata FROM package_versions WHERE repo_id = ? AND package_type = ? AND name = ? AND version = ?`)
	var v Version
	err := s.st.DB.QueryRowContext(ctx, q, repoID, pkgType, name, version).Scan(
		&v.ID, &v.RepoID, &v.PackageType, &v.Name, &v.Version, &v.Metadata,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("pkgsvc: get version: %w", err)
	}
	return &v, nil
}

// ListVersions returns every version of a package ordered by id.
func (s *Service) ListVersions(ctx context.Context, repoID int64, pkgType, name string) ([]Version, error) {
	q := s.st.Rebind(`SELECT id, repo_id, package_type, name, version, metadata FROM package_versions WHERE repo_id = ? AND package_type = ? AND name = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID, pkgType, name)
	if err != nil {
		return nil, fmt.Errorf("pkgsvc: list versions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Version
	for rows.Next() {
		var v Version
		if err := rows.Scan(&v.ID, &v.RepoID, &v.PackageType, &v.Name, &v.Version, &v.Metadata); err != nil {
			return nil, fmt.Errorf("pkgsvc: scan version: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pkgsvc: iterate versions: %w", err)
	}
	return out, nil
}

// AddFile streams body into the blob store and attaches it to versionID.
func (s *Service) AddFile(ctx context.Context, versionID int64, filename string, body io.Reader) (string, error) {
	digest, _, err := s.blobs.Put(ctx, body)
	if err != nil {
		return "", fmt.Errorf("pkgsvc: put blob: %w", err)
	}

	var blobID int64
	q := s.st.Rebind(`SELECT id FROM package_blobs WHERE digest = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, digest).Scan(&blobID); err != nil {
		return "", fmt.Errorf("pkgsvc: resolve blob id: %w", err)
	}

	q = s.st.Rebind(`INSERT INTO package_files (version_id, filename, blob_id, created_at) VALUES (?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, versionID, filename, blobID, time.Now().UTC()); err != nil {
		return "", fmt.Errorf("pkgsvc: insert file: %w", err)
	}
	return digest, nil
}

// OpenFile returns a reader and size for a file attached to versionID.
func (s *Service) OpenFile(ctx context.Context, versionID int64, filename string) (io.ReadCloser, int64, error) {
	q := s.st.Rebind(`SELECT b.digest, b.size_bytes FROM package_files f JOIN package_blobs b ON b.id = f.blob_id WHERE f.version_id = ? AND f.filename = ?`)
	var digest string
	var size int64
	err := s.st.DB.QueryRowContext(ctx, q, versionID, filename).Scan(&digest, &size)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("pkgsvc: lookup file: %w", err)
	}

	rc, err := s.blobs.Open(digest)
	if err != nil {
		return nil, 0, fmt.Errorf("pkgsvc: open blob: %w", err)
	}
	return rc, size, nil
}

// DeleteVersion removes a version and its file rows.
func (s *Service) DeleteVersion(ctx context.Context, versionID int64) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM package_files WHERE version_id = ?`), versionID); err != nil {
		return fmt.Errorf("pkgsvc: delete files: %w", err)
	}
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM package_versions WHERE id = ?`), versionID); err != nil {
		return fmt.Errorf("pkgsvc: delete version: %w", err)
	}
	return nil
}
