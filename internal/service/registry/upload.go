// SPDX-License-Identifier: AGPL-3.0
package registry

import (
	"fmt"
	"os"
)

// openStaged opens a staged upload file for streaming into the blob store.
func openStaged(path string) (*os.File, error) {
	if path == "" {
		return nil, ErrNotFound
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("registry: open staged blob: %w", err)
	}
	return f, nil
}
