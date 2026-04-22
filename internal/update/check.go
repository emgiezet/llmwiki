// Package update implements the background version-check-and-notify flow.
//
// Design goals:
//   - Never block the user-visible command. The check runs in a goroutine;
//     the caller drains a buffered channel non-blockingly after the subcommand
//     finishes. If the HTTP call is still in flight at that point, the notice
//     is deferred to the next invocation.
//   - Respect a 24h cache so that repeated invocations do not spam the GitHub
//     API. Cache lives at ~/.llmwiki/last_update_check.json.
//   - Stay silent when it would be noise: dev builds, CI, non-TTY output, or
//     an explicit opt-out via $LLMWIKI_NO_UPDATE_CHECK=1.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// LatestReleaseURL is the GitHub Releases endpoint the checker polls.
	LatestReleaseURL = "https://api.github.com/repos/emgiezet/llmwiki/releases/latest"
	// cacheTTL is how long a successful lookup is trusted before we hit the
	// network again.
	cacheTTL = 24 * time.Hour
	// httpTimeout bounds any network call the checker makes.
	httpTimeout = 2 * time.Second
)

// Checker encapsulates the dependencies that vary between production and
// tests: wall clock, HTTP client, env lookup, TTY detection, and the cache
// path. Production callers should use NewChecker().
type Checker struct {
	CachePath   string
	Now         func() time.Time
	HTTPGet     func(ctx context.Context, url string) ([]byte, error)
	Getenv      func(string) string
	StderrIsTTY bool
}

// NewChecker returns a Checker wired up with real clock, HTTP, env, and TTY
// detection. CachePath defaults to ~/.llmwiki/last_update_check.json.
func NewChecker() *Checker {
	home, _ := os.UserHomeDir()
	return &Checker{
		CachePath:   filepath.Join(home, ".llmwiki", "last_update_check.json"),
		Now:         time.Now,
		HTTPGet:     defaultHTTPGet,
		Getenv:      os.Getenv,
		StderrIsTTY: isTerminal(os.Stderr),
	}
}

// cacheEntry is the on-disk schema for the check cache.
type cacheEntry struct {
	Latest    string    `json:"latest"`
	CheckedAt time.Time `json:"checked_at"`
}

// CheckAsync kicks off the version check in a background goroutine. It
// returns a buffered channel of size 1 that will receive the announcement
// string (or empty string for "no notice") when the check completes.
//
// Callers should drain the channel with a non-blocking select after their
// main work is done. If nothing has been sent yet, the notice is simply
// deferred — the goroutine cannot leak because the channel is buffered.
func (c *Checker) CheckAsync(ctx context.Context, currentVersion string) <-chan string {
	out := make(chan string, 1)
	go func() {
		defer close(out)
		notice, _ := c.check(ctx, currentVersion)
		out <- notice
	}()
	return out
}

// check performs the full check flow and returns the notice to print
// (empty string if none). The error return is intentionally swallowed by
// CheckAsync — surfaces are available to tests.
func (c *Checker) check(ctx context.Context, currentVersion string) (string, error) {
	if !c.shouldCheck(currentVersion) {
		return "", nil
	}

	latest, err := c.resolveLatest(ctx)
	if err != nil || latest == "" {
		return "", err
	}

	if !isNewer(latest, currentVersion) {
		return "", nil
	}
	return fmt.Sprintf("llmwiki %s available (you have %s) — run 'llmwiki update' to install.",
		stripV(latest), stripV(currentVersion)), nil
}

// shouldCheck applies the suppression rules that decide whether a check runs
// at all. Anything true here returns false (= skip).
func (c *Checker) shouldCheck(currentVersion string) bool {
	if currentVersion == "" || currentVersion == "dev" {
		return false
	}
	if !c.StderrIsTTY {
		return false
	}
	if c.Getenv("LLMWIKI_NO_UPDATE_CHECK") == "1" {
		return false
	}
	if strings.EqualFold(c.Getenv("CI"), "true") {
		return false
	}
	return true
}

// resolveLatest returns the latest released version tag (e.g. "v1.2.3"),
// consulting the 24h cache before hitting the network.
func (c *Checker) resolveLatest(ctx context.Context) (string, error) {
	if cached, ok := c.readCache(); ok {
		if c.Now().Sub(cached.CheckedAt) < cacheTTL {
			return cached.Latest, nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	latest, err := c.fetchLatest(ctx)
	if err != nil {
		// Fall back to a stale cached entry if available — better a stale
		// version than an empty one, and we preserve the "silent on error"
		// promise by returning nil error.
		if cached, ok := c.readCache(); ok {
			return cached.Latest, nil
		}
		return "", err
	}
	_ = c.writeCache(cacheEntry{Latest: latest, CheckedAt: c.Now()})
	return latest, nil
}

// fetchLatest calls the GitHub Releases API and parses tag_name out of the
// JSON response. Uses only the fields we need so the schema can evolve.
func (c *Checker) fetchLatest(ctx context.Context) (string, error) {
	body, err := c.HTTPGet(ctx, LatestReleaseURL)
	if err != nil {
		return "", err
	}
	var resp struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse release JSON: %w", err)
	}
	if resp.TagName == "" {
		return "", errors.New("release JSON missing tag_name")
	}
	return resp.TagName, nil
}

// readCache loads the cache file; returns ok=false for any error, including
// missing file.
func (c *Checker) readCache() (cacheEntry, bool) {
	data, err := os.ReadFile(c.CachePath)
	if err != nil {
		return cacheEntry{}, false
	}
	var e cacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return cacheEntry{}, false
	}
	return e, true
}

// writeCache persists the cache entry. Directory is created if absent.
// Errors are reported but non-fatal to the calling flow.
func (c *Checker) writeCache(e cacheEntry) error {
	if err := os.MkdirAll(filepath.Dir(c.CachePath), 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return os.WriteFile(c.CachePath, data, 0o600)
}

// isNewer reports whether latest > current using semver ordering.
// Both tags may include or omit the leading 'v'.
func isNewer(latest, current string) bool {
	l := ensureV(latest)
	c := ensureV(current)
	if !semver.IsValid(l) || !semver.IsValid(c) {
		return false
	}
	return semver.Compare(l, c) > 0
}

func ensureV(s string) string {
	if strings.HasPrefix(s, "v") {
		return s
	}
	return "v" + s
}

// stripV removes the leading 'v' for display. "v1.2.3" → "1.2.3".
func stripV(s string) string {
	return strings.TrimPrefix(s, "v")
}

// defaultHTTPGet is the production HTTPGet implementation.
func defaultHTTPGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB ceiling
}
