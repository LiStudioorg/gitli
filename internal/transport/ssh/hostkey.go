// SPDX-License-Identifier: AGPL-3.0
package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// generateHostKey creates a fresh Ed25519 host key and its PEM encoding.
func generateHostKey() (ssh.Signer, []byte, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, nil, err
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, nil, err
	}
	pemBytes := pem.EncodeToMemory(pemBlock)
	return signer, pemBytes, nil
}
