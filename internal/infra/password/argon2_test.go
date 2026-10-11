// SPDX-License-Identifier: AGPL-3.0
package password

import "testing"

func TestHashVerify(t *testing.T) {
	h := NewHasher(DefaultParams(), "pepper")
	hash, err := h.Hash("s3cret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := h.Verify("s3cret", hash); err != nil {
		t.Fatalf("verify correct password: %v", err)
	}
	if err := h.Verify("wrong", hash); err != ErrMismatch {
		t.Fatalf("verify wrong password: got %v, want mismatch", err)
	}
}

func TestPepperMatters(t *testing.T) {
	h1 := NewHasher(DefaultParams(), "pepper-a")
	h2 := NewHasher(DefaultParams(), "pepper-b")
	hash, _ := h1.Hash("pw")
	if err := h2.Verify("pw", hash); err != ErrMismatch {
		t.Fatalf("different pepper should fail: got %v", err)
	}
}

func TestPHCFormat(t *testing.T) {
	h := NewHasher(DefaultParams(), "")
	hash, err := h.Hash("pw")
	if err != nil {
		t.Fatal(err)
	}
	if len(hash) < 9 || hash[:9] != "$argon2id" {
		t.Fatalf("unexpected PHC format: %s", hash)
	}
}
