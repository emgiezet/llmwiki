package ingestion_test

import (
	"testing"

	"github.com/mgz/llmwiki/internal/ingestion"
	"github.com/stretchr/testify/assert"
)

func TestBuildProjectPrompt_IncludesScan(t *testing.T) {
	prompt := ingestion.BuildProjectPrompt("myproject", "README.md content here", "")
	assert.Contains(t, prompt, "README.md content here")
	assert.Contains(t, prompt, "## Domain")
	assert.Contains(t, prompt, "## Architecture")
	assert.Contains(t, prompt, "## Services")
	assert.Contains(t, prompt, "## Features")
	assert.Contains(t, prompt, "## Flows")
	assert.Contains(t, prompt, "## Integrations")
	assert.Contains(t, prompt, "## Tech Stack")
	assert.Contains(t, prompt, "## Configuration")
	assert.Contains(t, prompt, "## Notes")
	assert.Contains(t, prompt, "## Tags")
	assert.Contains(t, prompt, "thorough")
	assert.NotContains(t, prompt, "Be concise")
}

func TestBuildProjectPrompt_IncludesExisting(t *testing.T) {
	existing := "## Domain\nOld description."
	prompt := ingestion.BuildProjectPrompt("myproject", "scan data", existing)
	assert.Contains(t, prompt, "Old description.")
	assert.Contains(t, prompt, "update")
}

func TestBuildServicePrompt_IncludesScan(t *testing.T) {
	prompt := ingestion.BuildServicePrompt("api-gateway", "myproject", "proto file content", "")
	assert.Contains(t, prompt, "proto file content")
	assert.Contains(t, prompt, "## Purpose")
	assert.Contains(t, prompt, "## Architecture")
	assert.Contains(t, prompt, "## API Surface")
	assert.Contains(t, prompt, "## Data Model")
	assert.Contains(t, prompt, "## Integrations")
	assert.Contains(t, prompt, "## Configuration")
	assert.Contains(t, prompt, "## Notes")
	assert.Contains(t, prompt, "## Tags")
	assert.Contains(t, prompt, "api-gateway")
	assert.Contains(t, prompt, "thorough")
	assert.NotContains(t, prompt, "Be concise")
}
