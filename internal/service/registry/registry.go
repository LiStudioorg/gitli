// SPDX-License-Identifier: AGPL-3.0
// Package registry implements a Docker / OCI Registry V2 content store.
package registry

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	pkgstore "github.com/LiStudioorg/gitli/internal/service/package"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing manifest, tag or blob.
var ErrNotFound = errors.New("registry: not found")

// ErrBadDigest indicates a malformed digest reference.
var ErrBadDigest = errors.New("registry: invalid digest")

// Media types accepted for manifest lists / image indexes.
const (
	MediaTypeOCIIndex           = "application/vnd.oci.image.index.v1+json"
	MediaTypeDockerManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
)

// Descriptor references a blob or nested manifest by digest.
type Descriptor struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

// Manifest is an OCI / Docker image manifest.
type Manifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	Config        Descriptor   `json:"config"`
	Layers        []Descriptor `json:"layers"`
}

// Service provides repository, blob and manifest operations for a registry.
type Service struct {
	st    *store.Store
	blobs *pkgstore.BlobStore
	log   *slog.Logger
}

// New builds a registry Service backed by the shared store and blob store.
func New(st *store.Store, blobs *pkgstore.BlobStore, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{st: st, blobs: blobs, log: log}
}

// ValidateDigest reports whether d is a well-formed "sha256:<64 hex>" digest.
func (s *Service) ValidateDigest(d string) bool {
	const prefix = "sha256:"
	if len(d) != len(prefix)+sha256.Size*2 || d[:len(prefix)] != prefix {
		return false
	}
	hexPart := d[len(prefix):]
	for i := 0; i < len(hexPart); i++ {
		c := hexPart[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}

// DigestOf returns the canonical "sha256:<hex>" digest of body.
func DigestOf(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// StagedPath returns the staged file path recorded for an upload session.
func (s *Service) StagedPath(ctx context.Context, repoID int64, uuid string) (string, error) {
	q := s.st.Rebind(`SELECT stored_path FROM docker_uploads WHERE repo_id = ? AND uuid = ?`)
	var path string
	if err := s.st.DB.QueryRowContext(ctx, q, repoID, uuid).Scan(&path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return path, nil
}

// StartUpload creates a blob upload session and returns its UUID.
func (s *Service) StartUpload(ctx context.Context, repoID int64, name string) (string, error) {
	if s.st == nil || s.st.DB == nil {
		return "", errors.New("registry: nil database")
	}
	uuid, err := newUUID()
	if err != nil {
		return "", err
	}
	q := s.st.Rebind(`INSERT INTO docker_uploads (repo_id, uuid, stored_path, received_bytes, created_at) VALUES (?, ?, ?, 0, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, repoID, uuid, name, time.Now().UTC()); err != nil {
		return "", fmt.Errorf("registry: insert upload: %w", err)
	}
	return uuid, nil
}

// FinishUpload commits a staged blob into the BlobStore and clears the upload.
func (s *Service) FinishUpload(ctx context.Context, repoID int64, name, uuid, digest string) error {
	if !s.ValidateDigest(digest) {
		return ErrBadDigest
	}
	q := s.st.Rebind(`SELECT stored_path FROM docker_uploads WHERE repo_id = ? AND uuid = ?`)
	var storedPath string
	if err := s.st.DB.QueryRowContext(ctx, q, repoID, uuid).Scan(&storedPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("registry: load upload: %w", err)
	}
	if err := s.commitStaged(ctx, storedPath, digest); err != nil {
		return err
	}
	del := s.st.Rebind(`DELETE FROM docker_uploads WHERE repo_id = ? AND uuid = ?`)
	if _, err := s.st.DB.ExecContext(ctx, del, repoID, uuid); err != nil {
		return fmt.Errorf("registry: delete upload: %w", err)
	}
	return nil
}

// commitStaged streams a staged file through the BlobStore, verifying digest.
func (s *Service) commitStaged(ctx context.Context, path, digest string) error {
	if s.blobs == nil {
		return errors.New("registry: nil blob store")
	}
	f, err := openStaged(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	got, _, err := s.blobs.Put(ctx, f)
	if err != nil {
		return fmt.Errorf("registry: commit blob: %w", err)
	}
	if got != digest {
		return fmt.Errorf("registry: digest mismatch: got sha256:%s", got)
	}
	return nil
}

// PutManifest stores a manifest body and points the reference at it.
//
// reference is treated as a tag unless it is a digest reference, in which case
// no tag row is written. Docker and OCI image manifests and manifest lists are
// all accepted here.
func (s *Service) PutManifest(ctx context.Context, repoID int64, name, reference, mediaType string, body []byte) (string, error) {
	if s.st == nil || s.st.DB == nil {
		return "", errors.New("registry: nil database")
	}
	if reference == "" {
		return "", errors.New("registry: empty reference")
	}
	digest := DigestOf(body)

	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("registry: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	insManifest := s.st.Rebind(`INSERT INTO docker_manifests (repo_id, name, digest, media_type, content, created_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, insManifest, repoID, name, digest, mediaType, string(body), time.Now().UTC()); err != nil {
		return "", fmt.Errorf("registry: insert manifest: %w", err)
	}

	if !s.ValidateDigest(reference) {
		upTag := s.st.Rebind(`INSERT INTO docker_tags (repo_id, name, tag, manifest_id, created_at) VALUES (?, ?, ?, ?, ?)`)
		if _, err := tx.ExecContext(ctx, upTag, repoID, name, reference, digest, time.Now().UTC()); err != nil {
			return "", fmt.Errorf("registry: upsert tag: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("registry: commit manifest: %w", err)
	}
	return digest, nil
}

// PutManifestList stores a manifest list / OCI image index.
func (s *Service) PutManifestList(ctx context.Context, repoID int64, name, reference, mediaType string, body []byte) (string, error) {
	switch mediaType {
	case MediaTypeOCIIndex, MediaTypeDockerManifestList:
	default:
		return "", fmt.Errorf("registry: unsupported manifest list media type %q", mediaType)
	}
	return s.PutManifest(ctx, repoID, name, reference, mediaType, body)
}

// GetManifest resolves reference (a tag or a digest) to its raw manifest.
func (s *Service) GetManifest(ctx context.Context, repoID int64, name, reference string) (content []byte, mediaType, digest string, err error) {
	if s.st == nil || s.st.DB == nil {
		return nil, "", "", errors.New("registry: nil database")
	}
	if s.ValidateDigest(reference) {
		q := s.st.Rebind(`SELECT content, media_type, digest FROM docker_manifests WHERE repo_id = ? AND name = ? AND digest = ? ORDER BY id DESC LIMIT 1`)
		row := s.st.DB.QueryRowContext(ctx, q, repoID, name, reference)
		if err := row.Scan(&content, &mediaType, &digest); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, "", "", ErrNotFound
			}
			return nil, "", "", fmt.Errorf("registry: get manifest by digest: %w", err)
		}
		return content, mediaType, digest, nil
	}

	q := s.st.Rebind(`SELECT m.content, m.media_type, m.digest FROM docker_tags t JOIN docker_manifests m ON m.digest = t.manifest_id WHERE t.repo_id = ? AND t.name = ? AND t.tag = ? ORDER BY t.id DESC LIMIT 1`)
	row := s.st.DB.QueryRowContext(ctx, q, repoID, name, reference)
	if err := row.Scan(&content, &mediaType, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", "", ErrNotFound
		}
		return nil, "", "", fmt.Errorf("registry: get manifest by tag: %w", err)
	}
	return content, mediaType, digest, nil
}

// ListTags returns the tags of a repository, sorted ascending.
func (s *Service) ListTags(ctx context.Context, repoID int64, name string) ([]string, error) {
	if s.st == nil || s.st.DB == nil {
		return nil, errors.New("registry: nil database")
	}
	q := s.st.Rebind(`SELECT tag FROM docker_tags WHERE repo_id = ? AND name = ? ORDER BY tag ASC`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID, name)
	if err != nil {
		return nil, fmt.Errorf("registry: list tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tags := make([]string, 0, 8)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("registry: scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registry: iterate tags: %w", err)
	}
	return tags, nil
}

// DeleteManifest removes a manifest and any tags pointing at its digest.
func (s *Service) DeleteManifest(ctx context.Context, repoID int64, name, digest string) error {
	if !s.ValidateDigest(digest) {
		return ErrBadDigest
	}
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("registry: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	delTags := s.st.Rebind(`DELETE FROM docker_tags WHERE repo_id = ? AND name = ? AND manifest_id = ?`)
	if _, err := tx.ExecContext(ctx, delTags, repoID, name, digest); err != nil {
		return fmt.Errorf("registry: delete tags: %w", err)
	}
	delManifest := s.st.Rebind(`DELETE FROM docker_manifests WHERE repo_id = ? AND name = ? AND digest = ?`)
	res, err := tx.ExecContext(ctx, delManifest, repoID, name, digest)
	if err != nil {
		return fmt.Errorf("registry: delete manifest: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("registry: commit delete: %w", err)
	}
	return nil
}

// GetBlob opens the content-addressed blob identified by digest.
func (s *Service) GetBlob(digest string) (io.ReadCloser, error) {
	if s.blobs == nil {
		return nil, errors.New("registry: nil blob store")
	}
	return s.blobs.Open(digest)
}

// newUUID returns a random RFC 4122 version 4 UUID string.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("registry: read random: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
