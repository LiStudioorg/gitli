// SPDX-License-Identifier: AGPL-3.0
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

type loginLimiter struct {
	mu        sync.Mutex
	attempts  map[string]*attemptRecord
	max       int
	lockout   time.Duration
	lastSweep time.Time
}

// sweepInterval bounds how often expired records are purged. Without this the
// map grows unbounded with attacker-chosen usernames (M-09 memory leak).
const sweepInterval = 5 * time.Minute

type attemptRecord struct {
	count  int
	lastAt time.Time
}

func newLoginLimiter(max int, lockout time.Duration) *loginLimiter {
	return &loginLimiter{attempts: map[string]*attemptRecord{}, max: max, lockout: lockout}
}

func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweepLocked()
	rec, ok := l.attempts[key]
	if !ok {
		return true
	}
	if time.Since(rec.lastAt) >= l.lockout {
		delete(l.attempts, key)
		return true
	}
	return rec.count < l.max
}

func (l *loginLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweepLocked()
	rec, ok := l.attempts[key]
	if !ok || time.Since(rec.lastAt) >= l.lockout {
		l.attempts[key] = &attemptRecord{count: 1, lastAt: time.Now()}
		return
	}
	rec.count++
	rec.lastAt = time.Now()
}

// sweepLocked drops records whose lockout window has elapsed. Called with mu held.
func (l *loginLimiter) sweepLocked() {
	now := time.Now()
	if now.Sub(l.lastSweep) < sweepInterval {
		return
	}
	l.lastSweep = now
	for k, rec := range l.attempts {
		if now.Sub(rec.lastAt) >= l.lockout {
			delete(l.attempts, k)
		}
	}
}

// success clears the failure counter after a valid credential, so a user who
// eventually types the right password is not punished for earlier typos.
func (l *loginLimiter) success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func generateRecoveryCodes(n int) ([]string, []string, error) {
	plain := make([]string, 0, n)
	hashes := make([]string, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 5)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, err
		}
		code := hex.EncodeToString(buf)
		plain = append(plain, code)
		hashes = append(hashes, hashString(code))
	}
	return plain, hashes, nil
}

func (s *Service) storeRecoveryCodes(ctx context.Context, userID int64, hashes []string) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM user_secrets WHERE user_id = ? AND name = 'recovery'`), userID); err != nil {
		return err
	}
	for _, h := range hashes {
		if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT INTO user_secrets (user_id, name, ciphertext) VALUES (?, 'recovery', ?)`), userID, h); err != nil {
			return fmt.Errorf("auth: store recovery code: %w", err)
		}
	}
	return nil
}

func (s *Service) verifyRecoveryCode(ctx context.Context, userID int64, code string) (bool, error) {
	hash := hashString(code)
	q := s.st.Rebind(`SELECT id FROM user_secrets WHERE user_id = ? AND name = 'recovery' AND ciphertext = ?`)
	var id int64
	err := s.st.DB.QueryRowContext(ctx, q, userID, hash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM user_secrets WHERE id = ?`), id); err != nil {
		return false, err
	}
	return true, nil
}
