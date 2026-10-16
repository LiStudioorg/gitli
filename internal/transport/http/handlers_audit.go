// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/audit"
)

// parseAuditQuery builds a query filter from URL parameters.
func parseAuditQuery(r *http.Request) audit.Query {
	q := audit.Query{
		Event:  r.URL.Query().Get("event"),
		Result: r.URL.Query().Get("result"),
		Target: r.URL.Query().Get("target"),
	}
	if v := r.URL.Query().Get("actor_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			q.ActorID = &n
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Offset = n
		}
	}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q.To = &t
		}
	}
	return q
}

// handleAuditQuery implements GET /api/v1/admin/audit.
func (s *Server) handleAuditQuery(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	logs, err := s.deps.Audit.Query(r.Context(), parseAuditQuery(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

// handleAuditGet implements GET /api/v1/admin/audit/{id}.
func (s *Server) handleAuditGet(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	log, err := s.deps.Audit.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "audit log not found")
		return
	}
	writeJSON(w, http.StatusOK, log)
}

// handleAuditExport implements GET /api/v1/admin/audit/export (streaming).
func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	format := r.URL.Query().Get("format")
	if format != "json" {
		format = "csv"
	}
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	}
	w.Header().Set("Content-Disposition", `attachment; filename="audit-export.`+format+`"`)
	if _, err := s.deps.Audit.Stream(r.Context(), parseAuditQuery(r), format, w); err != nil {
		s.deps.Log.Warn("audit export failed", "error", err)
	}
}

// handleAuditStats implements GET /api/v1/admin/audit/stats.
func (s *Server) handleAuditStats(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	stats, err := s.deps.Audit.StatsSince(r.Context(), time.Now().AddDate(0, 0, -days))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleAuditArchive implements POST /api/v1/admin/audit/archive.
func (s *Server) handleAuditArchive(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	days := s.deps.Cfg.Audit.RetentionDays
	if v := r.URL.Query().Get("retention_days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			days = n
		}
	}
	n, err := s.deps.Audit.Archive(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"archived": n, "retention_days": days})
}

// --- alert rules ---

func (s *Server) handleListAlertRules(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	rules, err := s.deps.Audit.ListAlertRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

type alertRuleReq struct {
	Name      string `json:"name"`
	Event     string `json:"event"`
	Threshold int    `json:"threshold"`
	WindowSec int    `json:"window_seconds"`
	Channel   string `json:"channel"`
	Target    string `json:"target"`
	Enabled   bool   `json:"enabled"`
}

func (s *Server) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var req alertRuleReq
	if !decodeJSON(w, r, &req) {
		return
	}
	rule, err := s.deps.Audit.CreateAlertRule(r.Context(), audit.AlertRule{
		Name: req.Name, Event: req.Event, Threshold: req.Threshold, WindowSec: req.WindowSec,
		Channel: req.Channel, Target: req.Target, Enabled: req.Enabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.Audit.DeleteAlertRule(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
