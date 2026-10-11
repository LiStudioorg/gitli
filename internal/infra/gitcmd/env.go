// SPDX-License-Identifier: AGPL-3.0
package gitcmd

import "os"

func envSource() func(string) (string, bool) {
	return os.LookupEnv
}
