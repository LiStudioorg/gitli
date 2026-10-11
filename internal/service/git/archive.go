// SPDX-License-Identifier: AGPL-3.0
package git

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// ArchiveStream starts a git archive/bundle command and returns its stdout pipe
// along with a wait function. Callers must stream and then call wait.
func (s *Service) ArchiveStream(ctx context.Context, path, ref, format string) (io.ReadCloser, func() error, error) {
	if ref == "" {
		ref = "HEAD"
	}
	args, err := archiveArgs(ref, format)
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.CommandContext(ctx, s.runner.Path(), args...)
	cmd.Dir = path
	cmd.Env = s.runner.Env()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	wait := func() error { return cmd.Wait() }
	return stdout, wait, nil
}

func archiveArgs(ref, format string) ([]string, error) {
	switch format {
	case "zip":
		return []string{"archive", "--format=zip", "--prefix=", ref}, nil
	case "tar.gz", "targz", "tgz":
		return []string{"archive", "--format=tar.gz", ref}, nil
	case "bundle":
		return []string{"bundle", "create", "-", ref}, nil
	default:
		return nil, fmt.Errorf("git: unsupported archive format %q", format)
	}
}
