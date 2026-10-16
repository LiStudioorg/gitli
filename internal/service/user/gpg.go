// SPDX-License-Identifier: AGPL-3.0
package user

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// fingerprintSigningKey computes a stable fingerprint for a GPG public key.
// It parses OpenPGP armored keys to obtain the canonical key fingerprint; if
// parsing fails it falls back to a SHA-256 of the normalized key material so
// uploads never fail on unusual-but-valid inputs.
func fingerprintSigningKey(pubKey string) (string, error) {
	trimmed := strings.TrimSpace(pubKey)
	if trimmed == "" {
		return "", fmt.Errorf("user: empty signing key")
	}
	if strings.Contains(trimmed, "BEGIN PGP PUBLIC KEY BLOCK") {
		if entities, err := openpgp.ReadArmoredKeyRing(strings.NewReader(trimmed)); err == nil && len(entities) > 0 {
			return strings.ToUpper(hex.EncodeToString(entities[0].PrimaryKey.Fingerprint[:])), nil
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(strings.Fields(trimmed), "\n")))
	return hex.EncodeToString(sum[:]), nil
}
