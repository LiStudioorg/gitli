// SPDX-License-Identifier: AGPL-3.0
package lfs

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const validOID = "d1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ddl := `CREATE TABLE lfs_objects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		oid VARCHAR(64) NOT NULL,
		size_bytes INTEGER NOT NULL DEFAULT 0,
		stored_path VARCHAR(1024) NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database), t.TempDir())
}

func TestVerifyOID(t *testing.T) {
	s := &Service{}
	if !s.VerifyOID(validOID) {
		t.Fatal("valid oid rejected")
	}
	bad := []string{
		"",
		"../x",
		"abc",
		strings.ToUpper(validOID),
		validOID + "0",
		strings.Repeat("g", 64),
	}
	for _, oid := range bad {
		if s.VerifyOID(oid) {
			t.Errorf("bad oid accepted: %q", oid)
		}
	}
}

func TestPath(t *testing.T) {
	root := t.TempDir()
	s := &Service{root: root}
	p, err := s.Path(validOID)
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	want := filepath.Join(root, validOID[:2], validOID[2:4], validOID)
	if p != want {
		t.Fatalf("path = %q want %q", p, want)
	}
	for _, oid := range []string{"../x", "abc", ""} {
		if _, err := s.Path(oid); err == nil {
			t.Errorf("path accepted bad oid %q", oid)
		}
	}
}

func TestPutGetRoundtrip(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	payload := bytes.Repeat([]byte("gitli-lfs-payload-"), 1024)

	if err := s.Put(ctx, 1, validOID, int64(len(payload)), bytes.NewReader(payload)); err != nil {
		t.Fatalf("put: %v", err)
	}
	if !s.Exists(validOID) {
		t.Fatal("object not on disk after put")
	}

	rc, size, err := s.Get(ctx, 1, validOID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = rc.Close() }()
	if size != int64(len(payload)) {
		t.Fatalf("size = %d want %d", size, len(payload))
	}
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("roundtrip payload mismatch")
	}

	if err := s.Put(ctx, 1, validOID, int64(len(payload)), bytes.NewReader(payload)); err != nil {
		t.Fatalf("idempotent put: %v", err)
	}

	if err := s.Delete(ctx, 1, validOID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if s.Exists(validOID) {
		t.Fatal("object still on disk after delete")
	}
	if _, _, err := s.Get(ctx, 1, validOID); err == nil {
		t.Fatal("get after delete should fail")
	}
}

func TestPutSizeMismatch(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	err := s.Put(ctx, 1, validOID, 10, strings.NewReader("short"))
	if err == nil {
		t.Fatal("expected size mismatch error")
	}
	if s.Exists(validOID) {
		t.Fatal("partial object left on disk after mismatch")
	}
}
