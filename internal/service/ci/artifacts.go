// SPDX-License-Identifier: AGPL-3.0
package ci

import (
	"context"
	"fmt"
)

// Artifact is a stored build output produced by a job.
type Artifact struct {
	ID         int64
	JobID      int64
	Name       string
	StoredPath string
	SizeBytes  int64
}

// AddArtifact records a stored artifact for a job.
func (s *Service) AddArtifact(ctx context.Context, jobID int64, name, storedPath string, size int64) (*Artifact, error) {
	q := s.st.Rebind(`INSERT INTO ci_artifacts (job_id, name, stored_path, size_bytes) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, jobID, name, storedPath, size)
	if err != nil {
		return nil, fmt.Errorf("insert artifact: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("artifact id: %w", err)
	}
	return &Artifact{ID: id, JobID: jobID, Name: name, StoredPath: storedPath, SizeBytes: size}, nil
}

// ListArtifacts returns all artifacts attached to a job in insertion order.
func (s *Service) ListArtifacts(ctx context.Context, jobID int64) ([]Artifact, error) {
	q := s.st.Rebind(`SELECT id, job_id, name, stored_path, size_bytes FROM ci_artifacts WHERE job_id = ? ORDER BY id ASC`)
	rows, err := s.st.DB.QueryContext(ctx, q, jobID)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Artifact
	for rows.Next() {
		var a Artifact
		if err := rows.Scan(&a.ID, &a.JobID, &a.Name, &a.StoredPath, &a.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan artifact: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListArtifactsByPipeline returns all artifacts for every job in a pipeline.
func (s *Service) ListArtifactsByPipeline(ctx context.Context, pipelineID int64) ([]Artifact, error) {
	q := s.st.Rebind(`SELECT a.id, a.job_id, a.name, a.stored_path, a.size_bytes
		FROM ci_artifacts a
		JOIN ci_jobs j ON j.id = a.job_id
		WHERE j.pipeline_id = ?
		ORDER BY a.id ASC`)
	rows, err := s.st.DB.QueryContext(ctx, q, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("list pipeline artifacts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Artifact
	for rows.Next() {
		var a Artifact
		if err := rows.Scan(&a.ID, &a.JobID, &a.Name, &a.StoredPath, &a.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan artifact: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteArtifact removes an artifact by id.
func (s *Service) DeleteArtifact(ctx context.Context, id int64) error {
	q := s.st.Rebind(`DELETE FROM ci_artifacts WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	return nil
}
