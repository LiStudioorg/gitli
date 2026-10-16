// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"encoding/json"
	"net/http"
)

// requireAdmin returns the admin user or writes 403.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	u := requireUser(w, r)
	if u == nil {
		return false
	}
	if !u.IsAdmin {
		writeError(w, http.StatusForbidden, "administrator privileges required")
		return false
	}
	return true
}

type createRunnerReq struct {
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
}

func (s *Server) handleCreateRunner(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var req createRunnerReq
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	labels := "[]"
	if len(req.Labels) > 0 {
		if b, err := json.Marshal(req.Labels); err == nil {
			labels = string(b)
		}
	}
	res, err := s.deps.Runners.Create(r.Context(), req.Name, labels)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":        res.Runner.ID,
		"runner_id": res.Runner.RunnerID,
		"name":      res.Runner.Name,
		"labels":    res.Runner.Labels,
		"token":     res.Token,
	})
}

func (s *Server) handleListRunners(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	runners, err := s.deps.Runners.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type row struct {
		ID         int64  `json:"id"`
		RunnerID   string `json:"runner_id"`
		Name       string `json:"name"`
		Labels     string `json:"labels"`
		Version    string `json:"version"`
		OS         string `json:"os"`
		Arch       string `json:"arch"`
		Status     string `json:"status"`
		LastSeenAt any    `json:"last_seen_at"`
	}
	out := make([]row, 0, len(runners))
	for _, rr := range runners {
		out = append(out, row{
			ID: rr.ID, RunnerID: rr.RunnerID, Name: rr.Name, Labels: rr.Labels,
			Version: rr.Version, OS: rr.OS, Arch: rr.Arch, Status: rr.Status, LastSeenAt: rr.LastSeenAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleResetRunnerToken(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	token, err := s.deps.Runners.ResetToken(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "runner not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token})
}

type updateRunnerReq struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Labels []string `json:"labels"`
}

func (s *Server) handleUpdateRunner(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateRunnerReq
	if !decodeJSON(w, r, &req) {
		return
	}
	labels := ""
	if req.Labels != nil {
		if b, err := json.Marshal(req.Labels); err == nil {
			labels = string(b)
		}
	}
	if err := s.deps.Runners.Update(r.Context(), id, req.Name, labels, req.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleDeleteRunner(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.Runners.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListRunnerJobsAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rr, err := s.deps.Runners.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "runner not found")
		return
	}
	jobs, err := s.deps.Runners.ListJobsByRunner(r.Context(), rr.RunnerID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}
