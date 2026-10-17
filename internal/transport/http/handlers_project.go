// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	projects, err := s.deps.Projects.ListByRepo(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

type createProjectReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	var req createProjectReq
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := s.deps.Projects.Create(r.Context(), u.ID, &rp.ID, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, cols, err := s.deps.Projects.Board(r.Context(), id)
	if err != nil || (p.RepoID != nil && *p.RepoID != rp.ID) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": p, "columns": cols})
}

type addCardReq struct {
	ColumnID int64  `json:"column_id"`
	IssueID  *int64 `json:"issue_id"`
	PullID   *int64 `json:"pull_id"`
	Title    string `json:"title"`
}

func (s *Server) handleAddProjectCard(w http.ResponseWriter, r *http.Request) {
	if requireUser(w, r) == nil {
		return
	}
	if _, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite); !ok {
		return
	}
	var req addCardReq
	if !decodeJSON(w, r, &req) {
		return
	}
	card, err := s.deps.Projects.AddCard(r.Context(), req.ColumnID, req.IssueID, req.PullID, req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, card)
}

type moveCardReq struct {
	ColumnID int64 `json:"column_id"`
	Position int   `json:"position"`
}

func (s *Server) handleMoveProjectCard(w http.ResponseWriter, r *http.Request) {
	if requireUser(w, r) == nil {
		return
	}
	if _, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite); !ok {
		return
	}
	cardID, err := pathInt(r, "cardId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}
	var req moveCardReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Projects.MoveCard(r.Context(), cardID, req.ColumnID, req.Position); err != nil {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleDeleteProjectCard(w http.ResponseWriter, r *http.Request) {
	if requireUser(w, r) == nil {
		return
	}
	if _, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite); !ok {
		return
	}
	cardID, err := pathInt(r, "cardId")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid card id")
		return
	}
	if err := s.deps.Projects.DeleteCard(r.Context(), cardID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type addColumnReq struct {
	Name string `json:"name"`
}

func (s *Server) handleAddProjectColumn(w http.ResponseWriter, r *http.Request) {
	if requireUser(w, r) == nil {
		return
	}
	if _, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite); !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req addColumnReq
	if !decodeJSON(w, r, &req) {
		return
	}
	col, err := s.deps.Projects.AddColumn(r.Context(), id, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, col)
}
