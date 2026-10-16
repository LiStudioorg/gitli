// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/ci"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
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

func (s *Server) handleListCache(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	entries, err := s.deps.CI.ListCache(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []ci.CacheEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleDeleteCache(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	key := urlParam(r, "key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if err := s.deps.CI.DeleteCache(r.Context(), rp.ID, key); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListPipelineArtifacts(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline id")
		return
	}
	repoID, err := s.deps.CI.PipelineRepoID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}
	if !s.deps.RBAC.Check(r.Context(), u.ID, u.IsAdmin, rbac.Requirement{Resource: "actions", Action: "read", RepoID: repoID}) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	arts, err := s.deps.CI.ListArtifactsByPipeline(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if arts == nil {
		arts = []ci.Artifact{}
	}
	writeJSON(w, http.StatusOK, arts)
}

func (s *Server) handleDeleteArtifact(w http.ResponseWriter, r *http.Request) {
	if requireUser(w, r) == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid artifact id")
		return
	}
	if err := s.deps.CI.DeleteArtifact(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type createCIRunnerReq struct {
	Name string `json:"name"`
	Tags string `json:"tags"`
}

func (s *Server) handleCreateCIRunner(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var req createCIRunnerReq
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	id, token, err := s.deps.CI.RegisterRunner(r.Context(), req.Name, req.Tags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "token": token})
}

func (s *Server) handleListCIRunners(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	runners, err := s.deps.CI.ListRunners(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if runners == nil {
		runners = []ci.RunnerReg{}
	}
	writeJSON(w, http.StatusOK, runners)
}

type updateCIRunnerReq struct {
	Status string `json:"status"`
}

func (s *Server) handleUpdateCIRunner(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateCIRunnerReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.CI.SetRunnerStatus(r.Context(), id, req.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleDeleteCIRunner(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.CI.DeleteRunner(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListPipelineJobs(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid pipeline id")
		return
	}
	repoID, err := s.deps.CI.PipelineRepoID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}
	if !s.deps.RBAC.Check(r.Context(), u.ID, u.IsAdmin, rbac.Requirement{Resource: "actions", Action: "read", RepoID: repoID}) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
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
