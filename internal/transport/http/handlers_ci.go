// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/ci"
)

type createPipelineReq struct {
	CommitSHA string `json:"commit_sha"`
	Ref       string `json:"ref"`
}

func (s *Server) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	var req createPipelineReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.CommitSHA == "" {
		writeError(w, http.StatusBadRequest, "commit_sha is required")
		return
	}
	p, err := s.deps.CI.CreatePipeline(r.Context(), rp.ID, req.CommitSHA, req.Ref)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	pipelines, err := s.deps.CI.ListPipelines(r.Context(), rp.ID, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if pipelines == nil {
		pipelines = []*ci.Pipeline{}
	}
	writeJSON(w, http.StatusOK, pipelines)
}

func (s *Server) handleListPipelineJobs(w http.ResponseWriter, r *http.Request) {
	if requireUser(w, r) == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline id")
		return
	}
	jobs, err := s.deps.CI.ListJobs(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if jobs == nil {
		jobs = []*ci.Job{}
	}
	writeJSON(w, http.StatusOK, jobs)
}
