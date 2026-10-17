// SPDX-License-Identifier: AGPL-3.0
package ipallow

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE orgs (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, ip_allowlist_enabled INTEGER DEFAULT 0);
CREATE TABLE ip_allowlist (id INTEGER PRIMARY KEY AUTOINCREMENT, org_id INTEGER, cidr TEXT, description TEXT DEFAULT '', enabled INTEGER DEFAULT 1, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:ipallow?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO orgs (id, name) VALUES (1, 'acme')`); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestValidateCIDR(t *testing.T) {
	for _, ok := range []string{"192.168.1.0/24", "10.0.0.1", "::1", "2001:db8::/32"} {
		if err := ValidateCIDR(ok); err != nil {
			t.Fatalf("%s should be valid: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "not-an-ip", "999.1.1.1/24"} {
		if err := ValidateCIDR(bad); err == nil {
			t.Fatalf("%q should be invalid", bad)
		}
	}
}

func TestEnforcement(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	// Disabled -> always allowed.
	if ok, _ := s.Check(ctx, 1, "8.8.8.8"); !ok {
		t.Fatal("disabled allowlist should allow all")
	}
	if _, err := s.Add(ctx, 1, "192.168.1.0/24", "office", true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(ctx, 1, "203.0.113.5", "vpn", true); err != nil {
		t.Fatal(err)
	}
	if err := s.SetEnabled(ctx, 1, true); err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"192.168.1.42": true,
		"203.0.113.5":  true,
		"8.8.8.8":      false,
		"203.0.113.6":  false,
	}
	for ip, want := range cases {
		got, err := s.Check(ctx, 1, ip)
		if err != nil || got != want {
			t.Fatalf("Check(%s) = %v %v, want %v", ip, got, err, want)
		}
	}
}
