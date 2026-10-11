// SPDX-License-Identifier: AGPL-3.0
// Package http assembles the HTTP transport: router, middleware and handlers.
package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/transport/http/middleware"
	"github.com/LiStudioorg/gitli/internal/version"
	"github.com/LiStudioorg/gitli/web"
)

// Server holds dependencies for the HTTP transport.
type Server struct {
	cfg *config.Config
	db  *db.DB
	log *slog.Logger
}

// NewServer constructs the HTTP transport.
func NewServer(cfg *config.Config, database *db.DB, log *slog.Logger) *Server {
	return &Server{cfg: cfg, db: database, log: log}
}

// Router builds and returns the fully wired chi router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger(s.log))
	r.Use(middleware.Recover(s.log))
	r.Use(middleware.SecurityHeaders(isHTTPS(s.cfg.Server.ExternalURL)))
	r.Use(middleware.CORS(s.cfg.Server.ExternalURL))

	r.Get("/healthz", s.handleHealth)
	r.Get("/api/v1/version", s.handleVersion)

	api := chi.NewRouter()
	api.Use(middleware.CSRF())
	api.Get("/version", s.handleVersion)
	r.Mount("/api/v1", api)

	web.Mount(r, s.log)
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.db.PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	info := version.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"version": info.Version,
		"commit":  info.Commit,
		"date":    info.Date,
	})
}
