package ingestion_test

import (
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/ingestion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultProjectSections resolves the byte-stable "default" preset for
// project prompts — used as a baseline across tests.
func defaultProjectSections(t *testing.T) []ingestion.Section {
	t.Helper()
	s, err := ingestion.ResolveSections(config.ExtractionConfig{}, ingestion.ScopeProject)
	require.NoError(t, err)
	return s
}

func defaultServiceSections(t *testing.T) []ingestion.Section {
	t.Helper()
	s, err := ingestion.ResolveSections(config.ExtractionConfig{}, ingestion.ScopeService)
	require.NoError(t, err)
	return s
}

func TestBuildProjectPrompt_IncludesScan(t *testing.T) {
	prompt := ingestion.BuildProjectPrompt("myproject", "README.md content here", "", "", defaultProjectSections(t), 0)
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
	prompt := ingestion.BuildProjectPrompt("myproject", "scan data", existing, "", defaultProjectSections(t), 0)
	assert.Contains(t, prompt, "Old description.")
	assert.Contains(t, prompt, "update")
}

func TestBuildServicePrompt_IncludesScan(t *testing.T) {
	prompt := ingestion.BuildServicePrompt("api-gateway", "myproject", "proto file content", "", "", defaultServiceSections(t), 0)
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

func TestBuildProjectPrompt_MinimalPresetOmitsSections(t *testing.T) {
	sections, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "minimal"}, ingestion.ScopeProject)
	require.NoError(t, err)
	prompt := ingestion.BuildProjectPrompt("p", "scan", "", "", sections, 0)

	assert.Contains(t, prompt, "## Domain")
	assert.Contains(t, prompt, "## Architecture")
	assert.Contains(t, prompt, "## Features")
	assert.Contains(t, prompt, "## Tags")

	// Minimal drops the heavier sections.
	assert.NotContains(t, prompt, "## Services")
	assert.NotContains(t, prompt, "## Flows")
	assert.NotContains(t, prompt, "## System Diagram")
}

func TestBuildProjectPrompt_SoftwarePresetIncludesSoftwareSections(t *testing.T) {
	sections, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "software"}, ingestion.ScopeProject)
	require.NoError(t, err)
	prompt := ingestion.BuildProjectPrompt("p", "scan", "", "", sections, 0)

	assert.Contains(t, prompt, "## Patterns")
	assert.Contains(t, prompt, "## Good Practices")
	assert.Contains(t, prompt, "## Testing Principles")
	assert.Contains(t, prompt, "## Run & Setup")
	assert.Contains(t, prompt, "## Technical Debt")

	// Feature-only sections should not leak in.
	assert.NotContains(t, prompt, "## Features")
	assert.NotContains(t, prompt, "## Roadmap")
}

func TestBuildProjectPrompt_FeaturePresetIncludesRoadmap(t *testing.T) {
	sections, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "feature"}, ingestion.ScopeProject)
	require.NoError(t, err)
	prompt := ingestion.BuildProjectPrompt("p", "scan", "", "", sections, 0)

	assert.Contains(t, prompt, "## Features")
	assert.Contains(t, prompt, "## Roadmap")
	assert.NotContains(t, prompt, "## Architecture")
	assert.NotContains(t, prompt, "## Patterns")
}

func TestBuildProjectPrompt_MaxTokensAddsWordBudgetHint(t *testing.T) {
	prompt := ingestion.BuildProjectPrompt("p", "scan", "", "", defaultProjectSections(t), 4000)
	// 4000 * 3 / 4 = 3000 words
	assert.Contains(t, prompt, "under roughly 3000 words")

	noBudget := ingestion.BuildProjectPrompt("p", "scan", "", "", defaultProjectSections(t), 0)
	assert.NotContains(t, noBudget, "under roughly")
}

func TestBuildProjectPrompt_ExplicitSectionsRespectsOrder(t *testing.T) {
	sections, err := ingestion.ResolveSections(config.ExtractionConfig{
		Sections: []string{"tags", "domain", "architecture"},
	}, ingestion.ScopeProject)
	require.NoError(t, err)
	prompt := ingestion.BuildProjectPrompt("p", "scan", "", "", sections, 0)

	tagsIdx := stringIndex(prompt, "## Tags")
	domainIdx := stringIndex(prompt, "## Domain")
	archIdx := stringIndex(prompt, "## Architecture")
	assert.True(t, tagsIdx >= 0 && domainIdx >= 0 && archIdx >= 0, "all sections should be present")
	assert.True(t, tagsIdx < domainIdx && domainIdx < archIdx, "order must follow explicit config")
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
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
