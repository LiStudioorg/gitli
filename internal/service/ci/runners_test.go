// SPDX-License-Identifier: AGPL-3.0
package ci

import (
	"context"
	"testing"
)

func TestRegisterAndAuthenticateRunner(t *testing.T) {
	s := newCITestService(t)
	ctx := context.Background()

	id, token, err := s.RegisterRunner(ctx, "runner-1", "linux,docker")
	if err != nil {
		t.Fatalf("register runner: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero runner id")
	}
	if token == "" {
		t.Fatalf("expected a plaintext token")
	}

	ok, err := s.AuthenticateRunner(ctx, id, token)
	if err != nil {
		t.Fatalf("authenticate runner: %v", err)
	}
	if !ok {
		t.Fatalf("expected authentication to succeed with correct token")
	}

	bad, err := s.AuthenticateRunner(ctx, id, "glcr_wrong")
	if err != nil {
		t.Fatalf("authenticate wrong token: %v", err)
	}
	if bad {
		t.Fatalf("expected authentication to fail with wrong token")
	}
}

func TestListSetStatusDeleteRunner(t *testing.T) {
	s := newCITestService(t)
	ctx := context.Background()

	id, _, err := s.RegisterRunner(ctx, "runner-1", "linux")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	runners, err := s.ListRunners(ctx)
	if err != nil {
		t.Fatalf("list runners: %v", err)
	}
	if len(runners) != 1 {
		t.Fatalf("want 1 runner, got %d", len(runners))
	}
	if runners[0].Name != "runner-1" || runners[0].Tags != "linux" || runners[0].Status != "offline" {
		t.Fatalf("unexpected runner: %+v", runners[0])
	}

	if err := s.SetRunnerStatus(ctx, id, "online"); err != nil {
		t.Fatalf("set status: %v", err)
	}
	runners, err = s.ListRunners(ctx)
	if err != nil {
		t.Fatalf("list after status: %v", err)
	}
	if runners[0].Status != "online" {
		t.Fatalf("status not updated: %+v", runners[0])
	}

	if err := s.DeleteRunner(ctx, id); err != nil {
		t.Fatalf("delete runner: %v", err)
	}
	runners, err = s.ListRunners(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(runners) != 0 {
		t.Fatalf("want 0 runners after delete, got %d", len(runners))
	}
}
