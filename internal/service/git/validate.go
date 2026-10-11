// SPDX-License-Identifier: AGPL-3.0
package git

import "github.com/LiStudioorg/gitli/internal/infra/secutil"

// validateRef rejects ref/path arguments that could be interpreted as git
// options (leading '-') or contain control characters.
func validateRef(ref string) error {
	return secutil.ValidateRef(ref)
}

// validateSubPath rejects option-like or unsafe path arguments.
func validateSubPath(p string) error {
	return secutil.ValidateSubPath(p)
}
