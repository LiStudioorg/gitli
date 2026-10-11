// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"log/slog"
	"os"
	"strings"

	"github.com/LiStudioorg/gitli/internal/config"
)

// NewLogger builds a slog logger from configuration.
func NewLogger(cfg config.LogConfig) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}

func ensureDirs(cfg *config.Config) error {
	dirs := []string{
		cfg.Storage.Repos,
		cfg.Storage.LFS,
		cfg.Storage.Packages,
		cfg.Storage.Avatars,
		cfg.Storage.Attachments,
	}
	if cfg.Database.Driver == "sqlite" {
		if i := strings.LastIndex(cfg.Database.DSN, "/"); i > 0 {
			dirs = append(dirs, cfg.Database.DSN[:i])
		}
	}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if err := os.MkdirAll(d, 0o750); err != nil {
			return err
		}
	}
	return nil
}
