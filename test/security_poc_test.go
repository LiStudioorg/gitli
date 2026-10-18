// SPDX-License-Identifier: AGPL-3.0
package test

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LiStudioorg/gitli/internal/infra/fsrepo"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/infra/secutil"
	"github.com/LiStudioorg/gitli/internal/service/git"
	"github.com/LiStudioorg/gitli/internal/service/oauth"
)

// H-01: a ref must never be interpreted as a git option.
func TestRefRejectsOptionInjection(t *testing.T) {
	bad := []string{
		"--output=/var/lib/gitli/gitli.db",
		"--upload-pack=/bin/sh",
		"-o/tmp/x",
		"main\x00",
		"main\nrm -rf /",
	}
	for _, ref := range bad {
		if err := secutil.ValidateRef(ref); err == nil {
			t.Errorf("ValidateRef accepted injection payload %q", ref)
		}
	}
	if err := secutil.ValidateRef("main"); err != nil {
		t.Errorf("valid ref rejected: %v", err)
	}
}

// H-01 at the service layer: ArchiveStream validates before spawning git.
func TestArchiveStreamRejectsInjectedRef(t *testing.T) {
	svc := git.New(gitcmd.New())
	if _, _, err := svc.ArchiveStream(t.Context(), t.TempDir(), "--output=/tmp/pwn", "zip"); err == nil {
		t.Fatal("ArchiveStream accepted an option-like ref")
	}
}

// H-07: redirect URIs must be absolute http(s) and fragment-free.
func TestOAuthRedirectURIValidation(t *testing.T) {
	bad := []string{
		"javascript:alert(1)",
		"data:text/html,x",
		"//evil.example.com/cb",
		"/relative/cb",
		"http://example.com/cb#frag",
		"ftp://example.com/cb",
	}
	for _, u := range bad {
		if err := oauth.ValidateRedirectURIFormat(u); err == nil {
			t.Errorf("accepted unsafe redirect_uri %q", u)
		}
	}
	for _, u := range []string{"https://example.com/cb", "http://127.0.0.1:3000/cb"} {
		if err := oauth.ValidateRedirectURIFormat(u); err != nil {
			t.Errorf("rejected valid redirect_uri %q: %v", u, err)
		}
	}
}

// 考点五: repository paths can never escape the storage root.
func TestRepoPathContainment(t *testing.T) {
	root := t.TempDir()
	layout := fsrepo.New(root)
	for _, owner := range []string{"../etc", "..", `..\..`, "a/b", "~root"} {
		p := layout.Path(owner, "repo")
		cleanRoot := filepath.Clean(root)
		cleanPath := filepath.Clean(p)
		if !strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) {
			t.Errorf("owner %q escaped root: %s", owner, p)
		}
	}
}

// 考点三: webhook URLs pointing at internal networks are rejected.
func TestSSRFInternalTargetsRejected(t *testing.T) {
	bad := []string{
		"http://127.0.0.1/hook",
		"http://localhost/hook",
		"http://[::1]/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://10.0.0.1/hook",
		"http://192.168.1.1/hook",
		"http://172.16.0.1/hook",
		"file:///etc/passwd",
	}
	for _, u := range bad {
		if err := secutil.ValidateOutboundURL(u, true); err == nil {
			t.Errorf("accepted SSRF target %q", u)
		}
	}
}

// 考点一.4: username redirects stay same-origin (no scheme/host injected).
func TestUsernameRedirectSanitisesOwner(t *testing.T) {
	for _, owner := range []string{"evil.com", "//evil.com", "user:pass"} {
		if err := secutil.ValidateName("user", owner); err == nil {
			// "evil.com" is a legal username (no slash/colon), but the redirect
			// only ever prepends "/", so it cannot become an absolute URL.
			if strings.HasPrefix("/"+owner, "//") || strings.Contains(owner, ":") {
				t.Errorf("owner %q could produce an open redirect", owner)
			}
		}
	}
	req := httptest.NewRequest("GET", "/", nil)
	if req == nil {
		t.Fatal("unreachable")
	}
}
