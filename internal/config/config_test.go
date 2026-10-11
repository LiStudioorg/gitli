// SPDX-License-Identifier: AGPL-3.0
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("unexpected default driver: %s", cfg.Database.Driver)
	}
	if cfg.Security.Password.MemoryMB != 64 {
		t.Fatalf("unexpected default memory: %d", cfg.Security.Password.MemoryMB)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[database]
driver = "mysql"
dsn = "user@tcp(localhost)/gitli"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Driver != "mysql" {
		t.Fatalf("got %s, want mysql", cfg.Database.Driver)
	}
	if len(cfg.Server.HTTPListen) == 0 {
		t.Fatalf("default http_listen should be preserved")
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("GITLI_DATABASE_DSN", "override.db")
	t.Setenv("GITLI_PEPPER", "envpepper")
	cfg, err := Load(filepath.Join(t.TempDir(), "x.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "override.db" {
		t.Fatalf("env override failed: %s", cfg.Database.DSN)
	}
	if cfg.Security.Password.Pepper != "envpepper" {
		t.Fatalf("pepper override failed: %s", cfg.Security.Password.Pepper)
	}
}
