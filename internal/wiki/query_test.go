package wiki_test

import (
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTestWiki creates a temporary wiki root with a mix of single-service,
// personal, and multi-service entries plus a matching _index.md.
func buildTestWiki(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	require.NoError(t, wiki.WriteProjectEntry(
		filepath.Join(root, "clients", "acme", "foo.md"),
		wiki.ProjectMeta{Name: "foo", Customer: "acme", Type: "client", Status: "active", Tags: []string{"go", "cli"}},
		"## Domain\nFoo handles billing.\n\n## Flows\nInvoice creation.\n",
	))
	require.NoError(t, wiki.WriteProjectEntry(
		filepath.Join(root, "personal", "bar.md"),
		wiki.ProjectMeta{Name: "bar", Type: "personal", Status: "active", Tags: []string{"rust"}},
		"## Domain\nBar is a personal toy.\n",
	))
	require.NoError(t, wiki.WriteMultiProjectEntry(
		filepath.Join(root, "clients", "acme", "platform", "acme_platform_index.md"),
		wiki.MultiProjectMeta{Name: "platform", Customer: "acme", Type: "client", Status: "active", Services: []string{"api", "web"}, Tags: []string{"go"}},
		"## Domain\nPlatform is the core product.\n",
	))
	require.NoError(t, wiki.WriteServiceEntry(
		filepath.Join(root, "clients", "acme", "platform", "api.md"),
		wiki.ServiceMeta{Service: "api", Project: "platform", Customer: "acme"},
		"## Responsibilities\nServes the REST API.\n",
	))
	require.NoError(t, wiki.WriteServiceEntry(
		filepath.Join(root, "clients", "acme", "platform", "web.md"),
		wiki.ServiceMeta{Service: "web", Project: "platform", Customer: "acme"},
		"## Responsibilities\nServes the web UI.\n",
	))

	require.NoError(t, wiki.WriteIndex(filepath.Join(root, "_index.md"), []wiki.IndexEntry{
		{Name: "foo", Customer: "acme", Type: "client", Status: "active", WikiPath: "clients/acme/foo.md"},
		{Name: "bar", Customer: "", Type: "personal", Status: "active", WikiPath: "personal/bar.md"},
		{Name: "platform", Customer: "acme", Type: "client", Status: "active", WikiPath: "clients/acme/platform/acme_platform_index.md"},
	}))

	return root
}

func names(matches []wiki.ProjectMatch) []string {
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.Name
	}
	return out
}

func TestStore_Search_NoFilters_ReturnsAll(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	matches, err := s.Search("", "")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"foo", "bar", "platform"}, names(matches))
}

func TestStore_Search_ByClient_ReturnsClientProjects(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	matches, err := s.Search("acme", "")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"foo", "platform"}, names(matches))
}

func TestStore_Search_ByClient_IsCaseInsensitive(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	matches, err := s.Search("ACME", "")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"foo", "platform"}, names(matches))
}

func TestStore_Search_ByProject_IsSubstringMatch(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	matches, err := s.Search("", "ba")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"bar"}, names(matches))
}

func TestStore_Search_ByClientAndProject_Intersects(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	matches, err := s.Search("acme", "platform")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"platform"}, names(matches))
}

func TestStore_Search_NoMatch_ReturnsEmpty(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	matches, err := s.Search("nope", "")

	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestStore_Search_PopulatesTagsAndSummary(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	matches, err := s.Search("", "foo")

	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, []string{"go", "cli"}, matches[0].Tags)
	assert.Contains(t, matches[0].Summary, "Foo handles billing.")
	assert.NotContains(t, matches[0].Summary, "Invoice creation.") // only Domain section
}

func TestStore_GetProject_SingleService_ReturnsBody(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	content, meta, err := s.GetProject("", "foo", "")

	require.NoError(t, err)
	assert.Equal(t, "foo", meta.Name)
	assert.Contains(t, content, "Foo handles billing.")
	assert.Contains(t, content, "Invoice creation.")
}

func TestStore_GetProject_MultiService_ReturnsIndexBody(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	content, meta, err := s.GetProject("acme", "platform", "")

	require.NoError(t, err)
	assert.Equal(t, "platform", meta.Name)
	assert.Contains(t, content, "Platform is the core product.")
}

func TestStore_GetProject_WithService_ReturnsServiceBody(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	content, _, err := s.GetProject("acme", "platform", "api")

	require.NoError(t, err)
	assert.Contains(t, content, "Serves the REST API.")
	assert.NotContains(t, content, "Serves the web UI.")
}

func TestStore_GetProject_NotFound_Errors(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	_, _, err := s.GetProject("", "nonexistent", "")

	require.Error(t, err)
}

func TestStore_GetProject_Ambiguous_ErrorsWithCandidates(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	// "a" matches both "bar" and "platform" (substring) across clients.
	_, _, err := s.GetProject("", "a", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bar")
	assert.Contains(t, err.Error(), "platform")
}

func TestStore_GetProject_ServiceOnSingleService_Errors(t *testing.T) {
	s := wiki.NewStore(buildTestWiki(t))

	_, _, err := s.GetProject("", "foo", "api")

	require.Error(t, err)
}

func TestStore_ListServices_ReturnsServiceNames(t *testing.T) {
	root := buildTestWiki(t)
	s := wiki.NewStore(root)

	services, err := s.ListServices("clients/acme/platform/acme_platform_index.md")

	require.NoError(t, err)
	assert.Equal(t, []string{"api", "web"}, services)
}
