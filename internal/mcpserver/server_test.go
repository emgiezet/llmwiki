package mcpserver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/mcpserver"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestWiki(t *testing.T) *wiki.Store {
	t.Helper()
	root := t.TempDir()

	require.NoError(t, wiki.WriteProjectEntry(
		filepath.Join(root, "clients", "acme", "foo.md"),
		wiki.ProjectMeta{Name: "foo", Customer: "acme", Type: "client", Status: "active", Tags: []string{"go"}},
		"## Domain\nFoo handles billing.\n",
	))
	require.NoError(t, wiki.WriteMultiProjectEntry(
		filepath.Join(root, "clients", "acme", "platform", "acme_platform_index.md"),
		wiki.MultiProjectMeta{Name: "platform", Customer: "acme", Type: "client", Status: "active"},
		"## Domain\nPlatform is the core product.\n",
	))
	require.NoError(t, wiki.WriteServiceEntry(
		filepath.Join(root, "clients", "acme", "platform", "api.md"),
		wiki.ServiceMeta{Service: "api", Project: "platform", Customer: "acme"},
		"## Responsibilities\nServes the REST API.\n",
	))
	require.NoError(t, wiki.WriteIndex(filepath.Join(root, "_index.md"), []wiki.IndexEntry{
		{Name: "foo", Customer: "acme", Type: "client", Status: "active", WikiPath: "clients/acme/foo.md"},
		{Name: "platform", Customer: "acme", Type: "client", Status: "active", WikiPath: "clients/acme/platform/acme_platform_index.md"},
	}))

	return wiki.NewStore(root)
}

func TestHandlers_Search_ReturnsMatches(t *testing.T) {
	h := mcpserver.NewHandlers(buildTestWiki(t))

	_, out, err := h.Search(context.Background(), nil, mcpserver.SearchInput{Client: "acme"})

	require.NoError(t, err)
	assert.Equal(t, 2, out.Count)
	require.Len(t, out.Projects, 2)
	var foo *mcpserver.ProjectInfo
	for i := range out.Projects {
		if out.Projects[i].Name == "foo" {
			foo = &out.Projects[i]
		}
	}
	require.NotNil(t, foo)
	assert.Equal(t, "acme", foo.Customer)
	assert.Contains(t, foo.Summary, "Foo handles billing.")
}

func TestHandlers_Search_NoFilters_ReturnsAll(t *testing.T) {
	h := mcpserver.NewHandlers(buildTestWiki(t))

	_, out, err := h.Search(context.Background(), nil, mcpserver.SearchInput{})

	require.NoError(t, err)
	assert.Equal(t, 2, out.Count)
}

func TestHandlers_Get_ReturnsContent(t *testing.T) {
	h := mcpserver.NewHandlers(buildTestWiki(t))

	_, out, err := h.Get(context.Background(), nil, mcpserver.GetInput{Project: "foo"})

	require.NoError(t, err)
	assert.Equal(t, "foo", out.Name)
	assert.Contains(t, out.Content, "Foo handles billing.")
}

func TestHandlers_Get_WithService_ReturnsServiceContent(t *testing.T) {
	h := mcpserver.NewHandlers(buildTestWiki(t))

	_, out, err := h.Get(context.Background(), nil, mcpserver.GetInput{Client: "acme", Project: "platform", Service: "api"})

	require.NoError(t, err)
	assert.Equal(t, "api", out.Service)
	assert.Contains(t, out.Content, "Serves the REST API.")
}

func TestHandlers_Get_NotFound_ReturnsError(t *testing.T) {
	h := mcpserver.NewHandlers(buildTestWiki(t))

	_, _, err := h.Get(context.Background(), nil, mcpserver.GetInput{Project: "nope"})

	require.Error(t, err)
}

func TestNew_RegistersServer(t *testing.T) {
	// New should build a server without panicking and with tools registered.
	srv := mcpserver.New(buildTestWiki(t))
	require.NotNil(t, srv)
}
