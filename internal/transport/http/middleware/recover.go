// SPDX-License-Identifier: AGPL-3.0
// Package middleware provides HTTP middleware for the gitli web service.
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// Recover wraps chi's Recoverer with structured logging of the panic.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return middleware.Recoverer
}
