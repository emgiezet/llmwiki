//go:build !windows

package memory

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestProbeLock_NoFile(t *testing.T) {
	if err := ProbeLock(t.TempDir()); err != nil {
		t.Errorf("expected nil for missing file, got %v", err)
	}
}

func TestProbeLock_Free(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gray.db")
	if err := os.WriteFile(path, []byte("dummy"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ProbeLock(dir); err != nil {
		t.Errorf("expected nil for free lock, got %v", err)
	}
}

func TestProbeLock_Busy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gray.db")
	if err := os.WriteFile(path, []byte("dummy"), 0o600); err != nil {
		t.Fatal(err)
	}
	fd, err := syscall.Open(path, syscall.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	if err := ProbeLock(dir); !errors.Is(err, ErrLockBusy) {
		t.Errorf("expected ErrLockBusy, got %v", err)
	}
}
