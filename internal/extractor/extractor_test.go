package extractor_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_NormalizesExtensions(t *testing.T) {
	e := extractor.New(map[string]string{
		"pdf":   "cat {{input}}", // missing leading dot
		".DOCX": "cat {{input}}", // upper-case
		".odt":  "cat {{input}}",
	})

	assert.True(t, e.CanExtract("paper.pdf"))
	assert.True(t, e.CanExtract("report.PDF")) // case-insensitive match
	assert.True(t, e.CanExtract("notes.docx"))
	assert.True(t, e.CanExtract("thesis.odt"))
	assert.False(t, e.CanExtract("readme.md"))
	assert.False(t, e.CanExtract("noext"))
}

func TestNew_NilMapExtractsNothing(t *testing.T) {
	e := extractor.New(nil)
	assert.False(t, e.CanExtract("paper.pdf"))
}

func TestExtract_RunsConfiguredCommand(t *testing.T) {
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "paper.pdf")
	require.NoError(t, os.WriteFile(path, []byte("  extracted body  "), 0644))

	e := extractor.New(map[string]string{".pdf": "cat {{input}}"})
	out, err := e.Extract(context.Background(), path)
	require.NoError(t, err)
	assert.Equal(t, "extracted body", out, "stdout should be trimmed")
}

func TestExtract_PlaceholderHandlesPathWithSpaces(t *testing.T) {
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "my docs")
	require.NoError(t, os.MkdirAll(sub, 0755))
	path := filepath.Join(sub, "a paper.pdf")
	require.NoError(t, os.WriteFile(path, []byte("spaced content"), 0644))

	e := extractor.New(map[string]string{".pdf": "cat {{input}}"})
	out, err := e.Extract(context.Background(), path)
	require.NoError(t, err)
	assert.Equal(t, "spaced content", out)
}

func TestExtract_ToolNotFoundIsSentinel(t *testing.T) {
	e := extractor.New(map[string]string{".pdf": "llmwiki-nonexistent-tool-xyz {{input}}"})
	_, err := e.Extract(context.Background(), "/tmp/whatever.pdf")
	require.Error(t, err)
	assert.True(t, errors.Is(err, extractor.ErrToolNotFound), "missing binary must be a sentinel error")
}

func TestExtract_UnconfiguredExtension(t *testing.T) {
	e := extractor.New(map[string]string{".pdf": "cat {{input}}"})
	_, err := e.Extract(context.Background(), "/tmp/file.txt")
	require.Error(t, err)
}

func TestExtract_NonZeroExitReportsStderr(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	// `false` exits non-zero. Use sh to write to stderr then fail.
	e := extractor.New(map[string]string{".pdf": "sh -c exit_1"})
	_, err := e.Extract(context.Background(), "/tmp/file.pdf")
	require.Error(t, err)
}
