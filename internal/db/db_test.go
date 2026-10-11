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
	cases := []struct {
		in      string
		dialect Dialect
		driver  string
	}{
		{"", DialectSQLite, "sqlite"},
		{"sqlite", DialectSQLite, "sqlite"},
		{"sqlite3", DialectSQLite, "sqlite"},
		{"  SQLITE3 ", DialectSQLite, "sqlite"},
		{"mysql", DialectMySQL, "mysql"},
		{"postgres", DialectPostgres, "pgx"},
		{"postgresql", DialectPostgres, "pgx"},
		{"pgx", DialectPostgres, "pgx"},
		{"PGX", DialectPostgres, "pgx"},
	}
	for _, c := range cases {
		d, driver, err := resolveDriver(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if d != c.dialect || driver != c.driver {
			t.Fatalf("%q: got %s/%s, want %s/%s", c.in, d, driver, c.dialect, c.driver)
		}
	}
	if _, _, err := resolveDriver("oracle"); err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestNormalizeSQLiteDSN(t *testing.T) {
	const pragmas = "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty defaults to data path", "", "./data/gitli.db" + pragmas},
		{"plain path gets pragmas", "./data/gitli.db", "./data/gitli.db" + pragmas},
		{"path with query uses ampersand", "file:x.db?_fk=1", "file:x.db?_fk=1&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"},
		{"url without query gets pragmas", "file://x.db", "file://x.db" + pragmas},
		{"url with query untouched", "file://x.db?_pragma=cache(shared)", "file://x.db?_pragma=cache(shared)"},
	}
	for _, c := range cases {
		if got := normalizeSQLiteDSN(c.in); got != c.want {
			t.Fatalf("%s: normalizeSQLiteDSN(%q)=%q want %q", c.name, c.in, got, c.want)
		}
	}
}
