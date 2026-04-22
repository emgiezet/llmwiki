package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// QueuedAbsorb is one pending session absorption, serialized to the
// absorb-queue.jsonl file when the memory DB is held by another process.
type QueuedAbsorb struct {
	Timestamp   time.Time `json:"timestamp"`
	ProjectName string    `json:"project_name"`
	Customer    string    `json:"customer"`
	Content     string    `json:"content"`
}

const queueFileName = "absorb-queue.jsonl"

// QueueAbsorb appends an entry to <memoryDir>/absorb-queue.jsonl. The file
// is created with 0600 if missing. Appends are line-oriented; each call
// writes exactly one complete JSON line.
func QueueAbsorb(memoryDir string, entry QueuedAbsorb) error {
	if err := os.MkdirAll(memoryDir, 0o700); err != nil {
		return fmt.Errorf("queue dir: %w", err)
	}
	path := filepath.Join(memoryDir, queueFileName)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open queue: %w", err)
	}
	defer f.Close()
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal queue entry: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write queue: %w", err)
	}
	return nil
}

// DrainResult reports what the drain did.
type DrainResult struct {
	Processed int
	Requeued  int // entries that failed to apply and were written back
	Path      string
}

// DrainAbsorbQueue processes every entry in the queue file by calling
// store.RememberIngestion for each one. Uses atomic rename to move the
// queue to <queue>.inflight before reading, so new QueueAbsorb calls can
// keep appending in parallel without interference. Entries that fail to
// apply are written back to the queue (never silently dropped).
//
// Returns (DrainResult, nil) if the queue was processed (possibly with
// requeued failures). Returns an error only if the queue file existed but
// couldn't be moved or read.
func DrainAbsorbQueue(ctx context.Context, memoryDir string, store *Store) (DrainResult, error) {
	res := DrainResult{Path: filepath.Join(memoryDir, queueFileName)}
	if store == nil || !store.Enabled() {
		return res, nil
	}
	inflightPath := res.Path + ".inflight"

	// Atomically move queue → inflight. If queue doesn't exist, nothing to do.
	if err := os.Rename(res.Path, inflightPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return res, nil
		}
		return res, fmt.Errorf("move queue to inflight: %w", err)
	}

	f, err := os.Open(inflightPath)
	if err != nil {
		return res, fmt.Errorf("open inflight: %w", err)
	}
	defer f.Close()

	var requeue []QueuedAbsorb
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024) // allow up to 8 MiB lines
	for scanner.Scan() {
		var entry QueuedAbsorb
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// Malformed line: drop it after logging, do not re-queue.
			fmt.Fprintf(os.Stderr, "warning: drop malformed queue entry: %v\n", err)
			continue
		}
		if err := store.RememberIngestion(ctx, entry.ProjectName, entry.Customer, entry.Content, nil); err != nil {
			requeue = append(requeue, entry)
			continue
		}
		res.Processed++
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		// Partial read: re-queue what we haven't processed by leaving inflight.
		// Bubble the error so the caller logs it.
		return res, fmt.Errorf("scan inflight: %w", err)
	}
	_ = f.Close()

	// Successful drain: remove inflight. Requeue failed entries back to the
	// main queue so the next run can retry.
	if err := os.Remove(inflightPath); err != nil {
		return res, fmt.Errorf("remove inflight: %w", err)
	}
	for _, entry := range requeue {
		if qerr := QueueAbsorb(memoryDir, entry); qerr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not requeue failed entry for %s: %v\n", entry.ProjectName, qerr)
			continue
		}
		res.Requeued++
	}
	return res, nil
}
