// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"encoding/json"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/transport/http/openapi"
)

// handleOpenAPI serves the generated OpenAPI 3.0 document.
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	base := s.deps.External.HTTPBase
	if base == "" {
		base = "http://" + r.Host
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(openapi.Spec(base))
}

// handleAPIDocs serves a self-contained Swagger UI page. Its CSP is relaxed to
// allow the Swagger UI CDN assets only for this page.
func (s *Server) handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; connect-src 'self'; object-src 'none'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>gitli API 文档</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: "/api/openapi.json",
    dom_id: "#swagger-ui",
    deepLinking: true,
    persistAuthorization: true
  });
};
</script>
</body>
</html>`
