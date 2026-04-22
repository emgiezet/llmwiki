package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQueueAbsorb_AppendsOneLine(t *testing.T) {
	dir := t.TempDir()
	if err := QueueAbsorb(dir, QueuedAbsorb{
		Timestamp:   time.Now(),
		ProjectName: "p",
		Customer:    "c",
		Content:     "hello",
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, queueFileName))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "\n"); n != 1 {
		t.Errorf("expected exactly one line, got %d", n)
	}
	var entry QueuedAbsorb
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry); err != nil {
		t.Errorf("line is not valid json: %v", err)
	}
	if entry.ProjectName != "p" {
		t.Errorf("round-trip lost field: %+v", entry)
	}
}

func TestQueueAbsorb_MultipleAppendsPreserveOrder(t *testing.T) {
	dir := t.TempDir()
	for i, name := range []string{"a", "b", "c"} {
		_ = i
		if err := QueueAbsorb(dir, QueuedAbsorb{ProjectName: name}); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(dir, queueFileName))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	want := []string{"a", "b", "c"}
	for i, line := range lines {
		var e QueuedAbsorb
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		if e.ProjectName != want[i] {
			t.Errorf("line %d: want %s, got %s", i, want[i], e.ProjectName)
		}
	}
}

// Drain against a nil / disabled store should be a no-op and not touch the file.
func TestDrainAbsorbQueue_DisabledStore(t *testing.T) {
	dir := t.TempDir()
	if err := QueueAbsorb(dir, QueuedAbsorb{ProjectName: "p"}); err != nil {
		t.Fatal(err)
	}
	res, err := DrainAbsorbQueue(context.TODO(), dir, &Store{})
	if err != nil {
		t.Errorf("drain disabled store returned error: %v", err)
	}
	if res.Processed != 0 {
		t.Errorf("disabled store should not process, got %d", res.Processed)
	}
	if _, err := os.Stat(filepath.Join(dir, queueFileName)); err != nil {
		t.Errorf("queue file should remain untouched: %v", err)
	}
}
