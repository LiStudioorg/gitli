// SPDX-License-Identifier: AGPL-3.0
package crypto

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	c, err := New(key)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("totp-secret")
	sealed, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Decrypt(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plain) {
		t.Fatalf("roundtrip mismatch: %q", got)
	}
}

func TestInvalidKey(t *testing.T) {
	if _, err := New([]byte("short")); err != ErrInvalidKey {
		t.Fatalf("got %v, want ErrInvalidKey", err)
	}
}

func TestTamperDetected(t *testing.T) {
	key, _ := GenerateKey()
	c, _ := New(key)
	sealed, _ := c.Encrypt([]byte("data"))
	sealed[len(sealed)-1] ^= 0xff
	if _, err := c.Decrypt(sealed); err == nil {
		t.Fatal("tampered ciphertext should fail")
	}
}
