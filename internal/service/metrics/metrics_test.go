// SPDX-License-Identifier: AGPL-3.0
package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestIncAndSnapshot(t *testing.T) {
	r := New()
	r.Inc("http_requests")
	r.Inc("http_requests")
	r.Add("bytes_out", 512)

	snap := r.Snapshot()
	if snap["http_requests"] != 2 {
		t.Fatalf("http_requests = %d want 2", snap["http_requests"])
	}
	if snap["bytes_out"] != 512 {
		t.Fatalf("bytes_out = %d want 512", snap["bytes_out"])
	}
}

func TestSnapshotIsCopy(t *testing.T) {
	r := New()
	r.Inc("a")
	snap := r.Snapshot()
	snap["a"] = 99
	if r.Snapshot()["a"] != 1 {
		t.Fatal("Snapshot should return an independent copy")
	}
}

func TestUptime(t *testing.T) {
	r := New()
	time.Sleep(2 * time.Millisecond)
	if r.Uptime() <= 0 {
		t.Fatalf("uptime = %v want > 0", r.Uptime())
	}
}

func TestPrometheusOutput(t *testing.T) {
	r := New()
	r.Inc("repos_created")
	r.Add("pushes", 5)

	out := r.Prometheus()

	for _, want := range []string{
		"# TYPE gitli_repos_created counter",
		"gitli_repos_created 1",
		"# TYPE gitli_pushes counter",
		"gitli_pushes 5",
		"# TYPE gitli_uptime_seconds gauge",
		"gitli_uptime_seconds ",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("Prometheus output missing %q:\n%s", want, out)
		}
	}
}

func TestPromNameNormalization(t *testing.T) {
	r := New()
	r.Inc("http.requests-total")
	out := r.Prometheus()
	if !strings.Contains(out, "gitli_http_requests_total 1") {
		t.Fatalf("unexpected normalization:\n%s", out)
	}
}
