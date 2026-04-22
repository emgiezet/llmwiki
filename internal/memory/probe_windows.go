//go:build windows

package memory

import "errors"

// ErrLockBusy is returned by ProbeLock when the bbolt DB is already held
// by another process.
var ErrLockBusy = errors.New("memory database is busy (held by another process)")

// ProbeLock is a best-effort no-op on Windows. bbolt uses LockFileEx which
// cannot be probed without actually opening the DB; callers will fall through
// to the normal open path.
func ProbeLock(dir string) error {
	return nil
}
