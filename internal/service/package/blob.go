// SPDX-License-Identifier: AGPL-3.0
// Package pkgstore provides package registry services.
//
// It lives in the internal/service/package directory; the package clause is
// named pkgstore because "package" is a reserved Go keyword.
package pkgstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// digestHexLen is the length of a lowercase SHA-256 hex digest.
const digestHexLen = sha256.Size * 2

// BlobStore stores package blobs on disk addressed by their SHA-256 digest.
//
// Blobs are content-addressed: identical content is stored exactly once and
// shared by every package that references the same digest. I/O is streamed so
// arbitrarily large artifacts never need to be buffered in memory.
type BlobStore struct {
	st   *store.Store
	root string
}

// NewBlobStore builds a BlobStore rooted at root using st for metadata.
func NewBlobStore(st *store.Store, root string) *BlobStore {
	return &BlobStore{st: st, root: root}
}

// Put streams r into the blob store and returns its SHA-256 digest and size.
//
// The reader is copied through a SHA-256 hasher into a temporary file, then the
// file is renamed into its content-addressed path. If the digest already exists
// the metadata insert is a no-op and the existing blob is reused.
func (b *BlobStore) Put(ctx context.Context, r io.Reader) (digest string, size int64, err error) {
	if b.st == nil || b.st.DB == nil {
		return "", 0, errors.New("pkgstore: blob store: nil database")
	}
	if r == nil {
		return "", 0, errors.New("pkgstore: blob store: nil reader")
	}
	if err := os.MkdirAll(b.root, 0o755); err != nil {
		return "", 0, fmt.Errorf("pkgstore: mkdir blob root: %w", err)
	}

	tmp, err := os.CreateTemp(b.root, "blob-*.tmp")
	if err != nil {
		return "", 0, fmt.Errorf("pkgstore: create temp blob: %w", err)
	}
	tmpName := tmp.Name()

	hasher := sha256.New()
	size, err = io.Copy(io.MultiWriter(tmp, hasher), r)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("pkgstore: stream blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("pkgstore: close temp blob: %w", err)
	}

	digest = hex.EncodeToString(hasher.Sum(nil))
	storedPath := filepath.Join(b.root, digest[:2], digest[2:4], digest)
	if err := os.MkdirAll(filepath.Dir(storedPath), 0o755); err != nil {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("pkgstore: mkdir blob shard: %w", err)
	}
	if _, err := os.Stat(storedPath); err == nil {
		_ = os.Remove(tmpName)
	} else if err := os.Rename(tmpName, storedPath); err != nil {
		_ = os.Remove(tmpName)
		return "", 0, fmt.Errorf("pkgstore: store blob: %w", err)
	}

	q := b.st.Rebind(`INSERT INTO package_blobs (digest, size_bytes, stored_path, created_at) VALUES (?, ?, ?, ?) ON CONFLICT (digest) DO NOTHING`)
	if _, err := b.st.DB.ExecContext(ctx, q, digest, size, storedPath, time.Now().UTC()); err != nil {
		return "", 0, fmt.Errorf("pkgstore: insert blob metadata: %w", err)
	}
	return digest, size, nil
}

// Open returns a reader over the stored blob identified by digest.
func (b *BlobStore) Open(digest string) (io.ReadCloser, error) {
	if err := validateDigest(digest); err != nil {
		return nil, err
	}
	f, err := os.Open(b.path(digest))
	if err != nil {
		return nil, fmt.Errorf("pkgstore: open blob: %w", err)
	}
	return f, nil
}

// Verify reports whether the stored blob still matches its digest.
func (b *BlobStore) Verify(digest string) (bool, error) {
	if err := validateDigest(digest); err != nil {
		return false, err
	}
	f, err := os.Open(b.path(digest))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("pkgstore: open blob: %w", err)
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false, fmt.Errorf("pkgstore: hash blob: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)) == digest, nil
}

// Delete removes the blob file and its metadata row.
func (b *BlobStore) Delete(ctx context.Context, digest string) error {
	if err := validateDigest(digest); err != nil {
		return err
	}
	if err := os.Remove(b.path(digest)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("pkgstore: remove blob: %w", err)
	}
	if b.st != nil && b.st.DB != nil {
		if _, err := b.st.DB.ExecContext(ctx, b.st.Rebind(`DELETE FROM package_blobs WHERE digest = ?`), digest); err != nil {
			return fmt.Errorf("pkgstore: delete blob metadata: %w", err)
		}
	}
	return nil
}

// path returns the sharded on-disk location for digest.
func (b *BlobStore) path(digest string) string {
	return filepath.Join(b.root, digest[:2], digest[2:4], digest)
}

// validateDigest reports whether digest is a lowercase 64-character SHA-256 hex string.
func validateDigest(digest string) error {
	if len(digest) != digestHexLen {
		return fmt.Errorf("pkgstore: invalid digest length %d", len(digest))
	}
	for i := 0; i < len(digest); i++ {
		c := digest[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return errors.New("pkgstore: digest must be lowercase hex")
		}
	}
	_, err := hex.DecodeString(digest)
	if err != nil {
		return fmt.Errorf("pkgstore: invalid digest: %w", err)
	}
	return nil
}
