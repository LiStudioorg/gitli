// SPDX-License-Identifier: AGPL-3.0
package cisecret

import (
	"context"
	"errors"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/crypto"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

func newTestService(t *testing.T, cipher *crypto.Cipher) (*Service, *db.DB) {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`CREATE TABLE ci_secrets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		name VARCHAR(255) NOT NULL,
		ciphertext TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	return New(store.New(database), cipher), database
}

func testCipher(t *testing.T) *crypto.Cipher {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	c, err := crypto.New(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	return c
}

func TestSetListGetDelete(t *testing.T) {
	cases := []struct {
		name   string
		cipher *crypto.Cipher
	}{
		{"with-cipher", testCipher(t)},
		{"nil-cipher", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, database := newTestService(t, tc.cipher)
			ctx := context.Background()

			if err := s.Set(ctx, 7, "TOKEN", "s3cr3t"); err != nil {
				t.Fatalf("set: %v", err)
			}
			if err := s.Set(ctx, 7, "OTHER", "value2"); err != nil {
				t.Fatalf("set other: %v", err)
			}
			// Upsert same name, new value.
			if err := s.Set(ctx, 7, "TOKEN", "rotated"); err != nil {
				t.Fatalf("upsert: %v", err)
			}
			// Same name in another repo must not collide.
			if err := s.Set(ctx, 8, "TOKEN", "other-repo"); err != nil {
				t.Fatalf("set other repo: %v", err)
			}

			names, err := s.List(ctx, 7)
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(names) != 2 {
				t.Fatalf("want 2 names, got %d: %v", len(names), names)
			}
			for _, n := range names {
				if n == "rotated" || n == "s3cr3t" || n == "value2" {
					t.Fatalf("List leaked a value: %q", n)
				}
			}

			got, err := s.Get(ctx, 7, "TOKEN")
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if got != "rotated" {
				t.Fatalf("want rotated, got %q", got)
			}

			if tc.cipher != nil {
				var stored string
				if err := database.QueryRow(`SELECT ciphertext FROM ci_secrets WHERE repo_id = 7 AND name = 'TOKEN'`).Scan(&stored); err != nil {
					t.Fatalf("read raw: %v", err)
				}
				if stored == "rotated" || stored[:4] != encPrefix {
					t.Fatalf("value not encrypted at rest: %q", stored)
				}
			}

			if err := s.Delete(ctx, 7, "TOKEN"); err != nil {
				t.Fatalf("delete: %v", err)
			}
			if _, err := s.Get(ctx, 7, "TOKEN"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("want ErrNotFound after delete, got %v", err)
			}
			if err := s.Delete(ctx, 7, "TOKEN"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("want ErrNotFound deleting missing, got %v", err)
			}

			// Cross-repo isolation.
			if v, err := s.Get(ctx, 8, "TOKEN"); err != nil || v != "other-repo" {
				t.Fatalf("repo 8 secret wrong: v=%q err=%v", v, err)
			}
		})
	}
}

func TestGetMissing(t *testing.T) {
	s, _ := newTestService(t, testCipher(t))
	if _, err := s.Get(context.Background(), 1, "NOPE"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
