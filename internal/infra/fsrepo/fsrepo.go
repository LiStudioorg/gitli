// SPDX-License-Identifier: AGPL-3.0
// Package fsrepo resolves and manages on-disk Git repositories.
package fsrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// Layout maps owners and repositories to filesystem paths.
type Layout struct {
	root string
}

// New creates a Layout rooted at root.
func New(root string) *Layout {
	return &Layout{root: root}
}

// Root returns the configured root directory.
func (l *Layout) Root() string { return l.root }

// Path returns the bare repository path for owner/name.
func (l *Layout) Path(owner, name string) string {
	return filepath.Join(l.root, sanitize(owner), sanitize(name)+".git")
}

// OwnerDir returns the directory that holds an owner's repositories.
func (l *Layout) OwnerDir(owner string) string {
	return filepath.Join(l.root, sanitize(owner))
}

// Exists reports whether a repository directory exists.
func (l *Layout) Exists(owner, name string) bool {
	info, err := os.Stat(l.Path(owner, name))
	return err == nil && info.IsDir()
}

// Ensure creates the owner directory and returns the repository path.
func (l *Layout) Ensure(owner, name string) (string, error) {
	if err := os.MkdirAll(l.OwnerDir(owner), 0o750); err != nil {
		return "", fmt.Errorf("fsrepo: mkdir owner: %w", err)
	}
	return l.Path(owner, name), nil
}

// Remove deletes the repository directory.
func (l *Layout) Remove(owner, name string) error {
	return os.RemoveAll(l.Path(owner, name))
}

// Move relocates a repository when its owner or name changes.
func (l *Layout) Move(oldOwner, oldName, newOwner, newName string) error {
	src := l.Path(oldOwner, oldName)
	dst := l.Path(newOwner, newName)
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("fsrepo: mkdir dst owner: %w", err)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("fsrepo: move: %w", err)
	}
	return nil
}

// ResolveWithinRoot validates that p stays inside the root, guarding traversal.
func (l *Layout) ResolveWithinRoot(p string) (string, error) {
	clean := filepath.Clean(p)
	rootClean := filepath.Clean(l.root)
	if clean != rootClean && !strings.HasPrefix(clean, rootClean+string(os.PathSeparator)) {
		return "", fmt.Errorf("fsrepo: path escapes root: %s", p)
	}
	return clean, nil
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "..", "-")
	if s == "" {
		return "_"
	}
	return unsafeName.ReplaceAllString(s, "-")
}
