// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/netutil"
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
	applied, err := a.DB.ApplyMigrations(ctx, migs)
	if err != nil {
		return err
	}
	if applied > 0 {
		a.Log.Info("database migrated", "applied", applied, "available", len(migs), "dialect", a.DB.Dialect)
	} else {
		a.Log.Info("database schema up to date", "version", len(migs), "dialect", a.DB.Dialect)
	}
	if a.Deps != nil && a.Deps.RBAC != nil {
		if err := a.Deps.RBAC.Seed(ctx); err != nil {
			return fmt.Errorf("seed rbac: %w", err)
		}
	}
	return nil
}

// Serve starts all HTTP and (optionally) SSH listeners, blocking until cancelled.
func (a *App) Serve(ctx context.Context) error {
	// Fresh installs must have their schema before any seed or query runs.
	if err := a.Migrate(ctx); err != nil {
		a.Log.Error("database migration failed, refusing to start", "error", err)
		return fmt.Errorf("migrate: %w", err)
	}
	if err := httptransport.SetTrustedProxies(a.Cfg.Server.TrustedProxies); err != nil {
		return err
	}
	a.printBanner()
	handler := httptransport.NewServer(a.Deps).Router()

	if a.Cfg.Runner.Enabled && a.Deps.Runners != nil {
		go a.runnerOfflineSweeper(ctx)
	}
	if a.Deps.Audit != nil && a.Cfg.Audit.RetentionDays > 0 {
		go a.runAuditRetention(ctx)
	}
	if a.Deps.Repos != nil {
		go a.runRepoCleanup(ctx)
	}
	if a.Deps.Explore != nil {
		go a.runTrendingAggregator(ctx)
	}
	if a.Cfg.Mail.Inbound.Enabled {
		go a.runInboundMailer(ctx)
	}
	if a.Deps.Digest != nil {
		go a.runDigestScheduler(ctx)
	}

	httpServers, err := a.startHTTP(ctx, handler)
	if err != nil {
		return err
	}

	var sshSrv *sshdispatch.Server
	if a.Cfg.Server.SSHEnabled {
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
			// Run in a goroutine: ListenAndServe blocks until ctx is cancelled,
			// and calling it inline would prevent signal handling from ever
			// being registered.
			go func(srv *sshdispatch.Server) {
				if err := srv.ListenAndServe(ctx); err != nil {
					a.Log.Warn("ssh server stopped", "error", err)
				}
			}(builtin)
		}
	} else {
		a.Log.Info("SSH server disabled by config")
	}
	_ = sshSrv

	sigCtx, stop := signal.NotifyContext(ctx, platform.SignalSet()...)
	defer stop()
	<-sigCtx.Done()

	a.Log.Warn(fmt.Sprintf("PID %d. Received signal. Shutting down gracefully...", os.Getpid()))

	// Give in-flight requests (large clones/pushes) up to 10s to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for _, srv := range httpServers {
		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()
			if err := s.Shutdown(shutdownCtx); err != nil {
				a.Log.Warn("HTTP listener shutdown incomplete", "error", err)
			}
			a.Log.Info("HTTP Listener: " + s.Addr + " Closed")
		}(srv)
	}
	wg.Wait()

	if sshSrv != nil {
		// The SSH server exits when its context is cancelled; the signal
		// context above already triggered that.
		a.Log.Info("SSH Listener: " + strings.Join(a.Cfg.Server.SSHListen, ",") + " Closed")
	}

	a.Log.Info("Background workers stopped")
	a.Log.Info(fmt.Sprintf("PID %d Gitli Web Finished", os.Getpid()))
	return nil
}

// startHTTP opens one listener per configured address and serves the shared handler.
func (a *App) startHTTP(ctx context.Context, handler http.Handler) ([]*http.Server, error) {
	listens := a.Cfg.Server.HTTPListen
	if len(listens) == 0 {
		listens = []string{"0.0.0.0:8080"}
	}
	servers := make([]*http.Server, 0, len(listens))
	for _, addr := range listens {
		ln, err := net.Listen(netutil.NetworkFor(addr), addr)
		if err != nil {
			return nil, fmt.Errorf("listen %s: %w", addr, err)
		}
		srv := &http.Server{
			Handler:           handler,
			Addr:              ln.Addr().String(),
			ReadHeaderTimeout: 15 * time.Second,
		}
		servers = append(servers, srv)
		family := netutil.Family(ln.Addr())
		a.Log.Info("HTTP listening", "addr", ln.Addr().String(), "family", family)
		go func(s *http.Server, l net.Listener) {
			if err := s.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
				a.Log.Error("http serve error", "error", err)
			}
		}(srv, ln)
	}
	_ = ctx
	return servers, nil
}

// Close releases the queue and database handle.
func (a *App) Close() error {
	if a.Deps != nil && a.Deps.Queue != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		a.Deps.Queue.Shutdown(ctx)
		cancel()
	}
	if a.Deps != nil && a.Deps.Audit != nil {
		a.Deps.Audit.Stop()
	}
	if a.DB != nil {
		err := a.DB.Close()
		a.Log.Info("Database connection pool closed")
		return err
	}
	return nil
}
