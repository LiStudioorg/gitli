// SPDX-License-Identifier: AGPL-3.0
//go:build windows

package platform

import "errors"

// DiskUsage is not implemented on Windows.
func DiskUsage(path string) (total, free uint64, err error) {
	return 0, 0, errors.New("platform: disk usage unsupported on windows")
}
