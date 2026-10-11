// SPDX-License-Identifier: AGPL-3.0
// Package weakpass embeds a weak-password blocklist and optional
// HaveIBeenPwned k-anonymity lookup.
package weakpass

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// minLength is the enforced minimum password length.
const minLength = 12

//go:generate sh -c "test -f top10000.txt || curl -sL https://raw.githubusercontent.com/danielmiessler/SecLists/master/Passwords/Common-Credentials/10-million-password-list-top-10000.txt -o top10000.txt"

// Checker validates passwords against length and weakness rules.
type Checker struct {
	mu     sync.RWMutex
	weak   map[string]struct{}
	hibp   *HIBP
	minLen int
}

// New builds a Checker from an embedded word list.
func New(minLen int) *Checker {
	if minLen < minLength {
		minLen = minLength
	}
	c := &Checker{weak: make(map[string]struct{}), minLen: minLen}
	c.loadDefault()
	return c
}

// WithHIBP attaches a HaveIBeenPwned client.
func (c *Checker) WithHIBP(h *HIBP) *Checker {
	c.hibp = h
	return c
}

// Validate enforces minimum length and blocklist membership.
func (c *Checker) Validate(password string) error {
	if len(password) < c.minLen {
		return fmt.Errorf("weakpass: password must be at least %d characters", c.minLen)
	}
	c.mu.RLock()
	_, bad := c.weak[strings.ToLower(password)]
	c.mu.RUnlock()
	if bad {
		return fmt.Errorf("weakpass: password is too common")
	}
	if c.hibp != nil {
		pwned, err := c.hibp.Pwned(password)
		if err == nil && pwned {
			return fmt.Errorf("weakpass: password appears in known breaches")
		}
	}
	return nil
}

func (c *Checker) loadDefault() {
	sc := bufio.NewScanner(strings.NewReader(embeddedList))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		c.weak[strings.ToLower(line)] = struct{}{}
	}
}

// HIBP queries the HaveIBeenPwned range API using k-anonymity.
type HIBP struct {
	client *http.Client
}

// NewHIBP builds an HIBP client.
func NewHIBP() *HIBP {
	return &HIBP{client: &http.Client{Timeout: 5 * time.Second}}
}

// Pwned reports whether the password hash prefix exists in the breach corpus.
func (h *HIBP) Pwned(password string) (bool, error) {
	sum := sha1.Sum([]byte(password))
	hexHash := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := hexHash[:5], hexHash[5:]
	resp, err := h.client.Get("https://api.pwnedpasswords.com/range/" + prefix)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("weakpass: hibp status %d", resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ":", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == suffix {
			return true, nil
		}
	}
	return false, sc.Err()
}
