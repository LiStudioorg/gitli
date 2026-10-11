// SPDX-License-Identifier: AGPL-3.0

package backup

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

// makeSource creates a temporary source tree with a few files and nested
// directories, returning its path and the expected contents keyed by
// slash-separated relative path.
func makeSource(t *testing.T) (string, map[string][]byte) {
	t.Helper()
	dir := t.TempDir()
	want := map[string][]byte{
		"a.txt":           []byte("hello world\n"),
		"nested/b.bin":    bytes.Repeat([]byte("x"), 200*1024),
		"nested/c/d.json": []byte(`{"k":1}`),
	}
	for rel, data := range want {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return dir, want
}

// assertRestored verifies that every expected file exists under dest with
// matching contents.
func assertRestored(t *testing.T, dest string, want map[string][]byte) {
	t.Helper()
	for rel, exp := range want {
		p := filepath.Join(dest, filepath.FromSlash(rel))
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if !bytes.Equal(got, exp) {
			t.Fatalf("content mismatch for %s: got %d bytes want %d", rel, len(got), len(exp))
		}
	}
}

func TestCreateRestorePlain(t *testing.T) {
	ctx := context.Background()
	src, want := makeSource(t)

	arc := filepath.Join(t.TempDir(), "plain.tar.gz")
	if err := Create(ctx, Options{SourceDir: src, OutputFile: arc}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "out")
	if err := Restore(ctx, arc, dest, nil); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	assertRestored(t, dest, want)
}

func TestCreateRestoreEncrypted(t *testing.T) {
	ctx := context.Background()
	src, want := makeSource(t)

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}

	arc := filepath.Join(t.TempDir(), "enc.bin")
	if err := Create(ctx, Options{SourceDir: src, OutputFile: arc, Encrypt: true, Key: key}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "out")
	if err := Restore(ctx, arc, dest, key); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	assertRestored(t, dest, want)
}

func TestRestoreEncryptedWrongKey(t *testing.T) {
	ctx := context.Background()
	src, _ := makeSource(t)

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}
	wrong := make([]byte, 32)
	if _, err := rand.Read(wrong); err != nil {
		t.Fatalf("rand: %v", err)
	}

	arc := filepath.Join(t.TempDir(), "enc.bin")
	if err := Create(ctx, Options{SourceDir: src, OutputFile: arc, Encrypt: true, Key: key}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "out")
	if err := Restore(ctx, arc, dest, wrong); err == nil {
		t.Fatal("expected error restoring with wrong key")
	}
}

func TestRestoreEncryptedTampered(t *testing.T) {
	ctx := context.Background()
	src, _ := makeSource(t)

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}

	arc := filepath.Join(t.TempDir(), "enc.bin")
	if err := Create(ctx, Options{SourceDir: src, OutputFile: arc, Encrypt: true, Key: key}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	data, err := os.ReadFile(arc)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	target := len(Magic) + 6
	data[target] ^= 0xff
	if err := os.WriteFile(arc, data, 0o600); err != nil {
		t.Fatalf("rewrite archive: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "out")
	if err := Restore(ctx, arc, dest, key); err == nil {
		t.Fatal("expected HMAC failure on tampered archive")
	}
}

func TestCreateRequiresKeyWhenEncrypting(t *testing.T) {
	src, _ := makeSource(t)
	arc := filepath.Join(t.TempDir(), "enc.bin")
	if err := Create(context.Background(), Options{SourceDir: src, OutputFile: arc, Encrypt: true}); err == nil {
		t.Fatal("expected error for missing key")
	}
}
