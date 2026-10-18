// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/git"
)

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := urlParam(r, "ref")
	sub := strings.TrimPrefix(urlParam(r, "*"), "/")
	entries, err := s.deps.Git.Tree(r.Context(), path, ref, sub)
	if errors.Is(err, git.ErrEmptyRepo) || (err == nil && len(entries) == 0 && s.deps.Git.IsEmpty(r.Context(), path)) {
		// Fresh repository: an empty listing plus a flag, never a git error.
		writeJSON(w, http.StatusOK, map[string]any{
			"ref":      ref,
			"path":     sub,
			"entries":  []git.TreeEntry{},
			"is_empty": true,
		})
		return
	}
	if err != nil {
		// Internal git failures are logged, not shown to the browser.
		s.deps.Log.Warn("tree listing failed", "repo", rp.Name, "ref", ref, "error", err)
		writeError(w, http.StatusInternalServerError, "无法读取仓库文件，请稍后重试")
		return
	}
	if s.deps.Git.IsEmpty(r.Context(), path) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ref": ref, "path": sub, "entries": []git.TreeEntry{}, "is_empty": true,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ref": ref, "path": sub, "entries": entries, "is_empty": false})
}

func (s *Server) handleBlob(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := urlParam(r, "ref")
	sub := strings.TrimPrefix(urlParam(r, "*"), "/")
	data, err := s.deps.Git.Blob(r.Context(), path, ref, sub)
	if err != nil {
		if s.deps.Git.IsEmpty(r.Context(), path) {
			writeError(w, http.StatusNotFound, "仓库还没有任何提交")
			return
		}
		s.deps.Log.Warn("blob read failed", "repo", rp.Name, "ref", ref, "path", sub, "error", err)
		writeError(w, http.StatusInternalServerError, "无法读取文件内容")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

func (s *Server) handleCommits(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	commits, err := s.deps.Git.Commits(r.Context(), path, urlParam(r, "ref"), 50)
	if err != nil {
		if s.deps.Git.IsEmpty(r.Context(), path) {
			writeJSON(w, http.StatusOK, []git.Commit{})
			return
		}
		s.deps.Log.Warn("commit listing failed", "repo", rp.Name, "error", err)
		writeError(w, http.StatusInternalServerError, "无法读取提交记录")
		return
	}
	if commits == nil {
		commits = []git.Commit{}
	}
	writeJSON(w, http.StatusOK, commits)
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = rp.DefaultBranch
	}
	nodes, err := s.deps.Git.Graph(r.Context(), path, ref, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	base := r.URL.Query().Get("base")
	head := r.URL.Query().Get("head")
	if base == "" || head == "" {
		writeError(w, http.StatusBadRequest, "必须提供 base 与 head 参数")
		return
	}
	diff, tooLarge, err := s.deps.Git.DiffLimited(r.Context(), path, base, head)
	if tooLarge {
		// Do not stream a huge diff into the browser: it would freeze the tab.
		writeJSON(w, http.StatusOK, map[string]any{
			"too_large": true,
			"message":   "变更内容过大，请在本地查看或下载",
		})
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(diff))
}

func (s *Server) handleBlame(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := urlParam(r, "ref")
	sub := strings.TrimPrefix(urlParam(r, "*"), "/")
	blame, err := s.deps.Git.Blame(r.Context(), path, ref, sub)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(blame))
}
