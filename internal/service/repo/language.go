// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// LanguageStat is a persisted language statistic for a repo.
type LanguageStat struct {
	Language   string  `json:"language"`
	Bytes      int64   `json:"bytes"`
	Percentage float64 `json:"percentage"`
}

// ReplaceLanguages stores the language breakdown for a repo, replacing prior rows.
func (s *Service) ReplaceLanguages(ctx context.Context, repoID int64, stats []LanguageStat) error {
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, s.st.Rebind(`DELETE FROM repo_languages WHERE repo_id = ?`), repoID); err != nil {
		return err
	}
	for _, st := range stats {
		if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO repo_languages (repo_id, language, bytes) VALUES (?, ?, ?)`), repoID, st.Language, st.Bytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Languages returns the computed language breakdown for a repo.
func (s *Service) Languages(ctx context.Context, repoID int64) ([]LanguageStat, error) {
	q := s.st.Rebind(`SELECT language, bytes FROM repo_languages WHERE repo_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var raw []LanguageStat
	var sum int64
	for rows.Next() {
		var st LanguageStat
		if err := rows.Scan(&st.Language, &st.Bytes); err != nil {
			return nil, err
		}
		raw = append(raw, st)
		sum += st.Bytes
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range raw {
		if sum > 0 {
			raw[i].Percentage = float64(int(float64(raw[i].Bytes)/float64(sum)*1000+0.5)) / 10
		}
	}
	for i := 1; i < len(raw); i++ {
		for j := i; j > 0 && raw[j].Bytes > raw[j-1].Bytes; j-- {
			raw[j], raw[j-1] = raw[j-1], raw[j]
		}
	}
	return raw, nil
}

var _ = domain.PermRead
