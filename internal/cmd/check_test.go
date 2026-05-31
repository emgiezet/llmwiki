package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emgiezet/llmwiki/internal/cmd"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCmd_noEntriesWithTracking(t *testing.T) {
	// Create a temp dir to act as the wiki root.
	wikiRoot := t.TempDir()

	// Write one .md file with no llmwiki_tracking block (via wiki.WriteProjectEntry).
	meta := wiki.ProjectMeta{
		Name:         "myproject",
		Customer:     "acme",
		Type:         "client",
		Path:         ".",
		LastIngested: time.Now().UTC(),
		// LLMWikiTracking intentionally left zero — no tracking block.
	}
	entryPath := filepath.Join(wikiRoot, "clients", "acme", "myproject.md")
	require.NoError(t, wiki.WriteProjectEntry(entryPath, meta, "\n## Summary\nA test project.\n"))

	// Build a root cobra command and add check to it.
	root := &cobra.Command{Use: "llmwiki"}
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.AddCommand(cmd.NewCheckCmd())

	// Capture stdout.
	var buf bytes.Buffer
	root.SetOut(&buf)

	root.SetArgs([]string{"check", ".", "--wiki-root", wikiRoot})
	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "not tracked", "expected 'not tracked' in output, got: %s", output)

	// Verify the file path appears in output.
	assert.Contains(t, output, "myproject.md")

	// Verify no stale entries were found (since entry has no tracking).
	assert.NotContains(t, output, "STALE")
}

func TestCheckCmd_jsonFlag(t *testing.T) {
	wikiRoot := t.TempDir()

	meta := wiki.ProjectMeta{
		Name:         "jsonproject",
		Customer:     "testcorp",
		Type:         "client",
		Path:         ".",
		LastIngested: time.Now().UTC(),
	}
	entryPath := filepath.Join(wikiRoot, "clients", "testcorp", "jsonproject.md")
	require.NoError(t, wiki.WriteProjectEntry(entryPath, meta, "\n## Summary\nJSON test.\n"))

	root := &cobra.Command{Use: "llmwiki"}
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.AddCommand(cmd.NewCheckCmd())

	var buf bytes.Buffer
	root.SetOut(&buf)

	root.SetArgs([]string{"check", ".", "--wiki-root", wikiRoot, "--json"})
	err := root.Execute()
	require.NoError(t, err)

	output := buf.String()
	// Should be valid JSON with expected fields.
	assert.Contains(t, output, `"project_dir"`)
	assert.Contains(t, output, `"entries"`)
	assert.Contains(t, output, `"any_stale"`)
	assert.Contains(t, output, `"not tracked"`)
}

func TestCheckCmd_skipIndexFile(t *testing.T) {
	wikiRoot := t.TempDir()

	// Write _index.md — should be skipped.
	indexPath := filepath.Join(wikiRoot, "_index.md")
	require.NoError(t, os.WriteFile(indexPath, []byte("---\nfoo: bar\n---\nindex content\n"), 0644))

	root := &cobra.Command{Use: "llmwiki"}
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.AddCommand(cmd.NewCheckCmd())

	var buf bytes.Buffer
	root.SetOut(&buf)

	root.SetArgs([]string{"check", ".", "--wiki-root", wikiRoot})
	err := root.Execute()
	require.NoError(t, err)

	// No entries found — output should indicate nothing to report.
	output := buf.String()
	assert.NotContains(t, output, "STALE")
}
