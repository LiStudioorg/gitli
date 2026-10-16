// SPDX-License-Identifier: AGPL-3.0
package registry

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	pkgstore "github.com/LiStudioorg/gitli/internal/service/package"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const uploadSchema = `
CREATE TABLE docker_manifests (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, name TEXT, digest TEXT, media_type TEXT, content TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE docker_tags (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, name TEXT, tag TEXT, manifest_id INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE docker_uploads (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, uuid TEXT, stored_path TEXT, received_bytes INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE package_blobs (id INTEGER PRIMARY KEY AUTOINCREMENT, digest TEXT UNIQUE, size_bytes INTEGER, stored_path TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
`

func TestUploadLifecycle(t *testing.T) {
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if _, err := database.Exec(uploadSchema); err != nil {
		t.Fatal(err)
	}
	st := store.New(database)
	root := t.TempDir()
	blobs := pkgstore.NewBlobStore(st, root)
	svc := New(st, blobs, nil)
	ctx := context.Background()

	staging := filepath.Join(root, "uploads")
	if err := os.MkdirAll(staging, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(staging, "blob")
	uuid, err := svc.StartUpload(ctx, 1, path)
	if err != nil {
		t.Fatalf("start upload: %v", err)
	}
	if got, err := svc.StagedPath(ctx, 1, uuid); err != nil || got != path {
		t.Fatalf("staged path = %q, %v", got, err)
	}
	payload := []byte("layer-bytes")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	// Use the digest the BlobStore itself computes so the assertion is stable.
	blobDigest, size, err := blobs.Put(ctx, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("blob put: %v", err)
	}
	if size != int64(len(payload)) {
		t.Fatalf("size = %d", size)
	}
	if err := svc.FinishUpload(ctx, 1, path, uuid, "sha256:"+blobDigest); err != nil {
		t.Fatalf("finish upload: %v", err)
	}
	if _, err := svc.StagedPath(ctx, 1, uuid); err == nil {
		t.Fatal("upload session should be removed after finish")
	}
	rc, err := svc.GetBlob("sha256:" + blobDigest)
	if err != nil {
		t.Fatalf("get blob: %v", err)
	}
	defer func() { _ = rc.Close() }()
	buf := make([]byte, len(payload))
	if _, err := rc.Read(buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != string(payload) {
		t.Fatalf("blob content = %q", buf)
	}
}
