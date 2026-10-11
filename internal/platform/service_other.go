// SPDX-License-Identifier: AGPL-3.0
//go:build !linux && !darwin

package platform

// ServiceTemplate returns a generic service template hint for the platform.
func ServiceTemplate() string {
	return "# gitli is not a native service on this platform; run `gitli serve` under a process supervisor."
}
