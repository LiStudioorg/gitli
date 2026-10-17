// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

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
	var out io.Writer = os.Stderr
	if cfg.File != "" {
		maxSize := cfg.MaxSizeMB
		if maxSize <= 0 {
			maxSize = 100
		}
		backups := cfg.MaxBackups
		if backups <= 0 {
			backups = 5
		}
		age := cfg.MaxAgeDays
		if age <= 0 {
			age = 30
		}
		out = &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    maxSize,
			MaxBackups: backups,
			MaxAge:     age,
			Compress:   cfg.Compress,
		}
	}
	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
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
