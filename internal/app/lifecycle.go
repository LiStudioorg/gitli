// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
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
	if err := a.DB.ApplyMigrations(ctx, migs); err != nil {
		return err
	}
	a.Log.Info("migrations applied", "count", len(migs), "dialect", a.DB.Dialect)
	return nil
}

// Serve starts all HTTP and (optionally) SSH listeners, blocking until cancelled.
func (a *App) Serve(ctx context.Context) error {
	handler := httptransport.NewServer(a.Deps).Router()

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
			if err := builtin.ListenAndServe(ctx); err != nil {
				a.Log.Warn("ssh server stopped", "error", err)
			}
		}
	} else {
		a.Log.Info("SSH server disabled by config")
	}
	_ = sshSrv

	sigCtx, stop := signal.NotifyContext(ctx, platform.SignalSet()...)
	defer stop()
	<-sigCtx.Done()

	a.Log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for _, srv := range httpServers {
		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()
			_ = s.Shutdown(shutdownCtx)
		}(srv)
	}
	wg.Wait()
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

func ipFamily(addr net.Addr) string {
	return netutil.Family(addr)
}

// Close releases the database handle.
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
