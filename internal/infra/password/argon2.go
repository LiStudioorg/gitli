// SPDX-License-Identifier: AGPL-3.0
// Package password implements Argon2id password hashing in PHC format.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params controls Argon2id resource usage.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams returns the recommended Argon2id parameters.
func DefaultParams() Params {
	return Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// maxConcurrentKDF bounds simultaneous Argon2 derivations. Each derivation
// allocates Memory KiB (64 MiB by default), so unbounded concurrency from
// parallel Git pushes can exhaust the host. Queuing is far cheaper than OOM.
const maxConcurrentKDF = 2

// kdfSem serialises Argon2 derivations across all Hasher instances.
var kdfSem = make(chan struct{}, maxConcurrentKDF)

// Hasher hashes and verifies passwords using Argon2id with an optional pepper.
type Hasher struct {
	params Params
	pepper []byte
}

// NewHasher builds a Hasher. A nil pepper disables peppering.
func NewHasher(p Params, pepper string) *Hasher {
	if p.Memory == 0 {
		p = DefaultParams()
	}
	return &Hasher{params: p, pepper: []byte(pepper)}
}

var (
	// ErrInvalidHash indicates a malformed stored hash.
	ErrInvalidHash = errors.New("password: invalid hash format")
	// ErrIncompatibleVersion indicates unsupported argon2 version.
	ErrIncompatibleVersion = errors.New("password: incompatible argon2 version")
	// ErrMismatch indicates the password did not match.
	ErrMismatch = errors.New("password: mismatch")
)

// Hash derives a PHC-formatted Argon2id hash for the given plaintext.
func (h *Hasher) Hash(plaintext string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: read salt: %w", err)
	}
	key := h.derive(h.peppered(plaintext), salt, h.params.Iterations, h.params.Memory, h.params.Parallelism, h.params.KeyLength)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.params.Memory, h.params.Iterations, h.params.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// derive runs argon2id while honouring the process-wide concurrency cap.
func (h *Hasher) derive(secret, salt []byte, iterations, memory uint32, parallelism uint8, keyLen uint32) []byte {
	kdfSem <- struct{}{}
	defer func() { <-kdfSem }()
	return argon2.IDKey(secret, salt, iterations, memory, parallelism, keyLen)
}

// Verify checks plaintext against a PHC hash, returning nil on success.
func (h *Hasher) Verify(plaintext, encoded string) error {
	p, salt, key, err := decodeHash(encoded)
	if err != nil {
		return err
	}
	computed := h.derive(h.peppered(plaintext), salt, p.Iterations, p.Memory, p.Parallelism, uint32(len(key)))
	if subtle.ConstantTimeCompare(key, computed) != 1 {
		return ErrMismatch
	}
	return nil
}

func (h *Hasher) peppered(plaintext string) []byte {
	if len(h.pepper) == 0 {
		return []byte(plaintext)
	}
	out := make([]byte, 0, len(plaintext)+len(h.pepper))
	out = append(out, plaintext...)
	out = append(out, h.pepper...)
	return out
}

func decodeHash(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return Params{}, nil, nil, ErrIncompatibleVersion
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(key))
	return p, salt, key, nil
}
