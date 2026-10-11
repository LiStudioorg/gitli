// SPDX-License-Identifier: AGPL-3.0
package pkgsvc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/package"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schemaDDL = `
CREATE TABLE package_versions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	repo_id BIGINT NOT NULL,
	package_type TEXT NOT NULL,
	name TEXT NOT NULL,
	version TEXT NOT NULL,
	metadata TEXT NOT NULL,
	created_at DATETIME NOT NULL
);
CREATE TABLE package_blobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	digest TEXT NOT NULL UNIQUE,
	size_bytes BIGINT NOT NULL,
	stored_path TEXT NOT NULL,
	created_at DATETIME NOT NULL
);
CREATE TABLE package_files (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	version_id BIGINT NOT NULL,
	filename TEXT NOT NULL,
	blob_id BIGINT NOT NULL,
	created_at DATETIME NOT NULL
);`

func newTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schemaDDL); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	st := store.New(database)
	blobs := pkgstore.NewBlobStore(st, t.TempDir())
	return New(st, blobs), st
}

func TestPublishGetListVersions(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t)

	if _, err := svc.PublishVersion(ctx, 1, "bogus", "pkg", "1.0.0", ""); err == nil {
		t.Fatal("expected error for unsupported package type")
	}

	id, err := svc.PublishVersion(ctx, 7, "npm", "left-pad", "1.0.0", `{"license":"MIT"}`)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}
	if _, err := svc.PublishVersion(ctx, 7, "npm", "left-pad", "1.1.0", ""); err != nil {
		t.Fatalf("publish 2: %v", err)
	}

	got, err := svc.GetVersion(ctx, 7, "npm", "left-pad", "1.0.0")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != id || got.RepoID != 7 || got.PackageType != "npm" || got.Name != "left-pad" || got.Version != "1.0.0" || got.Metadata != `{"license":"MIT"}` {
		t.Fatalf("unexpected version: %+v", got)
	}

	_, err = svc.GetVersion(ctx, 7, "npm", "left-pad", "9.9.9")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	list, err := svc.ListVersions(ctx, 7, "npm", "left-pad")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].Version != "1.0.0" || list[1].Version != "1.1.0" {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestAddOpenFileRoundtrip(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService(t)

	id, err := svc.PublishVersion(ctx, 3, "generic", "tool", "2.0.0", "")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	content := []byte("streaming package body contents")
	digest, err := svc.AddFile(ctx, id, "tool-2.0.0.tar.gz", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("add file: %v", err)
	}
	want := sha256.Sum256(content)
	if digest != hex.EncodeToString(want[:]) {
		t.Fatalf("digest mismatch: %s", digest)
	}

	rc, size, err := svc.OpenFile(ctx, id, "tool-2.0.0.tar.gz")
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer func() { _ = rc.Close() }()
	if size != int64(len(content)) {
		t.Fatalf("size mismatch: got %d", size)
	}
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("roundtrip mismatch: %q", got)
	}

	var fileCount, blobCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM package_files WHERE version_id = ?`, id).Scan(&fileCount); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM package_blobs WHERE digest = ?`, digest).Scan(&blobCount); err != nil {
		t.Fatalf("count blobs: %v", err)
	}
	if fileCount != 1 || blobCount != 1 {
		t.Fatalf("unexpected counts: files=%d blobs=%d", fileCount, blobCount)
	}

	if _, _, err := svc.OpenFile(ctx, id, "missing.bin"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteVersion(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService(t)

	id, err := svc.PublishVersion(ctx, 1, "maven", "com.example", "1.0", "")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := svc.AddFile(ctx, id, "a.jar", bytes.NewReader([]byte("jar"))); err != nil {
		t.Fatalf("add file: %v", err)
	}
	if err := svc.DeleteVersion(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.GetVersion(ctx, 1, "maven", "com.example", "1.0"); err != ErrNotFound {
		t.Fatalf("expected version gone, got %v", err)
	}
	var files int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM package_files WHERE version_id = ?`, id).Scan(&files); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if files != 0 {
		t.Fatalf("expected file rows removed, got %d", files)
	}
}

func TestFormatHelpers(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"npm", NPMTarballURL("left-pad", "1.0.0"), "left-pad/-/left-pad-1.0.0.tgz"},
		{"go", GoModulePath("LiStudioorg", "gitli"), "LiStudioorg/gitli"},
		{"maven", MavenPath("com.example.app", "core", "1.2.3", "core-1.2.3.jar"), "com/example/app/core/1.2.3/core-1.2.3.jar"},
		{"generic", GenericPath("tool", "2.0.0", "tool.tar.gz"), "tool/2.0.0/tool.tar.gz"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestValidateVersion(t *testing.T) {
	for _, ok := range []string{"1.0.0", "v1.2.3", "2024.01"} {
		if err := ValidateVersion(ok); err != nil {
			t.Fatalf("expected %q valid, got %v", ok, err)
		}
	}
	for _, bad := range []string{"", "a/b", "../etc", "..", "1/2"} {
		if err := ValidateVersion(bad); err == nil {
			t.Fatalf("expected %q rejected", bad)
		}
	}
}
