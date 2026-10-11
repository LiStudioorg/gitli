// SPDX-License-Identifier: AGPL-3.0
// Package migrations embeds per-dialect SQL migration files and runs them.
package migrations

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/LiStudioorg/gitli/internal/db"
)

//go:embed sqlite/*.sql mysql/*.sql postgres/*.sql
var files embed.FS

// FS returns the embedded migration filesystem.
func FS() embed.FS { return files }

// DirFor returns the subdirectory for a given dialect.
func DirFor(d db.Dialect) (string, error) {
	switch d {
	case db.DialectSQLite:
		return "sqlite", nil
	case db.DialectMySQL:
		return "mysql", nil
	case db.DialectPostgres:
		return "postgres", nil
	default:
		return "", fmt.Errorf("no migrations for dialect %q", d)
	}
}

// All returns the sorted list of embedded migration paths for a dialect.
func All(d db.Dialect) ([]string, error) {
	dir, err := DirFor(d)
	if err != nil {
		return nil, err
	}
	entries, err := fs.Glob(files, dir+"/*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	return entries, nil
}
