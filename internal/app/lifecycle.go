// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"time"

	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/migrations"
	"github.com/LiStudioorg/gitli/internal/platform"
	httptransport "github.com/LiStudioorg/gitli/internal/transport/http"
	sshdispatch "github.com/LiStudioorg/gitli/internal/transport/ssh"
)

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

// Serve starts the HTTP and (optionally) SSH servers and blocks until cancelled.
func (a *App) Serve(ctx context.Context) error {
	handler := httptransport.NewServer(a.Deps).Router()
	httpSrv := &http.Server{
		Addr:              a.Cfg.Server.HTTP,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	errCh := make(chan error, 2)
	go func() {
		a.Log.Info("http server listening", "addr", a.Cfg.Server.HTTP)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	var sshSrv *sshdispatch.Server
	if a.Cfg.SSH.Enabled {
		builtin, err := sshdispatch.NewServer(sshdispatch.Options{
			Config:   a.Cfg,
			Log:      a.Log,
			Auth:     a.sshAuthorizer(),
			Dispatch: a.sshDispatcher(),
		})
		if err != nil {
			a.Log.Warn("ssh init failed, continuing with HTTP only", "error", err)
		} else {
			sshSrv = builtin
			go func() {
				if err := builtin.ListenAndServe(ctx); err != nil {
					a.Log.Warn("ssh server stopped", "error", err)
				}
			}()
		}
	} else {
		a.Log.Info("SSH server disabled by config")
	}
	_ = sshSrv

	sigCtx, stop := signal.NotifyContext(ctx, platform.SignalSet()...)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-sigCtx.Done():
		a.Log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
}

// Close releases the database handle.
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
