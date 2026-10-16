// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/repo"
)

type forkReq struct {
	TargetName string `json:"name"`
	Org        string `json:"org"`
}

func (s *Server) handleForkRepo(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	var req forkReq
	if r.ContentLength > 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	params := repo.ForkParams{
		Source:     rp,
		TargetName: req.TargetName,
		OwnerID:    u.ID,
		OwnerType:  domain.OwnerUser,
	}
	if req.Org != "" {
		o, err := s.deps.Orgs.ByName(r.Context(), req.Org)
		if err != nil {
			writeError(w, http.StatusBadRequest, "org not found")
			return
		}
		params.OwnerType = domain.OwnerOrg
		params.OrgID = &o.ID
		params.OwnerID = o.OwnerID
	}
	fork, err := s.deps.Repos.Fork(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "repo.fork", "success")
	s.enqueueLanguageRefresh(fork.ID)
	writeJSON(w, http.StatusCreated, fork)
}
