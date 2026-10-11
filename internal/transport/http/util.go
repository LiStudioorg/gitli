// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func isHTTPS(externalURL string) bool {
	return strings.HasPrefix(strings.ToLower(externalURL), "https://")
}
