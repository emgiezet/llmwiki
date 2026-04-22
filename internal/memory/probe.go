//go:build !windows

package memory

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// ErrLockBusy is returned by ProbeLock when the bbolt DB is already held
// by another process. Callers should treat this as a graceful skip, not
// a fatal error.
var ErrLockBusy = errors.New("memory database is busy (held by another process)")

// ProbeLock returns ErrLockBusy if the bbolt DB at dir/gray.db is currently
// held by another process. Non-blocking. Returns nil if the file does not
// exist yet (graymatter will create it) or if the lock is free at the moment
// of probing. Best-effort: between probe and actual open, another process
// could take the lock — but this matters only for the slow-path optimization
// of the Stop hook.
func ProbeLock(dir string) error {
	path := filepath.Join(dir, "gray.db")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return nil // any other stat error: let graymatter handle it
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		return nil
	}
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return ErrLockBusy
	}
	_ = syscall.Flock(fd, syscall.LOCK_UN)
	return nil
}
