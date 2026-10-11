// SPDX-License-Identifier: AGPL-3.0
// Package platform abstracts OS-specific behavior behind build-tagged files.
package platform

import "os"

// Platform describes compile-time and runtime OS capabilities.
type Platform struct {
	GOOS   string
	GOARCH string
	// SupportsPerms reports whether POSIX permission bits are enforced.
	SupportsPerms bool
}

// Current returns the platform descriptor for the running binary.
func Current() Platform {
	return Platform{
		GOOS:          goos,
		GOARCH:        goarch,
		SupportsPerms: supportsPerms,
	}
}

// CheckDataDirPerm verifies that dir has permissions no looser than want.
// Platforms without POSIX permissions return nil after checking existence.
func CheckDataDirPerm(dir string, want os.FileMode) error {
	return checkDataDirPerm(dir, want)
}

// ServiceInstallHint returns a human-readable hint for installing a service.
func ServiceInstallHint() string {
	return serviceInstallHint()
}
