package update

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestChecker returns a Checker with all external dependencies mocked.
func newTestChecker(t *testing.T) *Checker {
	t.Helper()
	return &Checker{
		CachePath: filepath.Join(t.TempDir(), "cache.json"),
		Now:       func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) },
		HTTPGet: func(context.Context, string) ([]byte, error) {
			return []byte(`{"tag_name":"v1.2.3"}`), nil
		},
		Getenv:      func(string) string { return "" },
		StderrIsTTY: true,
	}
}

func TestCheck_AnnouncesWhenNewer(t *testing.T) {
	c := newTestChecker(t)
	notice, err := c.check(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notice == "" {
		t.Fatalf("expected notice, got empty")
	}
	if want := "1.2.3"; !contains(notice, want) {
		t.Errorf("notice should mention %q, got %q", want, notice)
	}
	if want := "1.0.0"; !contains(notice, want) {
		t.Errorf("notice should mention current version %q, got %q", want, notice)
	}
}

func TestCheck_SilentWhenEqual(t *testing.T) {
	c := newTestChecker(t)
	notice, err := c.check(context.Background(), "v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notice != "" {
		t.Errorf("expected no notice, got %q", notice)
	}
}

func TestCheck_SilentWhenOlder(t *testing.T) {
	c := newTestChecker(t)
	// current is newer than upstream — e.g. running a dev prerelease
	notice, _ := c.check(context.Background(), "v2.0.0")
	if notice != "" {
		t.Errorf("expected no notice when current > latest, got %q", notice)
	}
}

func TestCheck_SuppressedOnDevBuild(t *testing.T) {
	c := newTestChecker(t)
	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		t.Fatalf("HTTPGet must not be called on dev build")
		return nil, nil
	}
	notice, err := c.check(context.Background(), "dev")
	if err != nil || notice != "" {
		t.Errorf("dev build: expected silent no-error, got notice=%q err=%v", notice, err)
	}
}

func TestCheck_SuppressedWhenStderrNotTTY(t *testing.T) {
	c := newTestChecker(t)
	c.StderrIsTTY = false
	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		t.Fatalf("HTTPGet must not be called when stderr is not a TTY")
		return nil, nil
	}
	notice, _ := c.check(context.Background(), "v1.0.0")
	if notice != "" {
		t.Errorf("non-TTY: expected silent, got %q", notice)
	}
}

func TestCheck_SuppressedByEnvOptOut(t *testing.T) {
	c := newTestChecker(t)
	c.Getenv = func(k string) string {
		if k == "LLMWIKI_NO_UPDATE_CHECK" {
			return "1"
		}
		return ""
	}
	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		t.Fatalf("HTTPGet must not be called with opt-out env set")
		return nil, nil
	}
	notice, _ := c.check(context.Background(), "v1.0.0")
	if notice != "" {
		t.Errorf("opt-out env: expected silent, got %q", notice)
	}
}

func TestCheck_SuppressedInCI(t *testing.T) {
	c := newTestChecker(t)
	c.Getenv = func(k string) string {
		if k == "CI" {
			return "true"
		}
		return ""
	}
	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		t.Fatalf("HTTPGet must not be called when CI=true")
		return nil, nil
	}
	notice, _ := c.check(context.Background(), "v1.0.0")
	if notice != "" {
		t.Errorf("CI=true: expected silent, got %q", notice)
	}
}

func TestCheck_UsesCacheWithin24h(t *testing.T) {
	c := newTestChecker(t)
	// Seed cache from 1h ago with a newer version than what HTTPGet would
	// return — if the cache is used, we'll see that version in the notice.
	entry := cacheEntry{
		Latest:    "v1.1.0",
		CheckedAt: c.Now().Add(-1 * time.Hour),
	}
	data, _ := json.Marshal(entry)
	if err := os.WriteFile(c.CachePath, data, 0o600); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	httpCalls := 0
	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		httpCalls++
		return []byte(`{"tag_name":"v9.9.9"}`), nil
	}
	notice, _ := c.check(context.Background(), "v1.0.0")
	if httpCalls != 0 {
		t.Errorf("cache within TTL should skip HTTP, got %d call(s)", httpCalls)
	}
	if !contains(notice, "1.1.0") {
		t.Errorf("expected notice from cached v1.1.0, got %q", notice)
	}
}

func TestCheck_RefreshesCacheAfter24h(t *testing.T) {
	c := newTestChecker(t)
	// Seed cache from 25h ago
	entry := cacheEntry{
		Latest:    "v1.1.0",
		CheckedAt: c.Now().Add(-25 * time.Hour),
	}
	data, _ := json.Marshal(entry)
	_ = os.WriteFile(c.CachePath, data, 0o600)

	httpCalls := 0
	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		httpCalls++
		return []byte(`{"tag_name":"v2.0.0"}`), nil
	}
	notice, _ := c.check(context.Background(), "v1.0.0")
	if httpCalls != 1 {
		t.Errorf("expired cache should trigger one HTTP call, got %d", httpCalls)
	}
	if !contains(notice, "2.0.0") {
		t.Errorf("expected notice from fresh v2.0.0, got %q", notice)
	}
	// Cache should now hold the fresh value.
	fresh, ok := c.readCache()
	if !ok || fresh.Latest != "v2.0.0" {
		t.Errorf("cache should be refreshed to v2.0.0, got %+v ok=%v", fresh, ok)
	}
}

func TestCheck_NetworkFailureFallsBackToStaleCache(t *testing.T) {
	c := newTestChecker(t)
	entry := cacheEntry{
		Latest:    "v1.1.0",
		CheckedAt: c.Now().Add(-25 * time.Hour), // expired
	}
	data, _ := json.Marshal(entry)
	_ = os.WriteFile(c.CachePath, data, 0o600)

	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		return nil, errors.New("network down")
	}
	notice, err := c.check(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("network failure with stale cache should not surface error: %v", err)
	}
	if !contains(notice, "1.1.0") {
		t.Errorf("expected fallback to stale cache v1.1.0, got %q", notice)
	}
}

func TestCheck_NetworkFailureWithoutCacheIsSilent(t *testing.T) {
	c := newTestChecker(t)
	c.HTTPGet = func(context.Context, string) ([]byte, error) {
		return nil, errors.New("network down")
	}
	notice, _ := c.check(context.Background(), "v1.0.0")
	if notice != "" {
		t.Errorf("network failure w/o cache should be silent, got %q", notice)
	}
}

func TestCheckAsync_ChannelClosesAfterResult(t *testing.T) {
	c := newTestChecker(t)
	ch := c.CheckAsync(context.Background(), "v1.0.0")
	select {
	case got, ok := <-ch:
		if !ok {
			t.Fatal("channel closed without sending a value")
		}
		if !contains(got, "1.2.3") {
			t.Errorf("expected notice with v1.2.3, got %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CheckAsync should complete within timeout")
	}
	// Channel should be closed now.
	if _, ok := <-ch; ok {
		t.Error("expected channel to be closed after first receive")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.3", "v1.2.2", true},
		{"v1.2.3", "v1.2.3", false},
		{"v1.2.3", "v1.3.0", false},
		{"1.2.3", "1.2.2", true},     // missing v
		{"v2.0.0", "v1.9.9", true},
		{"v1.0.0", "dev", false},     // invalid semver → false
		{"garbage", "v1.0.0", false}, // invalid → false
	}
	for _, tc := range cases {
		got := isNewer(tc.latest, tc.current)
		if got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
}

// contains is a trivial substring helper to avoid pulling in testify.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
