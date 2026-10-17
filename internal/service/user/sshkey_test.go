// SPDX-License-Identifier: AGPL-3.0
package user

import (
	"context"
	"errors"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const keySchema = `
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, email TEXT, password_hash TEXT, full_name TEXT DEFAULT '', bio TEXT DEFAULT '', avatar_url TEXT DEFAULT '', website TEXT DEFAULT '', location TEXT DEFAULT '', is_admin INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1, totp_enabled INTEGER DEFAULT 0, totp_secret TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE ssh_keys (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, title TEXT, fingerprint TEXT UNIQUE, public_key TEXT, can_authenticate INTEGER DEFAULT 1, can_sign INTEGER DEFAULT 0, verified INTEGER DEFAULT 0, last_used_at DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE signing_keys (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, key_type TEXT DEFAULT 'gpg', title TEXT, fingerprint TEXT UNIQUE, public_key TEXT, source_key_id INTEGER, verified INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
`

const testSSHPub = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIxSM99xAJU7rC3M4/am9dGWvRzJtRiZbw+3UaYMiqac test@example"

func newTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(keySchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'alice', 'a@e.com', 'x')`); err != nil {
		t.Fatal(err)
	}
	st := store.New(database)
	return New(st), st
}

func TestAddSSHKeyRequiresPurpose(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	if _, err := s.AddSSHKey(ctx, 1, "k", testSSHPub, false, false); err == nil {
		t.Fatal("expected error when neither purpose is selected")
	}
}

// 回归测试：删除不属于自己的密钥必须返回 ErrNotFound，且不能删掉目标资源。
// 修复前只检查 SQL 错误，RowsAffected 为 0 时也返回 nil，越权请求会得到 200「成功」。
func TestDeleteKeyRejectsForeignOwner(t *testing.T) {
	s, st := newTestService(t)
	ctx := context.Background()
	if _, err := st.DB.ExecContext(ctx, `INSERT INTO users (id, username, email, password_hash) VALUES (2, 'bob', 'b@e.com', 'x')`); err != nil {
		t.Fatal(err)
	}
	k, err := s.AddSSHKey(ctx, 2, "bob-key", testSSHPub, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// alice(1) 试图删除 bob(2) 的密钥
	if err := s.DeleteSSHKey(ctx, 1, k.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign DeleteSSHKey: want ErrNotFound, got %v", err)
	}
	// 密钥必须还在
	var n int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM ssh_keys WHERE id = ?`, k.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("key must survive a cross-user delete, count = %d", n)
	}

	// 属主自己删除应当成功
	if err := s.DeleteSSHKey(ctx, 2, k.ID); err != nil {
		t.Fatalf("owner DeleteSSHKey: %v", err)
	}
	// 重复删除应当返回 ErrNotFound（此前会静默成功）
	if err := s.DeleteSSHKey(ctx, 2, k.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("repeat DeleteSSHKey: want ErrNotFound, got %v", err)
	}
}

func TestSSHKeyAuthPurpose(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	// auth-only key
	k, err := s.AddSSHKey(ctx, 1, "auth", testSSHPub, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserBySSHFingerprint(ctx, k.Fingerprint); err != nil {
		t.Fatalf("auth-only key should authenticate: %v", err)
	}
	// switch to sign-only
	if err := s.UpdateSSHKeyPurposes(ctx, 1, k.ID, false, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserBySSHFingerprint(ctx, k.Fingerprint); err == nil {
		t.Fatal("sign-only key must NOT authenticate")
	}
	if _, _, err := s.SigningKeyOwner(ctx, k.Fingerprint); err != nil {
		t.Fatalf("sign-only key should be a signing key: %v", err)
	}
}

func TestSigningKeysUnion(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	// SSH key that signs
	sshK, _ := s.AddSSHKey(ctx, 1, "sign", testSSHPub, false, true)
	// GPG key
	gpgK, err := s.AddSigningKey(ctx, 1, "gpg", "gpg1", "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nZm9v\n-----END PGP PUBLIC KEY BLOCK-----")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := s.SigningPublicKeys(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 signing keys (ssh+gpg), got %d", len(keys))
	}
	if _, kt, err := s.SigningKeyOwner(ctx, sshK.Fingerprint); err != nil || kt != "ssh" {
		t.Fatalf("ssh signing owner: %v type=%s", err, kt)
	}
	if _, kt, err := s.SigningKeyOwner(ctx, gpgK.Fingerprint); err != nil || kt != "gpg" {
		t.Fatalf("gpg signing owner: %v type=%s", err, kt)
	}
}

func TestUpdateRejectsNoPurpose(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	k, _ := s.AddSSHKey(ctx, 1, "k", testSSHPub, true, false)
	if err := s.UpdateSSHKeyPurposes(ctx, 1, k.ID, false, false); err == nil {
		t.Fatal("update to no-purpose should fail")
	}
}
