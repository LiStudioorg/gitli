// SPDX-License-Identifier: AGPL-3.0
package ci

import (
	"context"
	"errors"
	"testing"
)

func TestPutGetCache(t *testing.T) {
	s := newCITestService(t)
	ctx := context.Background()

	if err := s.PutCache(ctx, 5, "deps", "/cache/5/deps", 100); err != nil {
		t.Fatalf("put cache: %v", err)
	}
	e, err := s.GetCache(ctx, 5, "deps")
	if err != nil {
		t.Fatalf("get cache: %v", err)
	}
	if e.RepoID != 5 || e.Key != "deps" || e.StoredPath != "/cache/5/deps" || e.SizeBytes != 100 {
		t.Fatalf("unexpected cache entry: %+v", e)
	}

	if err := s.PutCache(ctx, 5, "deps", "/cache/5/deps2", 250); err != nil {
		t.Fatalf("upsert cache: %v", err)
	}
	e2, err := s.GetCache(ctx, 5, "deps")
	if err != nil {
		t.Fatalf("get cache after upsert: %v", err)
	}
	if e2.ID != e.ID || e2.StoredPath != "/cache/5/deps2" || e2.SizeBytes != 250 {
		t.Fatalf("upsert did not update in place: %+v", e2)
	}
}

func TestGetCacheMissing(t *testing.T) {
	s := newCITestService(t)
	ctx := context.Background()

	if _, err := s.GetCache(ctx, 1, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestListAndDeleteCache(t *testing.T) {
	s := newCITestService(t)
	ctx := context.Background()

	if err := s.PutCache(ctx, 1, "a", "/c/a", 1); err != nil {
		t.Fatalf("put a: %v", err)
	}
	if err := s.PutCache(ctx, 1, "b", "/c/b", 2); err != nil {
		t.Fatalf("put b: %v", err)
	}
	if err := s.PutCache(ctx, 2, "a", "/c/other", 3); err != nil {
		t.Fatalf("put other repo: %v", err)
	}

	list, err := s.ListCache(ctx, 1)
	if err != nil {
		t.Fatalf("list cache: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 entries, got %d", len(list))
	}

	if err := s.DeleteCache(ctx, 1, "a"); err != nil {
		t.Fatalf("delete cache: %v", err)
	}
	list, err = s.ListCache(ctx, 1)
	if err != nil {
		t.Fatalf("list cache after delete: %v", err)
	}
	if len(list) != 1 || list[0].Key != "b" {
		t.Fatalf("unexpected list after delete: %+v", list)
	}
}
