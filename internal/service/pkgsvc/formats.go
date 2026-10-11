// SPDX-License-Identifier: AGPL-3.0
package pkgsvc

import (
	"errors"
	"strings"
)

// NPMTarballURL documents the canonical NPM tarball path for a package.
func NPMTarballURL(name, version string) string {
	return name + "/-/" + name + "-" + version + ".tgz"
}

// GoModulePath returns the module path for an owner/repo pair.
func GoModulePath(owner, repo string) string {
	return owner + "/" + repo
}

// MavenPath returns the Maven repository layout path for an artifact.
func MavenPath(groupID, artifactID, version, filename string) string {
	group := strings.ReplaceAll(groupID, ".", "/")
	return group + "/" + artifactID + "/" + version + "/" + filename
}

// GenericPath returns the generic registry layout path for a file.
func GenericPath(pkgName, version, filename string) string {
	return pkgName + "/" + version + "/" + filename
}

// ValidateVersion rejects empty versions and values containing path
// separators or parent-directory references.
func ValidateVersion(v string) error {
	if v == "" {
		return errors.New("pkgsvc: empty version")
	}
	if strings.Contains(v, "/") || strings.Contains(v, "..") {
		return errors.New("pkgsvc: invalid version")
	}
	return nil
}
