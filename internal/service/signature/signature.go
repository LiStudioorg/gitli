// SPDX-License-Identifier: AGPL-3.0
// Package signature verifies and caches commit signature status.
package signature

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Signature describes the signature state of a commit.
type Signature struct {
	ID        int64
	RepoID    int64
	CommitSHA string
	SignerID  *int64
	Type      string
	Verified  bool
}

// Service verifies commit signatures and caches the result.
type Service struct {
	st     *store.Store
	runner *gitcmd.Runner
}

// New builds a signature Service.
func New(st *store.Store, runner *gitcmd.Runner) *Service {
	return &Service{st: st, runner: runner}
}

// shaPattern matches a full-length hexadecimal object name.
var shaPattern = regexp.MustCompile(`^[0-9a-fA-F]{40,64}$`)

// validSHA reports whether sha is a plausible hex object name.
func validSHA(sha string) bool {
	if sha == "" || strings.HasPrefix(sha, "-") {
		return false
	}
	if strings.ContainsAny(sha, " \t\n\r") {
		return false
	}
	for _, r := range sha {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return shaPattern.MatchString(sha)
}

// Verify runs git against the on-disk repository, records the result and
// returns the signature state. A commit that cannot be verified is reported
// with Verified=false rather than failing the call; an invalid sha or an
// unreadable repository is an error.
func (s *Service) Verify(ctx context.Context, repoPath, commitSHA string) (*Signature, error) {
	if !validSHA(commitSHA) {
		return nil, fmt.Errorf("signature: invalid commit sha %q", commitSHA)
	}
	if repoPath == "" {
		return nil, fmt.Errorf("signature: empty repo path")
	}
	if s.runner == nil {
		return nil, fmt.Errorf("signature: git runner not configured")
	}

	sig := &Signature{CommitSHA: strings.ToLower(commitSHA), Type: "none", Verified: false}

	if _, err := s.runner.Run(ctx, repoPath, "cat-file", "-e", commitSHA+"^{commit}"); err != nil {
		return nil, fmt.Errorf("signature: unknown commit %s: %w", commitSHA, err)
	}

	if _, err := s.runner.Run(ctx, repoPath, "verify-commit", "--raw", commitSHA); err == nil {
		sig.Verified = true
	}

	out, err := s.runner.Run(ctx, repoPath, "log", "-1", "--format=%G?", commitSHA)
	if err == nil {
		sig.Type = classify(strings.TrimSpace(out))
	}
	return sig, nil
}

// classify maps git's %G? indicator to a signature type label.
func classify(code string) string {
	switch {
	case strings.HasPrefix(code, "G"):
		return "gpg"
	case strings.HasPrefix(code, "U"):
		return "gpg"
	case strings.HasPrefix(code, "X"):
		return "x509"
	case strings.HasPrefix(code, "Y"):
		return "x509"
	case strings.HasPrefix(code, "B"):
		return "gpg"
	case strings.HasPrefix(code, "R"):
		return "gpg"
	default:
		return "none"
	}
}

// Get returns the cached signature for a commit, if previously recorded.
func (s *Service) Get(ctx context.Context, repoID int64, commitSHA string) (*Signature, error) {
	if !validSHA(commitSHA) {
		return nil, fmt.Errorf("signature: invalid commit sha %q", commitSHA)
	}
	q := s.st.Rebind(`SELECT id, repo_id, commit_sha, signer_id, signature_type, verified FROM commit_signatures WHERE repo_id = ? AND commit_sha = ?`)
	var sig Signature
	var signerID sql.NullInt64
	var verified int
	err := s.st.DB.QueryRowContext(ctx, q, repoID, strings.ToLower(commitSHA)).Scan(
		&sig.ID, &sig.RepoID, &sig.CommitSHA, &signerID, &sig.Type, &verified)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("signature: get: %w", err)
	}
	if signerID.Valid {
		v := signerID.Int64
		sig.SignerID = &v
	}
	sig.Verified = verified != 0
	return &sig, nil
}

// Save inserts or updates the stored signature for a commit.
func (s *Service) Save(ctx context.Context, sig *Signature) error {
	if sig == nil {
		return fmt.Errorf("signature: nil signature")
	}
	var verified int
	if sig.Verified {
		verified = 1
	}
	q := s.st.Rebind(`UPDATE commit_signatures SET signer_id = ?, signature_type = ?, verified = ? WHERE repo_id = ? AND commit_sha = ?`)
	res, err := s.st.DB.ExecContext(ctx, q, sig.SignerID, sig.Type, verified, sig.RepoID, sig.CommitSHA)
	if err != nil {
		return fmt.Errorf("signature: save: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	ins := s.st.Rebind(`INSERT INTO commit_signatures (repo_id, commit_sha, signer_id, signature_type, verified) VALUES (?, ?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, ins, sig.RepoID, sig.CommitSHA, sig.SignerID, sig.Type, verified); err != nil {
		return fmt.Errorf("signature: save: %w", err)
	}
	return nil
}
