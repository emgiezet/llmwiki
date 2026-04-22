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

func TestIngestProject_SingleService(t *testing.T) {
	projectDir := t.TempDir()
	wikiRoot := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# My App\nDoes insurance things."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/myapp\n"), 0644))

	fakeLLM := llm.NewFakeLLM("## Domain\nInsurance app.\n\n## Architecture\nMonolith.\n\n## Services\n- api: main service\n\n## Features\nQuote management.\n\n## Flows\nuser → api\n\n## Integrations\nPostgres\n\n## Tech Stack\nGo\n\n## Configuration\nPORT=8080\n\n## Notes\nNone.\n\n## Tags\ngo, rest, postgres")

	cfg := config.Merged{
		WikiRoot: wikiRoot,
		LLM:      "claude-code",
		Customer: "acme",
		Type:     "client",
	}

	err := ingestion.IngestProject(context.Background(), projectDir, "myapp", cfg, fakeLLM, nil)
	require.NoError(t, err)

	// Single-service project: wiki/clients/acme/myapp.md
	wikiFile := filepath.Join(wikiRoot, "clients", "acme", "myapp.md")
	data, err := os.ReadFile(wikiFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "## Domain")
	assert.Contains(t, string(data), "Insurance app.")
	assert.NotContains(t, string(data), "## Tags")

	entry, parseErr := wiki.ParseProjectEntry(data)
	require.NoError(t, parseErr)
	assert.Equal(t, []string{"go", "rest", "postgres"}, entry.Meta.Tags)

	// Index updated
	indexData, err := os.ReadFile(filepath.Join(wikiRoot, "_index.md"))
	require.NoError(t, err)
	assert.Contains(t, string(indexData), "myapp")
}

func TestIngestProject_MultiService(t *testing.T) {
	projectDir := t.TempDir()
	wikiRoot := t.TempDir()

	// Create docker-compose with two services
	compose := "services:\n  api-gateway:\n    image: go\n  worker:\n    image: go\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(compose), 0644))

	// Create service subdirectories
	for _, svc := range []string{"api-gateway", "worker"} {
		svcDir := filepath.Join(projectDir, svc)
		require.NoError(t, os.MkdirAll(svcDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(svcDir, "go.mod"), []byte("module example.com/"+svc+"\n"), 0644))
	}

	fakeBody := "## Purpose\nDoes stuff.\n\n## Architecture\nSimple.\n\n## API Surface\nNone.\n\n## Data Model\nNone.\n\n## Integrations\nNone.\n\n## Configuration\nNone.\n\n## Notes\nNone.\n\n## Tags\ngo, grpc"
	fakeLLM := llm.NewFakeLLM(fakeBody)

	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "claude-code", Customer: "insly", Type: "client"}

	err := ingestion.IngestProject(context.Background(), projectDir, "mmx3", cfg, fakeLLM, nil)
	require.NoError(t, err)

	// Multi-service: wiki/clients/insly/mmx3/api-gateway.md
	for _, svc := range []string{"api-gateway", "worker"} {
		wikiFile := filepath.Join(wikiRoot, "clients", "insly", "mmx3", svc+".md")
		data, err := os.ReadFile(wikiFile)
		require.NoError(t, err)
		assert.Contains(t, string(data), "## Purpose")
		assert.NotContains(t, string(data), "## Tags")

		svcEntry, parseErr := wiki.ParseServiceEntry(data)
		require.NoError(t, parseErr)
		assert.Equal(t, []string{"go", "grpc"}, svcEntry.Meta.Tags)
	}
}
