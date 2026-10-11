// SPDX-License-Identifier: AGPL-3.0
package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// Migration is a single versioned SQL script.
type Migration struct {
	Version string
	Name    string
	UpSQL   string
}

// ApplyMigrations runs all provided migrations in filename order inside a
// schema_migrations ledger so re-runs are idempotent.
func (d *DB) ApplyMigrations(ctx context.Context, migs []Migration) error {
	if err := d.ensureLedger(ctx); err != nil {
		return err
	}
	applied, err := d.appliedVersions(ctx)
	if err != nil {
		return err
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].Version < migs[j].Version })
	for _, m := range migs {
		if applied[m.Version] {
			continue
		}
		if err := d.applyOne(ctx, m); err != nil {
			return fmt.Errorf("migration %s: %w", m.Version, err)
		}
	}
	return nil
}

func (d *DB) applyOne(ctx context.Context, m Migration) error {
	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range splitStatements(m.UpSQL) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec: %w", err)
		}
	}
	insert := d.Rebind("INSERT INTO schema_migrations (version, name) VALUES (?, ?)")
	if _, err := tx.ExecContext(ctx, insert, m.Version, m.Name); err != nil {
		return fmt.Errorf("record: %w", err)
	}
	return tx.Commit()
}

func (d *DB) ensureLedger(ctx context.Context) error {
	stmt := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(64) NOT NULL PRIMARY KEY,
		name VARCHAR(255) NOT NULL DEFAULT '',
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if d.Dialect == DialectPostgres {
		stmt = `CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(64) NOT NULL PRIMARY KEY,
			name VARCHAR(255) NOT NULL DEFAULT '',
			applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`
	}
	_, err := d.ExecContext(ctx, stmt)
	return err
}

func (d *DB) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := d.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

// LoadMigrations reads embedded .sql files into Migration values.
func LoadMigrations(fsys fs.FS, paths []string) ([]Migration, error) {
	out := make([]Migration, 0, len(paths))
	for _, p := range paths {
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil, err
		}
		base := p[strings.LastIndex(p, "/")+1:]
		version := base
		name := base
		if i := strings.IndexByte(base, '_'); i >= 0 {
			version = base[:i]
			name = base[i+1:]
		}
		version = strings.TrimSuffix(version, ".sql")
		name = strings.TrimSuffix(name, ".sql")
		out = append(out, Migration{Version: version, Name: name, UpSQL: string(b)})
	}
	return out, nil
}

func splitStatements(sqlText string) []string {
	parts := strings.Split(sqlText, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" || strings.HasPrefix(s, "--") {
			continue
		}
		out = append(out, s)
	}
	return out
}
