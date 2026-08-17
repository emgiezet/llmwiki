package mcpserver_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/emgiezet/llmwiki/internal/mcpserver"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildKnowledgeWiki adds knowledge layers on top of the project fixture.
func buildKnowledgeWiki(t *testing.T) *wiki.Store {
	t.Helper()
	store := buildTestWiki(t)
	write := func(layer, name, content string) {
		dir := filepath.Join(store.Root, wiki.KnowledgeDir, layer)
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
	}
	write("global", "auth.md", "# Auth Standard\n\nAll services use OIDC via Keycloak.\n")
	write("global", "billing-policy.md", "# Billing Policy\n\nNet 30 for all clients.\n")
	write("platform-team", "deploy.md", "# Deploy\n\nArgoCD, one app per service.\n")
	return store
}

func TestHandlers_SearchKnowledge_AllLayersWhenNoneNamed(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, out, err := h.SearchKnowledge(context.Background(), nil, mcpserver.KnowledgeSearchInput{})

	require.NoError(t, err)
	assert.Equal(t, 3, out.Count)
	layers := map[string]int{}
	for _, e := range out.Entries {
		layers[e.Layer]++
	}
	assert.Equal(t, map[string]int{"global": 2, "platform-team": 1}, layers,
		"every result carries its layer, so agents can discover layer names")
}

func TestHandlers_SearchKnowledge_FiltersByLayer(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, out, err := h.SearchKnowledge(context.Background(), nil,
		mcpserver.KnowledgeSearchInput{Layer: "platform-team"})

	require.NoError(t, err)
	require.Equal(t, 1, out.Count)
	assert.Equal(t, "platform-team", out.Entries[0].Layer)
	assert.Equal(t, "deploy", out.Entries[0].Topic)
	assert.Equal(t, "Deploy", out.Entries[0].Title)
	assert.Contains(t, out.Entries[0].Snippet, "ArgoCD")
	assert.Equal(t, filepath.Join("knowledge", "platform-team", "deploy.md"), out.Entries[0].Path)
}

func TestHandlers_SearchKnowledge_FiltersByQuery(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, out, err := h.SearchKnowledge(context.Background(), nil,
		mcpserver.KnowledgeSearchInput{Query: "keycloak"})

	require.NoError(t, err)
	require.Equal(t, 1, out.Count)
	assert.Equal(t, "auth", out.Entries[0].Topic)
}

func TestHandlers_SearchKnowledge_QueryAndLayerCombined(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, out, err := h.SearchKnowledge(context.Background(), nil,
		mcpserver.KnowledgeSearchInput{Query: "keycloak", Layer: "platform-team"})

	require.NoError(t, err)
	assert.Equal(t, 0, out.Count, "query matches nothing in that layer")
	assert.Empty(t, out.Entries)
}

func TestHandlers_SearchKnowledge_EmptyWikiIsNotAnError(t *testing.T) {
	h := mcpserver.NewHandlers(wiki.NewStore(t.TempDir()))

	_, out, err := h.SearchKnowledge(context.Background(), nil, mcpserver.KnowledgeSearchInput{})

	require.NoError(t, err)
	assert.Equal(t, 0, out.Count)
}

func TestHandlers_SearchKnowledge_RejectsUnsafeLayer(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, _, err := h.SearchKnowledge(context.Background(), nil,
		mcpserver.KnowledgeSearchInput{Layer: "../../etc"})

	require.Error(t, err)
}

func TestHandlers_GetKnowledge_ByTopicSearchesAllLayers(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, out, err := h.GetKnowledge(context.Background(), nil,
		mcpserver.KnowledgeGetInput{Topic: "deploy"})

	require.NoError(t, err)
	assert.Equal(t, "platform-team", out.Layer, "layer is resolved and reported back")
	assert.Equal(t, "deploy", out.Topic)
	assert.Equal(t, "Deploy", out.Title)
	assert.Contains(t, out.Content, "ArgoCD, one app per service.")
}

func TestHandlers_GetKnowledge_WithExplicitLayer(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, out, err := h.GetKnowledge(context.Background(), nil,
		mcpserver.KnowledgeGetInput{Topic: "auth", Layer: "global"})

	require.NoError(t, err)
	assert.Equal(t, "global", out.Layer)
	assert.Contains(t, out.Content, "OIDC via Keycloak.")
}

func TestHandlers_GetKnowledge_NotFound(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, _, err := h.GetKnowledge(context.Background(), nil,
		mcpserver.KnowledgeGetInput{Topic: "nonexistent"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestHandlers_GetKnowledge_WrongLayerIsNotFound(t *testing.T) {
	h := mcpserver.NewHandlers(buildKnowledgeWiki(t))

	_, _, err := h.GetKnowledge(context.Background(), nil,
		mcpserver.KnowledgeGetInput{Topic: "deploy", Layer: "global"})

	require.Error(t, err)
}
