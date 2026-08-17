package ingestion_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/ingestion"
	"github.com/emgiezet/llmwiki/internal/llm"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// knowledgeSrc builds a small source directory to ingest.
func knowledgeSrc(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"),
		[]byte("# Onboarding\n\nDay one: get VPN access, then clone the monorepo.\n"), 0644))
	return dir
}

func TestIngestKnowledge_WritesToLayerDir(t *testing.T) {
	src := knowledgeSrc(t)
	wikiRoot := t.TempDir()
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "fake", Knowledge: []string{"global"}}
	fake := llm.NewFakeLLM("## Summary\n\nHow to onboard.\n\n## Tags\n\nonboarding, hr\n")

	err := ingestion.IngestKnowledge(context.Background(), src, "global", "onboarding", cfg, fake)
	require.NoError(t, err)

	path := filepath.Join(wikiRoot, "knowledge", "global", "onboarding.md")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	entry, err := wiki.ParseProjectEntry(data)
	require.NoError(t, err)
	assert.Equal(t, "onboarding", entry.Meta.Name)
	assert.Equal(t, "knowledge", entry.Meta.Type)
	assert.Equal(t, src, entry.Meta.Path)
	assert.Equal(t, []string{"onboarding", "hr"}, entry.Meta.Tags)
	assert.Contains(t, entry.Body, "How to onboard.")
	assert.NotContains(t, entry.Body, "## Tags", "tags are lifted into front matter")
}

func TestIngestKnowledge_DefaultsTopicToSourceDirName(t *testing.T) {
	parent := t.TempDir()
	src := filepath.Join(parent, "company-handbook")
	require.NoError(t, os.MkdirAll(src, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "README.md"), []byte("# Handbook\n"), 0644))

	wikiRoot := t.TempDir()
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "fake"}
	fake := llm.NewFakeLLM("## Summary\n\nThe handbook.\n")

	require.NoError(t, ingestion.IngestKnowledge(context.Background(), src, "global", "", cfg, fake))

	assert.FileExists(t, filepath.Join(wikiRoot, "knowledge", "global", "company-handbook.md"))
}

func TestIngestKnowledge_DoesNotTouchProjectIndex(t *testing.T) {
	src := knowledgeSrc(t)
	wikiRoot := t.TempDir()
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "fake", Type: "client", Customer: "acme"}
	fake := llm.NewFakeLLM("## Summary\n\nStuff.\n")

	require.NoError(t, ingestion.IngestKnowledge(context.Background(), src, "global", "policy", cfg, fake))

	// The knowledge layer is walk-based: no index bookkeeping, and no client
	// index generation even though Type=client and Customer is set.
	assert.NoFileExists(t, filepath.Join(wikiRoot, "_index.md"))
	assert.NoDirExists(t, filepath.Join(wikiRoot, "clients"))
}

func TestIngestKnowledge_IsDiscoverableBySearchKnowledge(t *testing.T) {
	src := knowledgeSrc(t)
	wikiRoot := t.TempDir()
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "fake"}
	fake := llm.NewFakeLLM("## Summary\n\nVPN access on day one.\n")

	require.NoError(t, ingestion.IngestKnowledge(context.Background(), src, "global", "onboarding", cfg, fake))

	got, err := wiki.NewStore(wikiRoot).SearchKnowledge([]string{"global"}, "vpn")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "onboarding", got[0].Topic)
	assert.Equal(t, "global", got[0].Layer)
}

func TestIngestKnowledge_RejectsUnsafeLayerAndTopic(t *testing.T) {
	src := knowledgeSrc(t)
	cfg := config.Merged{WikiRoot: t.TempDir(), LLM: "fake"}
	fake := llm.NewFakeLLM("## Summary\n\nx\n")

	err := ingestion.IngestKnowledge(context.Background(), src, "../../etc", "topic", cfg, fake)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge layer")

	err = ingestion.IngestKnowledge(context.Background(), src, "global", "../../escape", cfg, fake)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "topic")
}

func TestIngestKnowledge_PreservesExistingBodyAsContext(t *testing.T) {
	src := knowledgeSrc(t)
	wikiRoot := t.TempDir()
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "fake"}

	first := llm.NewFakeLLM("## Summary\n\nFirst pass.\n")
	require.NoError(t, ingestion.IngestKnowledge(context.Background(), src, "global", "onboarding", cfg, first))

	second := llm.NewFakeLLM("## Summary\n\nSecond pass.\n")
	require.NoError(t, ingestion.IngestKnowledge(context.Background(), src, "global", "onboarding", cfg, second))

	data, err := os.ReadFile(filepath.Join(wikiRoot, "knowledge", "global", "onboarding.md"))
	require.NoError(t, err)
	entry, err := wiki.ParseProjectEntry(data)
	require.NoError(t, err)
	assert.Contains(t, entry.Body, "Second pass.")
	assert.NotContains(t, entry.Body, "First pass.", "re-ingest replaces the body")
}
