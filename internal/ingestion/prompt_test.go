package ingestion_test

import (
	"testing"

	"github.com/mgz/llmwiki/internal/ingestion"
	"github.com/stretchr/testify/assert"
)

func TestBuildProjectPrompt_IncludesScan(t *testing.T) {
	prompt := ingestion.BuildProjectPrompt("myproject", "README.md content here", "", "")
	assert.Contains(t, prompt, "README.md content here")
	assert.Contains(t, prompt, "## Domain")
	assert.Contains(t, prompt, "## Architecture")
	assert.Contains(t, prompt, "## Services")
	assert.Contains(t, prompt, "## Features")
	assert.Contains(t, prompt, "## Flows")
	assert.Contains(t, prompt, "## System Diagram")
	assert.Contains(t, prompt, "## Data Model Diagram")
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
	prompt := ingestion.BuildProjectPrompt("myproject", "scan data", existing, "")
	assert.Contains(t, prompt, "Old description.")
	assert.Contains(t, prompt, "update")
}

func TestBuildServicePrompt_IncludesScan(t *testing.T) {
	prompt := ingestion.BuildServicePrompt("api-gateway", "myproject", "proto file content", "", "")
	assert.Contains(t, prompt, "proto file content")
	assert.Contains(t, prompt, "## Purpose")
	assert.Contains(t, prompt, "## Architecture")
	assert.Contains(t, prompt, "## API Surface")
	assert.Contains(t, prompt, "swagger")
	assert.Contains(t, prompt, "## System Diagram")
	assert.Contains(t, prompt, "## Data Model")
	assert.Contains(t, prompt, "## Data Model Diagram")
	assert.Contains(t, prompt, "## Integrations")
	assert.Contains(t, prompt, "## Configuration")
	assert.Contains(t, prompt, "## Notes")
	assert.Contains(t, prompt, "## Tags")
	assert.Contains(t, prompt, "api-gateway")
	assert.Contains(t, prompt, "thorough")
	assert.NotContains(t, prompt, "Be concise")
}

func TestBuildMaterializePrompt_FromScratch(t *testing.T) {
	facts := "Uses Go. Runs on Kubernetes. RabbitMQ for messaging."
	prompt := ingestion.BuildMaterializePrompt("myproject", facts, "")
	assert.Contains(t, prompt, "myproject")
	assert.Contains(t, prompt, facts)
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
	assert.NotContains(t, prompt, "PROJECT SCAN")
	assert.NotContains(t, prompt, "CURRENT WIKI ENTRY")
}

func TestBuildMaterializePrompt_WithExisting(t *testing.T) {
	facts := "Now also uses Redis."
	existing := "## Domain\nOld description.\n## Architecture\nMonolith."
	prompt := ingestion.BuildMaterializePrompt("myproject", facts, existing)
	assert.Contains(t, prompt, facts)
	assert.Contains(t, prompt, existing)
	assert.Contains(t, prompt, "Update the wiki entry")
	assert.Contains(t, prompt, "CURRENT WIKI ENTRY")
	assert.Contains(t, prompt, "ACCUMULATED FACTS")
	assert.NotContains(t, prompt, "PROJECT SCAN")
}
