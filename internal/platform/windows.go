// SPDX-License-Identifier: AGPL-3.0
//go:build windows

package platform

import (
	"fmt"
	"os"
)

const supportsPerms = false

// IsRoot always reports false on Windows (no POSIX root concept).
func IsRoot() bool { return false }

// SignalSet returns the signals to watch; Windows only supports os.Interrupt.
func SignalSet() []os.Signal {
	return []os.Signal{os.Interrupt}
}

func checkDataDirPerm(dir string, want os.FileMode) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("platform: %s is not a directory", dir)
	}
	return nil
}

func serviceInstallHint() string {
	return "install the Windows Service with deploy/windows/install-service.ps1"
}
