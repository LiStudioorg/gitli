// SPDX-License-Identifier: AGPL-3.0
package version

// Info describes the build version metadata.
type Info struct {
	Version string
	Commit  string
	Date    string
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Get returns the current build version metadata.
func Get() Info {
	return Info{Version: version, Commit: commit, Date: date}
}

// String returns a human readable version line.
func String() string {
	return version
}
