// SPDX-License-Identifier: AGPL-3.0
package fsrepo

import (
	"fmt"
	"os"
	"path/filepath"
)

// KnownHooks are the git server-side hooks gitli installs.
var KnownHooks = []string{"pre-receive", "post-receive", "update"}

// InstallHooks writes executable hook scripts into repoPath/hooks. Each script
// invokes `gitli hook <name>` so the enforcement logic lives in the binary.
// configPath must be absolute so the hook works regardless of git's CWD.
func (l *Layout) InstallHooks(repoPath, owner, repo, gitliBin, configPath string) error {
	hooksDir := filepath.Join(repoPath, "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		return fmt.Errorf("fsrepo: mkdir hooks: %w", err)
	}
	for _, name := range KnownHooks {
		script := hookScript(name, owner, repo, gitliBin, configPath)
		path := filepath.Join(hooksDir, name)
		if err := os.WriteFile(path, []byte(script), 0o750); err != nil {
			return fmt.Errorf("fsrepo: write hook %s: %w", name, err)
		}
	}
	return nil
}

// RemoveHooks deletes gitli-managed hook scripts, leaving others intact.
func (l *Layout) RemoveHooks(repoPath string) error {
	for _, name := range KnownHooks {
		_ = os.Remove(filepath.Join(repoPath, "hooks", name))
	}
	return nil
}

func hookScript(name, owner, repo, gitliBin, configPath string) string {
	if gitliBin == "" {
		gitliBin = "gitli"
	}
	cfgArg := ""
	if configPath != "" {
		cfgArg = fmt.Sprintf(" --config %q", configPath)
	}
	return fmt.Sprintf(`#!/bin/sh
# Installed by gitli. Do not edit; managed automatically.
exec %q hook %s --owner %q --repo %q%s
`, gitliBin, name, owner, repo, cfgArg)
}
