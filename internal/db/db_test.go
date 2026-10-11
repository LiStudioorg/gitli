// SPDX-License-Identifier: AGPL-3.0
package db

import "testing"

func TestRebindPostgres(t *testing.T) {
	d := &DB{Dialect: DialectPostgres}
	got := d.Rebind("SELECT * FROM users WHERE id=? AND name=?")
	want := "SELECT * FROM users WHERE id=$1 AND name=$2"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRebindSQLiteNoop(t *testing.T) {
	d := &DB{Dialect: DialectSQLite}
	q := "SELECT * FROM users WHERE id=?"
	if d.Rebind(q) != q {
		t.Fatal("sqlite rebind should be a no-op")
	}
}

func TestResolveDriver(t *testing.T) {
	cases := map[string]Dialect{
		"":         DialectSQLite,
		"sqlite":   DialectSQLite,
		"mysql":    DialectMySQL,
		"postgres": DialectPostgres,
		"pgx":      DialectPostgres,
	}
	for in, want := range cases {
		d, _, err := resolveDriver(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if d != want {
			t.Fatalf("%s: got %s, want %s", in, d, want)
		}
	}
	if _, _, err := resolveDriver("oracle"); err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}
