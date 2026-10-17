// SPDX-License-Identifier: AGPL-3.0
package openapi

import (
	"encoding/json"
	"testing"
)

func TestSpecIsValidJSON(t *testing.T) {
	b, err := json.Marshal(Spec("https://git.example.com"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.0.3" {
		t.Fatalf("openapi version = %v", doc["openapi"])
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) < 20 {
		t.Fatalf("expected many paths, got %d", len(paths))
	}
	if _, ok := paths["/api/v1/version"]; !ok {
		t.Fatal("missing /api/v1/version path")
	}
	if _, ok := doc["components"]; !ok {
		t.Fatal("missing components")
	}
}
