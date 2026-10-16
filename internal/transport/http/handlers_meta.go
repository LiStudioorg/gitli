// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// repoMeta is the public OpenGraph metadata payload for a repository page.
type repoMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Image       string `json:"image"`
}

// handleRepoMeta returns public OpenGraph metadata computed from the repo and
// the externally visible base URL. Private repos resolve to 404.
func (s *Server) handleRepoMeta(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, urlParam(r, "owner"), urlParam(r, "repo"), domain.PermRead)
	if !ok {
		return
	}
	base := strings.TrimRight(s.deps.External.HTTPBase, "/")
	title := urlParam(r, "owner") + "/" + rp.Name
	if rp.Description != "" {
		title += " - " + rp.Description
	}
	image := ""
	if rp.OwnerType == domain.OwnerOrg {
		image = base + "/api/v1/orgs/" + urlParam(r, "owner")
	}
	writeJSON(w, http.StatusOK, repoMeta{
		Title:       title,
		Description: rp.Description,
		URL:         base + "/" + urlParam(r, "owner") + "/" + rp.Name,
		Image:       image,
	})
}
