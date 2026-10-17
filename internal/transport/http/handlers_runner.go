// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/runner"
)

// runnerAuth validates the Bearer token + X-Runner-ID pair.
func (s *Server) runnerAuth(w http.ResponseWriter, r *http.Request) (*runner.Runner, bool) {
	if s.deps.Runners == nil {
		writeError(w, http.StatusNotFound, "Runner 子系统未启用")
		return nil, false
	}
	auth := r.Header.Get("Authorization")
	token := ""
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token = auth[7:]
	}
	runnerID := r.Header.Get("X-Runner-ID")
	if runnerID == "" {
		runnerID = r.URL.Query().Get("runner_id")
	}
	if token == "" || runnerID == "" {
		writeError(w, http.StatusUnauthorized, "缺少 Runner 凭据")
		return nil, false
	}
	rr, err := s.deps.Runners.Authenticate(r.Context(), runnerID, token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Runner 凭据无效")
		return nil, false
	}
	return rr, true
}

type runnerHeartbeatReq struct {
	RunnerID string   `json:"runner_id"`
	Status   string   `json:"status"`
	Version  string   `json:"version"`
	OS       string   `json:"os"`
	Arch     string   `json:"arch"`
	Labels   []string `json:"labels"`
}

func (s *Server) handleRunnerHeartbeat(w http.ResponseWriter, r *http.Request) {
	rr, ok := s.runnerAuth(w, r)
	if !ok {
		return
	}
	var req runnerHeartbeatReq
	if r.ContentLength > 0 {
		_ = decodeJSON(w, r, &req)
	}
	labels := ""
	if len(req.Labels) > 0 {
		if b, err := jsonMarshalStrings(req.Labels); err == nil {
			labels = b
		}
	}
	if err := s.deps.Runners.Heartbeat(r.Context(), rr.RunnerID, firstNonEmpty(req.Status, "active"), req.Version, req.OS, req.Arch, labels); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// handleRunnerNextJob claims the next pending job, or returns 204.
func (s *Server) handleRunnerNextJob(w http.ResponseWriter, r *http.Request) {
	rr, ok := s.runnerAuth(w, r)
	if !ok {
		return
	}
	_ = s.deps.Runners.Heartbeat(r.Context(), rr.RunnerID, "active", rr.Version, rr.OS, rr.Arch, "")
	job, err := s.deps.Runners.ClaimJob(r.Context(), rr.RunnerID)
	if err != nil {
		if err == runner.ErrNoJob {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	repo, err := s.deps.Repos.ByID(r.Context(), job.RepoID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "仓库不存在")
		return
	}
	ext := s.deps.External
	repoURL := ext.HTTPSCloneURL(s.ownerSlug(r, repo.OwnerID), repo.Name)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          strconv.FormatInt(job.ID, 10),
		"run_id":      strconv.FormatInt(job.ID, 10),
		"repo_url":    repoURL,
		"ref":         job.Ref,
		"sha":         job.CommitSHA,
		"token":       "",
		"workflow":    job.WorkflowContent,
		"env":         map[string]string{},
		"timeout_sec": int((30 * time.Minute).Seconds()),
	})
}

func (s *Server) handleRunnerJobLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.runnerAuth(w, r); !ok {
		return
	}
	jobID, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的任务 ID")
		return
	}
	// Stream the request body in bounded chunks to avoid unbounded memory.
	buf := make([]byte, 0, 32*1024)
	tmp := make([]byte, 16*1024)
	for {
		n, rerr := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) >= 32*1024 {
				if err := s.deps.Runners.AppendLog(r.Context(), jobID, string(buf)); err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				buf = buf[:0]
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			writeError(w, http.StatusBadRequest, "读取请求体失败")
			return
		}
	}
	if len(buf) > 0 {
		if err := s.deps.Runners.AppendLog(r.Context(), jobID, string(buf)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type runnerResultReq struct {
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error"`
}

func (s *Server) handleRunnerJobResult(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.runnerAuth(w, r); !ok {
		return
	}
	jobID, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的任务 ID")
		return
	}
	var req runnerResultReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Error != "" {
		_ = s.deps.Runners.AppendLog(r.Context(), jobID, "\n"+req.Error+"\n")
	}
	if err := s.deps.Runners.CompleteJob(r.Context(), jobID, req.Status, req.ExitCode); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// handleRunnerJobLogStream streams stored logs via SSE for the web UI.
func (s *Server) handleRunnerJobLogStream(w http.ResponseWriter, r *http.Request) {
	jobID, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的任务 ID")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "不支持流式传输")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	after := int64(0)
	for {
		chunks, err := s.deps.Runners.Logs(r.Context(), jobID, after, 200)
		if err != nil {
			return
		}
		for _, c := range chunks {
			after = c.ID
			writeSSE(w, "log", c.Chunk)
		}
		if len(chunks) > 0 {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *Server) ownerSlug(r *http.Request, ownerID int64) string {
	if u, err := s.deps.Users.ByID(r.Context(), ownerID); err == nil {
		return u.Username
	}
	return ""
}

func firstNonEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// handleListRepoWorkflowJobs lists recent runner jobs for a repository.
func (s *Server) handleListRepoWorkflowJobs(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	jobs, err := s.deps.Runners.ListJobsByRepo(r.Context(), rp.ID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

// handleCancelWorkflowJob cancels a job (requires write permission).
func (s *Server) handleCancelWorkflowJob(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	job, err := s.deps.Runners.JobByID(r.Context(), id)
	if err != nil || job.RepoID != rp.ID {
		writeError(w, http.StatusNotFound, "任务不存在")
		return
	}
	if err := s.deps.Runners.CancelJob(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "runner.job_cancel", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type runnerRegisterReq struct {
	ID      string   `json:"id"`
	OS      string   `json:"os"`
	Arch    string   `json:"arch"`
	Labels  []string `json:"labels"`
	Version string   `json:"version"`
}

// handleRunnerRegister validates credentials and records capability info.
// Unlike a token-exchange flow, the configured id+token remain valid; this is
// only a connectivity/registration report.
func (s *Server) handleRunnerRegister(w http.ResponseWriter, r *http.Request) {
	rr, ok := s.runnerAuth(w, r)
	if !ok {
		return
	}
	var req runnerRegisterReq
	if r.ContentLength > 0 {
		_ = decodeJSON(w, r, &req)
	}
	labels := ""
	if len(req.Labels) > 0 {
		if b, err := jsonMarshalStrings(req.Labels); err == nil {
			labels = b
		}
	}
	if err := s.deps.Runners.Heartbeat(r.Context(), rr.RunnerID, "active", req.Version, req.OS, req.Arch, labels); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"runner_id": rr.RunnerID,
		"name":      rr.Name,
		"status":    "ok",
	})
}
