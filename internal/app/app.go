// SPDX-License-Identifier: AGPL-3.0
// Package app wires configuration, storage, database and servers together.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/migrations"
	httptransport "github.com/LiStudioorg/gitli/internal/transport/http"
)

// App is the top-level application runtime.
type App struct {
	Cfg *config.Config
	DB  *db.DB
	Log *slog.Logger
}

// New builds an App from configuration, opening the database and ensuring storage roots.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	log := NewLogger(cfg.Log)
	if err := ensureDirs(cfg); err != nil {
		return nil, err
	}
	database, err := db.Open(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &App{Cfg: cfg, DB: database, Log: log}, nil
}

// Migrate applies all embedded migrations for the configured dialect.
func (a *App) Migrate(ctx context.Context) error {
	paths, err := migrations.All(a.DB.Dialect)
	if err != nil {
		return err
	}
	migs, err := db.LoadMigrations(migrations.FS(), paths)
	if err != nil {
		return err
	}
	if err := a.DB.ApplyMigrations(ctx, migs); err != nil {
		return err
	}
	a.Log.Info("migrations applied", "count", len(migs), "dialect", a.DB.Dialect)
	return nil
}

// Serve starts the HTTP server and blocks until the context is cancelled.
func (a *App) Serve(ctx context.Context) error {
	handler := httptransport.NewServer(a.Cfg, a.DB, a.Log).Router()
	srv := &http.Server{
		Addr:              a.Cfg.Server.HTTP,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		a.Log.Info("http server listening", "addr", a.Cfg.Server.HTTP)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-sigCtx.Done():
		a.Log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// Close releases the database handle.
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
