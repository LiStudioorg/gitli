package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open 打开 SQLite 数据库（WAL 模式）并执行内嵌迁移。
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	// SQLite 单写多读，限制连接数避免 busy 冲突
	conn.SetMaxOpenConns(1)
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := migrateUp(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func migrateUp(conn *sql.DB) error {
	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrate source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	// 注意：不要调用 m.Close()——它会关闭底层 *sql.DB（Migrate.Close 会调用 database.Close）。
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// NewQuerier 返回 sqlc 生成的 Queries。
func NewQuerier(conn *sql.DB) *Queries {
	return New(conn)
}
