package memory

import (
	"testing"
	"time"

	"github.com/mgz/llmwiki/internal/config"
)

// TestClose_NoDeadlock verifies that Store.Close returns within a reasonable
// timeout even when memory is not enabled (no-op path).
func TestClose_NoDeadlock(t *testing.T) {
	t.Parallel()

	s := &Store{}
	done := make(chan error, 1)
	go func() {
		done <- s.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close() returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close() did not return within 5 seconds (deadlock?)")
	}
}

// TestClose_NilStore ensures Close() on a nil *Store is safe.
func TestClose_NilStore(t *testing.T) {
	t.Parallel()

	var s *Store
	if err := s.Close(); err != nil {
		t.Errorf("nil Store.Close() returned error: %v", err)
	}
}

// TestNewFromConfig_Disabled verifies that a disabled-memory config returns
// a no-op store with no error.
func TestNewFromConfig_Disabled(t *testing.T) {
	t.Parallel()

	cfg := config.Merged{
		MemoryEnabled: false,
		MemoryDir:     t.TempDir(),
	}
	store, err := NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewFromConfig with disabled memory returned error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.Enabled() {
		t.Error("store should not be enabled when MemoryEnabled=false")
	}
	// Close on a no-op store must return quickly.
	done := make(chan error, 1)
	go func() { done <- store.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close() on disabled store returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close() on disabled store did not return within 5 seconds")
	}
}
