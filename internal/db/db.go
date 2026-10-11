// SPDX-License-Identifier: AGPL-3.0
package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/LiStudioorg/gitli/internal/config"
)

// Dialect identifies the active SQL dialect.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectMySQL    Dialect = "mysql"
	DialectPostgres Dialect = "postgres"
)

// DB wraps a database handle with its dialect metadata.
type DB struct {
	*sql.DB
	Dialect Dialect
}

// Open connects to the database described by cfg and verifies connectivity.
func Open(cfg *config.DatabaseConfig) (*DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config is nil")
	}
	dialect, driverName, err := resolveDriver(cfg.Driver)
	if err != nil {
		return nil, err
	}
	dsn := cfg.DSN
	if dialect == DialectSQLite {
		dsn = normalizeSQLiteDSN(dsn)
	}
	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", driverName, err)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping %s: %w", driverName, err)
	}
	return &DB{DB: sqlDB, Dialect: dialect}, nil
}

func resolveDriver(name string) (Dialect, string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "sqlite", "sqlite3":
		return DialectSQLite, "sqlite", nil
	case "mysql":
		return DialectMySQL, "mysql", nil
	case "postgres", "postgresql", "pgx":
		return DialectPostgres, "pgx", nil
	default:
		return "", "", fmt.Errorf("unsupported database driver: %q", name)
	}
}

// normalizeSQLiteDSN keeps modernc.org/sqlite usable with plain file paths
// by appending pragmas that improve robustness for a multi-connection pool.
func normalizeSQLiteDSN(dsn string) string {
	if dsn == "" {
		dsn = "./data/gitli.db"
	}
	if strings.Contains(dsn, "://") {
		if strings.Contains(dsn, "?") {
			return dsn
		}
		return dsn + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
}

// Rebind converts '?' placeholders to the dialect's native style.
func (d *DB) Rebind(query string) string {
	if d.Dialect != DialectPostgres {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(fmt.Sprintf("%d", n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}
