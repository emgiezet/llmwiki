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

	// Project-level index MUST use the v1.1.1 {customer}_{project}_index.md
	// convention, and the legacy _index.md MUST NOT exist alongside it.
	newIndex := filepath.Join(wikiRoot, "clients", "insly", "mmx3", "insly_mmx3_index.md")
	require.FileExists(t, newIndex, "multi-service project index must use {customer}_{project}_index.md naming")
	legacyIndex := filepath.Join(wikiRoot, "clients", "insly", "mmx3", "_index.md")
	_, legacyErr := os.Stat(legacyIndex)
	assert.True(t, os.IsNotExist(legacyErr), "legacy _index.md must not exist alongside the new-named index")
}

// TestIngestProject_MultiService_MigratesLegacyIndex verifies that when a
// pre-1.1.1 wiki already has a `_index.md` in the project directory, the
// next ingest writes the new {customer}_{project}_index.md AND removes the
// legacy file so we don't end up with duplicates.
func TestIngestProject_MultiService_MigratesLegacyIndex(t *testing.T) {
	projectDir := t.TempDir()
	wikiRoot := t.TempDir()

	// Scaffolding to trigger multi-service path.
	compose := "services:\n  api:\n    image: go\n  worker:\n    image: go\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(compose), 0644))
	for _, svc := range []string{"api", "worker"} {
		svcDir := filepath.Join(projectDir, svc)
		require.NoError(t, os.MkdirAll(svcDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(svcDir, "go.mod"), []byte("module m\n"), 0644))
	}

	// Seed a legacy _index.md simulating a 1.0.x wiki that predates the rename.
	projectWikiDir := filepath.Join(wikiRoot, "clients", "acme", "billing")
	require.NoError(t, os.MkdirAll(projectWikiDir, 0755))
	legacyPath := filepath.Join(projectWikiDir, "_index.md")
	require.NoError(t, os.WriteFile(legacyPath, []byte("---\nname: billing\n---\n# legacy body\n"), 0644))

	fakeBody := "## Purpose\nx\n\n## Architecture\nx\n\n## API Surface\nx\n\n## Data Model\nx\n\n## Integrations\nx\n\n## Configuration\nx\n\n## Notes\nx\n\n## Tags\ngo"
	fakeLLM := llm.NewFakeLLM(fakeBody)
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "claude-code", Customer: "acme", Type: "client"}

	require.NoError(t, ingestion.IngestProject(context.Background(), projectDir, "billing", cfg, fakeLLM, nil))

	// New name exists.
	require.FileExists(t, filepath.Join(projectWikiDir, "acme_billing_index.md"))
	// Legacy gone.
	_, err := os.Stat(legacyPath)
	assert.True(t, os.IsNotExist(err), "legacy _index.md must be removed during migration")
}

// TestIngestProject_Client_IndexUsesCustomerPrefix verifies the
// client-level overview index (produced by GenerateClientIndex inside
// IngestProject for type=client) uses "{customer}_index.md".
func TestIngestProject_Client_IndexUsesCustomerPrefix(t *testing.T) {
	projectDir := t.TempDir()
	wikiRoot := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# app"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module m\n"), 0o644))

	// Client-level overview requires full set of sections so ReadProjectSummaries
	// can extract them. The same fake is used for both project ingest and the
	// client overview (IngestProject loops both through one LLM instance).
	fake := llm.NewFakeLLM(
		"## Domain\nd\n\n" +
			"## Architecture\na\n\n" +
			"## Services\ns\n\n" +
			"## Features\nf\n\n" +
			"## Flows\nfl\n\n" +
			"## System Diagram\nsd\n\n" +
			"## Data Model Diagram\nd\n\n" +
			"## Integrations\ni\n\n" +
			"## Tech Stack\nts\n\n" +
			"## Configuration\nc\n\n" +
			"## Notes\nn\n\n" +
			"## Tags\ngo",
	)
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "claude-code", Customer: "acme", Type: "client"}

	require.NoError(t, ingestion.IngestProject(context.Background(), projectDir, "myapp", cfg, fake, nil))

	require.FileExists(t,
		filepath.Join(wikiRoot, "clients", "acme", "acme_index.md"),
		"client overview must use {customer}_index.md naming")
	_, err := os.Stat(filepath.Join(wikiRoot, "clients", "acme", "_index.md"))
	assert.True(t, os.IsNotExist(err), "legacy _index.md must not exist alongside new client index")
}

// TestIngestProject_Personal_NoCustomerPrefix verifies that the personal
// project type (no customer) produces "{project}_index.md" without a leading
// customer prefix and without a dangling leading underscore.
func TestIngestProject_Personal_NoCustomerPrefix(t *testing.T) {
	projectDir := t.TempDir()
	wikiRoot := t.TempDir()

	compose := "services:\n  web:\n    image: go\n  api:\n    image: go\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(compose), 0644))
	for _, svc := range []string{"web", "api"} {
		svcDir := filepath.Join(projectDir, svc)
		require.NoError(t, os.MkdirAll(svcDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(svcDir, "go.mod"), []byte("module m\n"), 0644))
	}

	fakeBody := "## Purpose\nx\n\n## Architecture\nx\n\n## API Surface\nx\n\n## Data Model\nx\n\n## Integrations\nx\n\n## Configuration\nx\n\n## Notes\nx\n\n## Tags\ngo"
	fakeLLM := llm.NewFakeLLM(fakeBody)
	// No Customer for personal projects.
	cfg := config.Merged{WikiRoot: wikiRoot, LLM: "claude-code", Type: "personal"}

	require.NoError(t, ingestion.IngestProject(context.Background(), projectDir, "myside", cfg, fakeLLM, nil))

	// Expect "myside_index.md" — no "_myside_index.md" with dangling underscore.
	indexPath := filepath.Join(wikiRoot, "personal", "", "myside", "myside_index.md")
	require.FileExists(t, indexPath, "personal project index must use bare {project}_index.md (no empty-customer prefix)")
}
