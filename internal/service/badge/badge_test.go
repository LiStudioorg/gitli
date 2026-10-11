// SPDX-License-Identifier: AGPL-3.0
package badge

import (
	"strings"
	"testing"
)

func TestSVGContainsLabel(t *testing.T) {
	out := SVG("build", "passing", "#10b981")
	if !strings.Contains(out, "build") {
		t.Fatalf("SVG missing label: %s", out)
	}
	if !strings.Contains(out, "passing") {
		t.Fatalf("SVG missing message: %s", out)
	}
}

func TestSVGEscapesAngleBrackets(t *testing.T) {
	out := SVG("a<b", "x>y", "#000000")
	if strings.Contains(out, "<b") {
		t.Fatalf("SVG contains unescaped input: %s", out)
	}
	if !strings.Contains(out, "a&lt;b") {
		t.Fatalf("SVG missing escaped label: %s", out)
	}
	if !strings.Contains(out, "x&gt;y") {
		t.Fatalf("SVG missing escaped message: %s", out)
	}
}

func TestColorForStatus(t *testing.T) {
	cases := map[string]string{
		"success": "#10b981",
		"passed":  "#10b981",
		"failure": "#ef4444",
		"failed":  "#ef4444",
		"pending": "#f59e0b",
		"running": "#f59e0b",
		"other":   "#3b82f6",
		"":        "#3b82f6",
	}
	for status, want := range cases {
		if got := ColorForStatus(status); got != want {
			t.Errorf("ColorForStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestShieldJSON(t *testing.T) {
	b := ShieldJSON("coverage", "80%", "#3b82f6")
	if !strings.Contains(string(b), "coverage") {
		t.Fatalf("ShieldJSON output unexpected: %s", b)
	}
}
