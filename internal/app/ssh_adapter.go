// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	gitcmdinfra "github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/infra/secutil"
	sshdispatch "github.com/LiStudioorg/gitli/internal/transport/ssh"
)

func (a *App) sshAuthorizer() sshdispatch.PublicKeyAuthorizer {
	return &sshAuthorizer{a: a}
}

func (a *App) sshDispatcher() sshdispatch.CommandDispatcher {
	return &sshDispatcher{a: a}
}

type sshAuthorizer struct{ a *App }

func (s *sshAuthorizer) AuthorizeSSHKey(ctx context.Context, fingerprint string) (string, bool) {
	u, err := s.a.Deps.Users.UserBySSHFingerprint(ctx, fingerprint)
	if err != nil || u == nil {
		return "", false
	}
	return u.Username, true
}

func (s *sshAuthorizer) TouchSSHKey(ctx context.Context, fingerprint string) {
	_ = s.a.Deps.Users.TouchSSHKey(ctx, fingerprint)
}

// sshDispatcher runs git-upload-pack / git-receive-pack against a repository
// resolved from the requested path, validating containment under the repos root.
type sshDispatcher struct{ a *App }

var errInvalidCommand = errors.New("ssh: invalid git command")

func (s *sshDispatcher) Dispatch(ctx context.Context, username, command string, in sshdispatch.InputReader, out, errOut sshdispatch.OutputWriter) error {
	verb, repoRef, ok := splitGitCommand(command)
	if !ok {
		return errInvalidCommand
	}
	repoRef = strings.TrimPrefix(repoRef, "/")
	repoRef = strings.TrimSuffix(repoRef, ".git")
	owner, name := splitOwnerRepo(repoRef)
	if err := secutil.ValidateName("owner", owner); err != nil {
		return err
	}
	if err := secutil.ValidateName("repo", name); err != nil {
		return err
	}
	full := s.a.Deps.FS.Path(owner, name)
	validated, err := gitcmdinfra.ValidateRepoPath(s.a.Deps.FS.Root(), full)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, s.a.Deps.GitCmd.Path(), gitSubcommand(verb), validated)
	cmd.Env = s.a.Deps.GitCmd.Env()
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

// gitSubcommand converts a git-* server verb (e.g. "git-upload-pack") into the
// equivalent `git <subcommand>` form ("upload-pack").
func gitSubcommand(verb string) string {
	return strings.TrimPrefix(verb, "git-")
}

func splitGitCommand(command string) (verb, repo string, ok bool) {
	for _, v := range []string{"git-upload-pack", "git-receive-pack", "git-upload-archive"} {
		prefix := v + " "
		if strings.HasPrefix(command, prefix) {
			return v, command[len(prefix):], true
		}
	}
	return "", "", false
}

func splitOwnerRepo(ref string) (owner, name string) {
	if i := strings.IndexByte(ref, '/'); i >= 0 {
		return ref[:i], ref[i+1:]
	}
	return ref, ref
}
