// SPDX-License-Identifier: AGPL-3.0
// Package token generates and hashes opaque tokens for PATs, sessions and API access.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// Token is a generated secret with its hash and display prefix.
type Token struct {
	Plain  string
	Hash   string
	Prefix string
}

// New generates a random token with the given prefix (e.g. "glpat").
// 32 random bytes produce a URL-safe base64 string.
func New(prefix string) (Token, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, fmt.Errorf("token: read random: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	plain := body
	if prefix != "" {
		plain = prefix + "_" + body
	}
	return Token{
		Plain:  plain,
		Hash:   Hash(plain),
		Prefix: body[:8],
	}, nil
}

// Hash returns the hex-encoded SHA-256 digest of the plaintext token.
func Hash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// Verify compares the hash of plain with a stored hash in constant time.
func Verify(plain, storedHash string) bool {
	return subtleEqual(Hash(plain), storedHash)
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
