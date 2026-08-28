package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contextWiki builds a wiki root with one project entry plus knowledge layers.
func contextWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, wiki.WriteProjectEntry(
		filepath.Join(root, "clients", "acme", "billing.md"),
		wiki.ProjectMeta{Name: "billing", Customer: "acme", Type: "client"},
		"\n## Domain\n\nInvoicing for acme.\n\n## Architecture\n\nGo monolith.\n",
	))
	return root
}

func writeLayerFile(t *testing.T, root, layer, name, content string) {
	t.Helper()
	dir := filepath.Join(root, wiki.KnowledgeDir, layer)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
}

func TestBuildContextOutput_AppendsKnowledgeWithLayerAttribution(t *testing.T) {
	root := contextWiki(t)
	writeLayerFile(t, root, "global", "auth.md", "# Auth Standard\n\n## Summary\n\nAll services use OIDC.\n")
	writeLayerFile(t, root, "platform-team", "deploy.md", "# Deploy\n\n## Summary\n\nArgoCD per service.\n")

	out, err := buildContextOutput(root, "billing", "", nil, []string{"platform-team", "global"})
	require.NoError(t, err)

	assert.Contains(t, out, "Invoicing for acme.", "project content still comes first")
	assert.Contains(t, out, "## Knowledge: platform-team/deploy")
	assert.Contains(t, out, "ArgoCD per service.")
	assert.Contains(t, out, "## Knowledge: global/auth")
	assert.Contains(t, out, "All services use OIDC.")

	// Layer priority order is preserved in the output.
	assert.Less(t, strings.Index(out, "## Knowledge: platform-team/deploy"),
		strings.Index(out, "## Knowledge: global/auth"))
	assert.Less(t, strings.Index(out, "Invoicing for acme."),
		strings.Index(out, "## Knowledge: platform-team/deploy"))
}

func TestBuildContextOutput_NoLayersIsUnchangedOutput(t *testing.T) {
	root := contextWiki(t)
	writeLayerFile(t, root, "global", "auth.md", "# Auth\n\n## Summary\n\nOIDC.\n")

	without, err := buildContextOutput(root, "billing", "", nil, nil)
	require.NoError(t, err)
	assert.NotContains(t, without, "Knowledge:")
	assert.NotContains(t, without, "OIDC")

	with, err := buildContextOutput(root, "billing", "", nil, []string{"global"})
	require.NoError(t, err)
	assert.Contains(t, with, "OIDC")
	assert.True(t, strings.HasPrefix(with, without),
		"knowledge is appended; the project part must be byte-identical")
}

func TestBuildContextOutput_EmptyKnowledgeTreeChangesNothing(t *testing.T) {
	root := contextWiki(t)

	withLayers, err := buildContextOutput(root, "billing", "", nil, []string{"global", "platform-team"})
	require.NoError(t, err)
	withoutLayers, err := buildContextOutput(root, "billing", "", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, withoutLayers, withLayers,
		"configured-but-unpopulated layers must not alter output for existing installs")
}

func TestRenderKnowledgeContext_PrefersKnowledgeSections(t *testing.T) {
	root := t.TempDir()
	writeLayerFile(t, root, "global", "research.md",
		"# Research\n\n## Summary\n\nThe short version.\n\n## Key Topics\n\nA, B, C\n\n"+
			"## Glossary\n\nlots of verbose glossary text that context injection does not need\n")

	got := renderKnowledgeContext(wiki.NewStore(root), []string{"global"})

	assert.Contains(t, got, "The short version.")
	assert.Contains(t, got, "A, B, C")
	assert.NotContains(t, got, "verbose glossary text",
		"only the summary-ish sections are injected when they exist")
}

func TestRenderKnowledgeContext_FallsBackToWholeBody(t *testing.T) {
	root := t.TempDir()
	// Hand-authored file with arbitrary headings — the primary authoring path.
	writeLayerFile(t, root, "global", "adr-001.md",
		"# Use Postgres\n\n## Decision\n\nPostgres over Mongo.\n\n## Consequences\n\nWe run PgBouncer.\n")

	got := renderKnowledgeContext(wiki.NewStore(root), []string{"global"})

	assert.Contains(t, got, "## Knowledge: global/adr-001")
	assert.Contains(t, got, "Postgres over Mongo.", "no ## Summary — inject the whole body")
	assert.Contains(t, got, "We run PgBouncer.")
}

func TestRenderKnowledgeContext_TruncatesPerEntry(t *testing.T) {
	root := t.TempDir()
	writeLayerFile(t, root, "global", "huge.md", "# Huge\n\n"+strings.Repeat("word ", 5000))

	got := renderKnowledgeContext(wiki.NewStore(root), []string{"global"})

	assert.Contains(t, got, "[truncated]")
	assert.Less(t, len(got), knowledgeEntryMaxChars+500, "per-entry cap must bound the output")
}

func TestRenderKnowledgeContext_EnforcesTotalBudget(t *testing.T) {
	root := t.TempDir()
	body := "# T\n\n" + strings.Repeat("word ", 1000)
	for _, name := range []string{"a.md", "b.md", "c.md", "d.md", "e.md", "f.md", "g.md", "h.md"} {
		writeLayerFile(t, root, "global", name, body)
	}

	got := renderKnowledgeContext(wiki.NewStore(root), []string{"global"})

	assert.LessOrEqual(t, len(got), knowledgeTotalMaxChars+knowledgeEntryMaxChars,
		"total budget must bound CLAUDE.md injection size")
	assert.Contains(t, got, "## Knowledge: global/a", "the highest-priority entries survive the budget")
}

func TestRenderKnowledgeContext_SkipsEmptyEntries(t *testing.T) {
	root := t.TempDir()
	writeLayerFile(t, root, "global", "empty.md", "")
	writeLayerFile(t, root, "global", "real.md", "# Real\n\nContent here.\n")

	got := renderKnowledgeContext(wiki.NewStore(root), []string{"global"})

	assert.NotContains(t, got, "empty")
	assert.Contains(t, got, "Content here.")
}

func TestRenderKnowledgeContext_InvalidLayerIsIgnored(t *testing.T) {
	root := t.TempDir()
	writeLayerFile(t, root, "global", "auth.md", "# Auth\n\nOIDC.\n")

	// A bad layer name must not abort context generation for the good ones.
	got := renderKnowledgeContext(wiki.NewStore(root), []string{"../etc", "global"})
	assert.Contains(t, got, "OIDC.")
}

func TestLoadAllWikiContent_IncludesKnowledgeLayers(t *testing.T) {
	// `query` needs no knowledge-specific code: loadAllWikiContent already
	// walks the whole wiki root and prefixes each file with its relative
	// path, which is what gives the LLM layer attribution for free.
	root := contextWiki(t)
	writeLayerFile(t, root, "global", "auth.md", "# Auth\n\nOIDC via Keycloak.\n")

	got, err := loadAllWikiContent(root)
	require.NoError(t, err)

	assert.Contains(t, got, filepath.Join("knowledge", "global", "auth.md"),
		"knowledge files are included, headed by their relative path")
	assert.Contains(t, got, "OIDC via Keycloak.")
	assert.Contains(t, got, "Invoicing for acme.", "project content still included")
}
