// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const archSchema = `
CREATE TABLE repos (id INTEGER PRIMARY KEY AUTOINCREMENT, owner_id INTEGER, owner_type TEXT DEFAULT 'user', org_id INTEGER, name TEXT, description TEXT DEFAULT '', default_branch TEXT DEFAULT 'main', is_private INTEGER DEFAULT 0, is_fork INTEGER DEFAULT 0, fork_id INTEGER, is_mirror INTEGER DEFAULT 0, stars_count INTEGER DEFAULT 0, forks_count INTEGER DEFAULT 0, size_kb INTEGER DEFAULT 0, archived INTEGER DEFAULT 0, archived_at DATETIME, deleted_at DATETIME, license TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE orgs (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, owner_id INTEGER, archived INTEGER DEFAULT 0);
`

func newArchService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(archSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO orgs (id, name, owner_id, archived) VALUES (1, 'acme', 1, 1)`); err != nil {
		t.Fatal(err)
	}
	return &Service{st: store.New(database)}
}

func TestArchiveRepo(t *testing.T) {
	s := newArchService(t)
	ctx := context.Background()
	if _, err := s.st.DB.Exec(`INSERT INTO repos (id, owner_id, name) VALUES (10, 1, 'r')`); err != nil {
		t.Fatal(err)
	}
	r := &domain.Repo{ID: 10, OwnerID: 1, OwnerType: domain.OwnerUser, Name: "r"}
	if s.IsArchived(ctx, r) {
		t.Fatal("should not be archived initially")
	}
	if err := s.Archive(ctx, 10, true); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.ByID(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !s.IsArchived(ctx, fresh) {
		t.Fatal("expected archived")
	}
	if err := s.Archive(ctx, 10, false); err != nil {
		t.Fatal(err)
	}
	fresh, _ = s.ByID(ctx, 10)
	if s.IsArchived(ctx, fresh) {
		t.Fatal("expected unarchived")
	}
}

func TestOrgArchiveMakesReposReadOnly(t *testing.T) {
	s := newArchService(t)
	ctx := context.Background()
	if _, err := s.st.DB.Exec(`INSERT INTO repos (id, owner_id, owner_type, org_id, name) VALUES (20, 1, 'org', 1, 'r')`); err != nil {
		t.Fatal(err)
	}
	r := &domain.Repo{ID: 20, OwnerID: 1, OwnerType: domain.OwnerOrg, OrgID: ptr(int64(1)), Name: "r"}
	if !s.IsArchived(ctx, r) {
		t.Fatal("repo in archived org should be read-only")
	}
}

func ptr(v int64) *int64 { return &v }
