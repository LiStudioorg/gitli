// SPDX-License-Identifier: AGPL-3.0
package http

import "github.com/LiStudioorg/gitli/internal/infra/secutil"

func validateName(kind, name string) error {
	return secutil.ValidateName(kind, name)
}
