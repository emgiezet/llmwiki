// Package safeio provides filesystem helpers that refuse to follow symlinks
// or operate on non-regular files, guarding against symlink TOCTOU attacks
// during recursive directory walks.
package safeio

import (
	"fmt"
	"os"
)

// ReadRegularFile reads path but refuses anything that isn't a regular
// file (symlinks, devices, pipes). Intended for WalkDir callbacks where
// an attacker could swap a regular file for a symlink mid-walk.
func ReadRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("refusing to read non-regular file %s (mode %s)", path, info.Mode())
	}
	return os.ReadFile(path)
}

// WriteRegularFile writes to path using WriteFile, but first lstats the
// target (if it exists) and refuses to overwrite anything that isn't a
// regular file. Use in WalkDir callbacks that update files in place.
func WriteRegularFile(path string, data []byte, perm os.FileMode) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to write to non-regular file %s (mode %s)", path, info.Mode())
		}
	}
	return os.WriteFile(path, data, perm)
}
