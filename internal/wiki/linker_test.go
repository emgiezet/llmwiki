package wiki_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyLinks_BasicReplacement(t *testing.T) {
	content := "---\nname: doc-svc\n---\n## Integrations\nUses qmt-api for quoting.\n"
	targets := []wiki.LinkTarget{
		{Name: "qmt-api", WikiPath: "clients/insly/qmt-api.md"},
	}
	result := wiki.ApplyLinks(content, "clients/insly/document-service.md", targets, "")
	assert.Contains(t, result, "[qmt-api](qmt-api.md)")
	assert.Contains(t, result, "## Integrations")
}

func TestApplyLinks_NoSelfLink(t *testing.T) {
	content := "---\nname: qmt-api\n---\n## Notes\nThis is qmt-api itself.\n"
	targets := []wiki.LinkTarget{
		{Name: "qmt-api", WikiPath: "clients/insly/qmt-api.md"},
	}
	result := wiki.ApplyLinks(content, "clients/insly/qmt-api.md", targets, "")
	assert.NotContains(t, result, "[qmt-api]")
}

func TestApplyLinks_PreservesExistingLinks(t *testing.T) {
	content := "## Notes\nSee [qmt-api](../qmt-api.md) for details.\n"
	targets := []wiki.LinkTarget{
		{Name: "qmt-api", WikiPath: "clients/insly/qmt-api.md"},
	}
	result := wiki.ApplyLinks(content, "clients/insly/doc-svc.md", targets, "")
	// Should not double-link — only one markdown link pattern
	assert.Equal(t, 1, countOccurrences(result, "[qmt-api]"))
}

func TestApplyLinks_PreservesCodeBlocks(t *testing.T) {
	content := "## Notes\n```\nqmt-api is configured here\n```\n"
	targets := []wiki.LinkTarget{
		{Name: "qmt-api", WikiPath: "clients/insly/qmt-api.md"},
	}
	result := wiki.ApplyLinks(content, "clients/insly/doc-svc.md", targets, "")
	assert.NotContains(t, result, "[qmt-api]")
}

func TestApplyLinks_RelativePaths(t *testing.T) {
	content := "## Integrations\nCalls document-service for docs.\n"
	targets := []wiki.LinkTarget{
		{Name: "document-service", WikiPath: "clients/insly/document-service.md"},
	}
	// Source is in a different directory
	result := wiki.ApplyLinks(content, "personal/llmwiki.md", targets, "")
	assert.Contains(t, result, "[document-service](../clients/insly/document-service.md)")
}

func TestApplyLinks_LongerNameFirst(t *testing.T) {
	content := "## Notes\nUses qmt-api-gateway and qmt-api.\n"
	targets := []wiki.LinkTarget{
		{Name: "qmt-api", WikiPath: "clients/insly/qmt-api.md"},
		{Name: "qmt-api-gateway", WikiPath: "clients/insly/mmx3/qmt-api-gateway.md"},
	}
	result := wiki.ApplyLinks(content, "personal/llmwiki.md", targets, "")
	assert.Contains(t, result, "[qmt-api-gateway]")
	assert.Contains(t, result, "[qmt-api](")
}

func TestApplyLinks_Idempotent(t *testing.T) {
	content := "## Notes\nUses qmt-api for quoting.\n"
	targets := []wiki.LinkTarget{
		{Name: "qmt-api", WikiPath: "clients/insly/qmt-api.md"},
	}
	first := wiki.ApplyLinks(content, "clients/insly/doc-svc.md", targets, "")
	second := wiki.ApplyLinks(first, "clients/insly/doc-svc.md", targets, "")
	assert.Equal(t, first, second)
}

func TestDiscoverLinkTargets(t *testing.T) {
	wikiRoot := t.TempDir()

	// Create index with two projects
	entries := []wiki.IndexEntry{
		{Name: "proj-a", Type: "client", Customer: "acme", Status: "active", WikiPath: "clients/acme/proj-a.md"},
		{Name: "proj-b", Type: "personal", Status: "active", WikiPath: "personal/proj-b.md"},
	}
	require.NoError(t, wiki.WriteIndex(filepath.Join(wikiRoot, "_index.md"), entries))

	// Create the wiki files
	require.NoError(t, os.MkdirAll(filepath.Join(wikiRoot, "clients", "acme"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wikiRoot, "clients", "acme", "proj-a.md"), []byte("# proj-a"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(wikiRoot, "personal"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wikiRoot, "personal", "proj-b.md"), []byte("# proj-b"), 0644))

	targets, err := wiki.DiscoverLinkTargets(wikiRoot)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(targets), 2)

	names := make([]string, len(targets))
	for i, t := range targets {
		names[i] = t.Name
	}
	assert.Contains(t, names, "proj-a")
	assert.Contains(t, names, "proj-b")
}

func TestLinkWikiFiles_Integration(t *testing.T) {
	wikiRoot := t.TempDir()

	// Create index
	entries := []wiki.IndexEntry{
		{Name: "svc-a", WikiPath: "clients/acme/svc-a.md"},
		{Name: "svc-b", WikiPath: "clients/acme/svc-b.md"},
	}
	require.NoError(t, wiki.WriteIndex(filepath.Join(wikiRoot, "_index.md"), entries))

	// Create wiki files that reference each other
	dir := filepath.Join(wikiRoot, "clients", "acme")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc-a.md"),
		[]byte("---\nname: svc-a\n---\n## Integrations\nCalls svc-b downstream.\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc-b.md"),
		[]byte("---\nname: svc-b\n---\n## Integrations\nCalled by svc-a upstream.\n"), 0644))

	require.NoError(t, wiki.LinkWikiFiles(wikiRoot))

	// svc-a.md should link to svc-b
	dataA, _ := os.ReadFile(filepath.Join(dir, "svc-a.md"))
	assert.Contains(t, string(dataA), "[svc-b](svc-b.md)")

	// svc-b.md should link to svc-a
	dataB, _ := os.ReadFile(filepath.Join(dir, "svc-b.md"))
	assert.Contains(t, string(dataB), "[svc-a](svc-a.md)")
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
