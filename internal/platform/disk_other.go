// SPDX-License-Identifier: AGPL-3.0
//go:build !linux && !darwin && !windows

package platform

import "errors"

// DiskUsage is not implemented on this platform.
func DiskUsage(path string) (total, free uint64, err error) {
	return 0, 0, errors.New("platform: disk usage unsupported on this platform")
}
