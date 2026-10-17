// SPDX-License-Identifier: AGPL-3.0
package http

import "net/http"

func (s *Server) handleGetDigestFrequency(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	freq, err := s.deps.Digest.GetFrequency(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"frequency": freq})
}

type digestFreqReq struct {
	Frequency string `json:"frequency"`
}

func (s *Server) handleSetDigestFrequency(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req digestFreqReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Digest.SetFrequency(r.Context(), u.ID, req.Frequency); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "frequency": req.Frequency})
}
