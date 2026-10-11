// SPDX-License-Identifier: AGPL-3.0
package git

import (
	"context"
	"strings"
	"testing"
)

func TestParsePreReceive(t *testing.T) {
	in := "0000000000000000000000000000000000000000 abcdef1234567890abcdef1234567890abcdef12 refs/heads/main\n" +
		"abcdef1234567890abcdef1234567890abcdef12 def4567890abcdef1234567890abcdef12345678 refs/tags/v1.0\n"
	updates, err := ParsePreReceive(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(updates))
	}
	if updates[0].BranchName() != "main" {
		t.Fatalf("branch=%q", updates[0].BranchName())
	}
	if updates[0].IsDelete() {
		t.Fatal("first update creates a branch, should not be a delete")
	}
	if updates[1].TagName() != "v1.0" {
		t.Fatalf("tag=%q", updates[1].TagName())
	}
}

func TestEnforceProtections(t *testing.T) {
	checker := fakeChecker{branches: []ProtectionRule{{Pattern: "main"}}, tags: []string{"v*"}}
	updates := []RefUpdate{{Old: "a", New: "b", Ref: "refs/heads/main"}}
	if err := EnforceProtections(context.Background(), checker, 1, updates, false); err == nil {
		t.Fatal("expected protected branch push to be rejected")
	}
	if err := EnforceProtections(context.Background(), checker, 1, updates, true); err != nil {
		t.Fatalf("admin should bypass: %v", err)
	}
	tagUpdates := []RefUpdate{{Old: "a", New: "b", Ref: "refs/tags/v1.0"}}
	if err := EnforceProtections(context.Background(), checker, 1, tagUpdates, false); err == nil {
		t.Fatal("expected protected tag update to be rejected")
	}
}

type fakeChecker struct {
	branches []ProtectionRule
	tags     []string
}

func (f fakeChecker) BranchProtections(_ context.Context, _ int64) ([]ProtectionRule, error) {
	return f.branches, nil
}
func (f fakeChecker) TagProtections(_ context.Context, _ int64) ([]string, error) {
	return f.tags, nil
}
