// Package fsutil provides filesystem primitives shared across infra:
// atomic writes, backups, path safety checks, and directory scans.
package fsutil

import (
	"errors"
	"path/filepath"
	"strings"
)

// EnsureWithin returns nil iff target is a subpath of root. It guards
// against zip-slip and similar path-escape attacks.
func EnsureWithin(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if strings.HasPrefix(rel, "..") || rel == ".." {
		return errors.New("fsutil: path escapes root")
	}
	return nil
}
