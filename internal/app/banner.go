// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"log/slog"
	"runtime"
	"strings"

	"github.com/LiStudioorg/gitli/internal/version"
)

// printBanner logs the startup summary: version, runtime, paths and the
// detected git binary. Kept as structured attributes so JSON mode stays useful.
func (a *App) printBanner() {
	info := version.Get()
	mode := "prod"
	if a.Cfg.Log.Level == "debug" {
		mode = "dev"
	}
	gitVersion := "unknown"
	if a.Deps != nil && a.Deps.GitCmd != nil {
		if v, err := a.Deps.GitCmd.Run(context.Background(), "", "--version"); err == nil {
			gitVersion = strings.TrimSpace(v)
		}
	}
	a.Log.Info("gitli starting",
		slog.String("version", info.Version),
		slog.String("go", runtime.Version()),
		slog.String("mode", mode),
		slog.String("config", a.configPath),
		slog.String("data_dir", a.Cfg.Storage.Repos),
		slog.String("git", gitVersion),
	)
}
