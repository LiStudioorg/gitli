// SPDX-License-Identifier: AGPL-3.0
// Package web embeds the compiled frontend and serves it as a static SPA.
package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the frontend assets rooted at dist/.
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// setAssetCacheHeaders gives content-hashed build output an immutable cache
// lifetime while keeping unhashed files revalidated.
func setAssetCacheHeaders(w http.ResponseWriter, path string) {
	if strings.HasPrefix(path, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}

// Mount registers static asset and SPA fallback handlers on the router.
func Mount(r chi.Router, log *slog.Logger) {
	sub, err := FS()
	if err != nil {
		log.Warn("frontend assets unavailable", "error", err)
		r.Get("/*", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "frontend not built", http.StatusServiceUnavailable)
		})
		return
	}
	fileServer := http.FileServer(http.FS(sub))
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := sub.Open(path); err == nil {
			_ = f.Close()
			setAssetCacheHeaders(w, path)
			fileServer.ServeHTTP(w, req)
			return
		}
		index, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// index.html must always revalidate so new deployments take effect.
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	}))
}
