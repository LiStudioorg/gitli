// SPDX-License-Identifier: AGPL-3.0
package git

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

// smartService names the two smart HTTP services git clients request.
const (
	ServiceUploadPack  = "git-upload-pack"
	ServiceReceivePack = "git-receive-pack"
)

// IsSmartService reports whether name is a supported smart HTTP service.
func IsSmartService(name string) bool {
	return name == ServiceUploadPack || name == ServiceReceivePack
}

// ServeSmartHTTP handles a smart HTTP info/refs or service request. It streams
// the client body to the git subprocess and the subprocess output back to the
// client without buffering. contentType should be "application/x-git-*-advertisement"
// for the info/refs phase or "application/x-git-*-result" for the service phase.
func (s *Service) ServeSmartHTTP(ctx context.Context, repoPath, service, contentType string, r io.Reader, w io.Writer) error {
	return s.ServeSmartHTTPWithEnv(ctx, repoPath, service, contentType, r, w, nil)
}

// ServeSmartHTTPWithEnv is ServeSmartHTTP with extra hook environment, such as
// the authenticated actor used for activity attribution.
func (s *Service) ServeSmartHTTPWithEnv(ctx context.Context, repoPath, service, contentType string, r io.Reader, w io.Writer, extraEnv map[string]string) error {
	if !IsSmartService(service) {
		return fmt.Errorf("git: unsupported service %q", service)
	}
	// git exposes these as `git upload-pack` / `git receive-pack` subcommands
	// (not `git-upload-pack`).
	sub := strings.TrimPrefix(service, "git-")
	args := []string{sub, "--stateless-rpc"}
	if contentType != "" && strings.Contains(contentType, "-advertisement") {
		args = append(args, "--advertise-refs")
	}
	args = append(args, ".")
	runner := s.runner
	if len(extraEnv) > 0 {
		runner = runner.WithEnv(extraEnv)
	}
	cmd := exec.CommandContext(ctx, runner.Path(), args...)
	cmd.Dir = repoPath
	cmd.Env = runner.Env()
	cmd.Stdin = r
	cmd.Stdout = w
	var errBuf strings.Builder
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", sub, err, errBuf.String())
	}
	return nil
}

// InfoRefsContentType returns the advertisement content type for a service.
func InfoRefsContentType(service string) string {
	return "application/x-" + service + "-advertisement"
}

// ResultContentType returns the result content type for a service.
func ResultContentType(service string) string {
	return "application/x-" + service + "-result"
}

// WritePacketLinePrefix writes the smart-HTTP service discovery preamble.
func WritePacketLinePrefix(w io.Writer, service string) error {
	line := fmt.Sprintf("# service=%s\n", service)
	plen := len(line) + 4
	prefix := fmt.Sprintf("%04x", plen)
	if _, err := io.WriteString(w, prefix+line+"0000"); err != nil {
		return err
	}
	return nil
}

var _ = http.StatusOK
