// SPDX-License-Identifier: AGPL-3.0
package version

// Info describes the build version metadata.
type Info struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "source"
)

// Get returns the current build version metadata.
func Get() Info {
	return Info{Version: version, Commit: commit, Date: date, BuiltBy: builtBy}
}

// String returns the raw version string.
func String() string {
	return version
}
