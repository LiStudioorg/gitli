// SPDX-License-Identifier: AGPL-3.0
package registry

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	pkgstore "github.com/LiStudioorg/gitli/internal/service/package"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const testDDL = `
CREATE TABLE docker_manifests (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	repo_id INTEGER,
	name TEXT,
	digest TEXT,
	media_type TEXT,
	content TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE docker_tags (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	repo_id INTEGER,
	name TEXT,
	tag TEXT,
	manifest_id INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE docker_uploads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	repo_id INTEGER,
	uuid TEXT,
	stored_path TEXT,
	received_bytes INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE package_blobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	digest TEXT UNIQUE,
	size_bytes INTEGER,
	stored_path TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(testDDL); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	st := store.New(database)
	blobs := pkgstore.NewBlobStore(st, t.TempDir())
	return New(st, blobs, nil)
}

func TestDigestOf(t *testing.T) {
	s := newTestService(t)
	got := DigestOf([]byte("gitli"))
	if !s.ValidateDigest(got) {
		t.Fatalf("DigestOf produced invalid digest: %s", got)
	}
	if got != DigestOf([]byte("gitli")) {
		t.Fatal("DigestOf is not stable")
	}
	if got == DigestOf([]byte("gitli!")) {
		t.Fatal("DigestOf collision on different content")
	}
}

func TestValidateDigest(t *testing.T) {
	s := newTestService(t)
	if !s.ValidateDigest(DigestOf([]byte("x"))) {
		t.Fatal("expected valid digest")
	}
	bad := []string{
		"",
		"sha256:",
		"sha256:abc",
		"md5:0123456789abcdef0123456789abcdef",
		"sha256:0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	for _, d := range bad {
		if s.ValidateDigest(d) {
			t.Fatalf("expected invalid digest: %q", d)
		}
	}
}

func TestPutAndGetManifest(t *testing.T) {
	ctx := context.Background()
	s := newTestService(t)

	body := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)
	mt := "application/vnd.oci.image.manifest.v1+json"

	digest, err := s.PutManifest(ctx, 1, "library/alpine", "latest", mt, body)
	if err != nil {
		t.Fatalf("put manifest: %v", err)
	}
	if digest != DigestOf(body) {
		t.Fatalf("digest mismatch: %s", digest)
	}

	byTag, gotMT, gotD, err := s.GetManifest(ctx, 1, "library/alpine", "latest")
	if err != nil {
		t.Fatalf("get by tag: %v", err)
	}
	if string(byTag) != string(body) || gotMT != mt || gotD != digest {
		t.Fatalf("tag lookup mismatch: mt=%s d=%s body=%s", gotMT, gotD, byTag)
	}

	byDigest, _, _, err := s.GetManifest(ctx, 1, "library/alpine", digest)
	if err != nil {
		t.Fatalf("get by digest: %v", err)
	}
	if string(byDigest) != string(body) {
		t.Fatalf("digest lookup mismatch: %s", byDigest)
	}

	tags, err := s.ListTags(ctx, 1, "library/alpine")
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 1 || tags[0] != "latest" {
		t.Fatalf("unexpected tags: %v", tags)
	}
}

func TestManifestListAndDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestService(t)

	body := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json"}`)
	digest, err := s.PutManifestList(ctx, 2, "library/busybox", "stable", MediaTypeOCIIndex, body)
	if err != nil {
		t.Fatalf("put manifest list: %v", err)
	}
	if _, err := s.PutManifestList(ctx, 2, "library/busybox", "bad", "application/json", body); err == nil {
		t.Fatal("expected error for unsupported manifest list media type")
	}
	if err := s.DeleteManifest(ctx, 2, "library/busybox", digest); err != nil {
		t.Fatalf("delete manifest: %v", err)
	}
	if _, _, _, err := s.GetManifest(ctx, 2, "library/busybox", "stable"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
