package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/angelnicolasc/graymatter"
	"github.com/emgiezet/llmwiki/internal/config"
)

// openRealStore builds a Store backed by a real graymatter handle in dir.
// Embeddings are forced to keyword and consolidation is disabled so the test
// never probes Ollama or reaches for the Anthropic API — ExtractFacts already
// degrades to "store the raw text" when no API key is set.
func openRealStore(t *testing.T, dir string, mutate func(*graymatter.Config)) *Store {
	t.Helper()

	gmCfg := graymatter.DefaultConfig()
	gmCfg.DataDir = dir
	gmCfg.EmbeddingMode = graymatter.EmbeddingKeyword
	gmCfg.AsyncConsolidate = false
	gmCfg.ConsolidateLLM = ""
	gmCfg.AnthropicAPIKey = ""
	gmCfg.VectorReconcileInterval = 0
	gmCfg.StrictWrite = true
	if mutate != nil {
		mutate(&gmCfg)
	}

	mem, err := graymatter.NewWithConfig(gmCfg)
	if err != nil {
		t.Fatalf("open graymatter store in %s: %v", dir, err)
	}
	return &Store{mem: mem}
}

// TestOpenStore_StrictWriteDegradesOnHeldLock is the regression test for the
// graymatter v0.18.0 upgrade. From v0.18.0 a store whose write lock is held
// by another process (the graymatter daemon lives for 2 minutes after every
// CLI call) no longer fails to open — it degrades to read-only and every
// write returns ErrStoreReadOnly. openStore sets StrictWrite so the open
// fails loudly instead, and the existing lock branch turns that into a no-op
// store plus a warning.
func TestOpenStore_StrictWriteDegradesOnHeldLock(t *testing.T) {
	dir := t.TempDir()

	holder := openRealStore(t, dir, nil)
	defer func() { _ = holder.Close() }()

	cfg := config.Merged{MemoryEnabled: true, MemoryDir: dir}
	got, err := openStore(cfg, dir)
	if err != nil {
		t.Fatalf("openStore against a locked dir should degrade, not fail: %v", err)
	}
	if got.Enabled() {
		t.Fatal("openStore returned an enabled store while the write lock was held; " +
			"StrictWrite is not in effect and writes would be silently dropped")
	}
}

// TestEnabled_FalseForReadOnlyStore covers the second half of the same hazard:
// graymatter reports Healthy() == true for a read-only handle, so Enabled must
// consult IsReadOnly or llmwiki would report successful ingests that wrote
// nothing.
func TestEnabled_FalseForReadOnlyStore(t *testing.T) {
	dir := t.TempDir()

	// Create the store first: a read-only open of a non-existent gray.db fails.
	seed := openRealStore(t, dir, nil)
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed store: %v", err)
	}

	ro := openRealStore(t, dir, func(c *graymatter.Config) {
		c.StrictWrite = false
		c.ReadOnly = true
	})
	defer func() { _ = ro.Close() }()

	if !ro.mem.Healthy() {
		t.Fatal("precondition failed: graymatter should report a read-only store as healthy")
	}
	if ro.Enabled() {
		t.Error("Enabled() must be false for a read-only store")
	}
}

// TestRememberIngestion_PropagatesWriteError pins that write failures reach the
// caller. Before the v0.18.0 upgrade every write error was discarded with
// `_ =` and the function always returned nil, which made the requeue path in
// DrainAbsorbQueue unreachable.
func TestRememberIngestion_PropagatesWriteError(t *testing.T) {
	s := openRealStore(t, t.TempDir(), nil)

	// Closing the underlying bbolt handle leaves Memory.store non-nil, so the
	// store still reports healthy and writable while every write fails. This
	// is the shape of a transient write failure.
	if err := s.mem.Close(); err != nil {
		t.Fatalf("close underlying store: %v", err)
	}
	if !s.Enabled() {
		t.Fatal("precondition failed: store should still look enabled after the handle is closed")
	}

	err := s.RememberIngestion(context.Background(), "proj", "acme", "some wiki body", []string{"go"})
	if err == nil {
		t.Fatal("RememberIngestion swallowed a write error")
	}
}

// TestRememberServiceIngestion_PropagatesWriteError is the per-service twin.
func TestRememberServiceIngestion_PropagatesWriteError(t *testing.T) {
	s := openRealStore(t, t.TempDir(), nil)
	if err := s.mem.Close(); err != nil {
		t.Fatalf("close underlying store: %v", err)
	}

	err := s.RememberServiceIngestion(context.Background(), "proj", "svc", "acme", "body", []string{"go"})
	if err == nil {
		t.Fatal("RememberServiceIngestion swallowed a write error")
	}
}

// TestRememberIngestion_SucceedsOnWritableStore guards the other direction:
// the new error plumbing must not report failures on a healthy store.
func TestRememberIngestion_SucceedsOnWritableStore(t *testing.T) {
	s := openRealStore(t, t.TempDir(), nil)
	defer func() { _ = s.Close() }()

	if err := s.RememberIngestion(context.Background(), "proj", "acme", "body", []string{"go"}); err != nil {
		t.Fatalf("RememberIngestion on a writable store: %v", err)
	}

	facts, err := s.mem.Recall(context.Background(), projectAgent("proj"), "body")
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(facts) == 0 {
		t.Error("expected the ingestion to be recallable")
	}
}

// TestDrainAbsorbQueue_RequeuesOnWriteFailure exercises the requeue branch of
// DrainAbsorbQueue, which was dead code while RememberIngestion could not fail.
func TestDrainAbsorbQueue_RequeuesOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	s := openRealStore(t, dir, nil)
	if err := s.mem.Close(); err != nil {
		t.Fatalf("close underlying store: %v", err)
	}

	if err := QueueAbsorb(dir, QueuedAbsorb{
		Timestamp:   time.Unix(0, 0).UTC(),
		ProjectName: "proj",
		Content:     "queued body",
	}); err != nil {
		t.Fatalf("queue: %v", err)
	}

	res, err := DrainAbsorbQueue(context.Background(), dir, s)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if res.Processed != 0 {
		t.Errorf("Processed = %d, want 0 (every write fails)", res.Processed)
	}
	if res.Requeued != 1 {
		t.Errorf("Requeued = %d, want 1", res.Requeued)
	}

	data, err := os.ReadFile(filepath.Join(dir, queueFileName))
	if err != nil {
		t.Fatalf("queue file should have been rewritten: %v", err)
	}
	if !strings.Contains(string(data), "queued body") {
		t.Errorf("requeued entry lost its content: %s", data)
	}
}
