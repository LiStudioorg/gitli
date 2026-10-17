// SPDX-License-Identifier: AGPL-3.0
package follow

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `CREATE TABLE follows (id INTEGER PRIMARY KEY AUTOINCREMENT, follower_id INTEGER, following_id INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE (follower_id, following_id));`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:follow?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestFollowLifecycle(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.Follow(ctx, 1, 1); err == nil {
		t.Fatal("self-follow should fail")
	}
	if err := s.Follow(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	// Idempotent.
	if err := s.Follow(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.IsFollowing(ctx, 1, 2); !ok {
		t.Fatal("expected following")
	}
	if ok, _ := s.IsFollowing(ctx, 2, 1); ok {
		t.Fatal("reverse should not follow")
	}
	followers, _, _ := s.Counts(ctx, 2)
	if followers != 1 {
		t.Fatalf("user2 followers = %d", followers)
	}
	_, following2, _ := s.Counts(ctx, 1)
	if following2 != 1 {
		t.Fatalf("user1 following = %d", following2)
	}
	fl, _ := s.ListFollowers(ctx, 2, 10)
	if len(fl) != 1 || fl[0] != 1 {
		t.Fatalf("followers list = %v", fl)
	}
	if err := s.Unfollow(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.IsFollowing(ctx, 1, 2); ok {
		t.Fatal("expected unfollowed")
	}
}
