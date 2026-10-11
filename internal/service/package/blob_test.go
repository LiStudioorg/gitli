// SPDX-License-Identifier: AGPL-3.0
package pkgstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const packageBlobsDDL = `CREATE TABLE package_blobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	digest TEXT NOT NULL UNIQUE,
	size_bytes BIGINT NOT NULL,
	stored_path TEXT NOT NULL,
	created_at DATETIME NOT NULL
)`

func newTestBlobStore(t *testing.T) *BlobStore {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(packageBlobsDDL); err != nil {
		t.Fatalf("create table: %v", err)
	}
	st := store.New(database)
	return NewBlobStore(st, t.TempDir())
}

func TestValidateDigest(t *testing.T) {
	sum := sha256.Sum256([]byte("gitli"))
	valid := hex.EncodeToString(sum[:])
	if err := validateDigest(valid); err != nil {
		t.Fatalf("expected valid digest, got %v", err)
	}
	reject := []string{
		"",
		"abc",
		strings.ToUpper(valid),
		valid + "0",
		strings.Repeat("z", digestHexLen),
	}
	for _, in := range reject {
		if err := validateDigest(in); err == nil {
			t.Fatalf("expected reject for %q", in)
		}
	}
}

func TestPutStreamingDedupRoundtrip(t *testing.T) {
	ctx := context.Background()
	b := newTestBlobStore(t)

	content := []byte("hello package blob store, stream me please")
	digest, size, err := b.Put(ctx, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	want := sha256.Sum256(content)
	if digest != hex.EncodeToString(want[:]) {
		t.Fatalf("digest mismatch: got %s", digest)
	}
	if size != int64(len(content)) {
		t.Fatalf("size mismatch: got %d want %d", size, len(content))
	}

	rc, err := b.Open(digest)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	got, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("roundtrip mismatch: got %q", got)
	}

	ok, err := b.Verify(digest)
	if err != nil || !ok {
		t.Fatalf("verify: ok=%v err=%v", ok, err)
	}

	digest2, size2, err := b.Put(ctx, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("put duplicate: %v", err)
	}
	if digest2 != digest || size2 != size {
		t.Fatalf("dedup mismatch: %s/%d", digest2, size2)
	}

	var count int
	if err := b.st.DB.QueryRow(`SELECT COUNT(*) FROM package_blobs WHERE digest = ?`, digest).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 metadata row, got %d", count)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	b := newTestBlobStore(t)

	digest, _, err := b.Put(ctx, bytes.NewReader([]byte("delete me")))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := b.Delete(ctx, digest); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if ok, err := b.Verify(digest); err != nil || ok {
		t.Fatalf("verify after delete: ok=%v err=%v", ok, err)
	}
	if _, err := b.Open(digest); err == nil {
		t.Fatal("expected open to fail after delete")
	}
}
