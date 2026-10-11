// SPDX-License-Identifier: AGPL-3.0
//go:build !windows

package platform

import (
	"fmt"
	"os"
	"syscall"
)

const supportsPerms = true

// IsRoot reports whether the current process runs as root.
func IsRoot() bool {
	return os.Getuid() == 0
}

// SignalSet returns the OS signals to watch for graceful shutdown.
func SignalSet() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

func checkDataDirPerm(dir string, want os.FileMode) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("platform: %s is not a directory", dir)
	}
	got := info.Mode().Perm()
	if got&^want != 0 {
		return fmt.Errorf("platform: %s has mode %04o, expected no bits outside %04o; run `chmod -R go-rwx`", dir, got, want)
	}
	return nil
}

func serviceInstallHint() string {
	return "install the systemd unit from deploy/systemd/gitli.service (Linux) or the launchd plist from deploy/launchd (macOS)"
}
