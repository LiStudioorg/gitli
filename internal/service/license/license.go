// SPDX-License-Identifier: AGPL-3.0
// Package license detects SPDX license identifiers from LICENSE files.
package license

import (
	"strings"

	"github.com/google/licensecheck"
)

// Detect identifies the most likely SPDX license id from file contents.
// It returns ("", false) when no license is recognized with enough coverage.
func Detect(data []byte) (string, bool) {
	cov := licensecheck.Scan(data)
	if cov.Percent < 75 {
		return "", false
	}
	for _, m := range cov.Match {
		if m.IsURL || m.ID == "" {
			continue
		}
		id := m.ID
		// licensecheck may return "Apache-2.0 OR MIT" style ids; take the first.
		if i := strings.Index(id, " "); i > 0 {
			id = id[:i]
		}
		return id, true
	}
	return "", false
}

// CandidatePaths lists the well-known LICENSE file locations.
var CandidatePaths = []string{
	"LICENSE", "LICENSE.md", "LICENSE.txt", "LICENCE", "LICENCE.md",
	"COPYING", "COPYING.md", "COPYING.txt",
}
