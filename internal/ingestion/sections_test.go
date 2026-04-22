package ingestion_test

import (
	"strings"
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/ingestion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSections_DefaultPresetProjectScope(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{}, ingestion.ScopeProject)
	require.NoError(t, err)

	ids := sectionIDs(got)
	// The "default" preset must keep today's project prompt shape.
	assert.Contains(t, ids, "domain")
	assert.Contains(t, ids, "architecture")
	assert.Contains(t, ids, "services")
	assert.Contains(t, ids, "features")
	assert.Contains(t, ids, "flows")
	assert.Contains(t, ids, "system_diagram")
	assert.Contains(t, ids, "data_model_diagram")
	assert.Contains(t, ids, "integrations")
	assert.Contains(t, ids, "tech_stack")
	assert.Contains(t, ids, "configuration")
	assert.Contains(t, ids, "notes")
	assert.Contains(t, ids, "tags")

	// Service-only sections must be filtered out of project scope.
	assert.NotContains(t, ids, "purpose")
	assert.NotContains(t, ids, "api_surface")
	assert.NotContains(t, ids, "data_model")
}

func TestResolveSections_DefaultPresetServiceScope(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{}, ingestion.ScopeService)
	require.NoError(t, err)

	ids := sectionIDs(got)
	assert.Contains(t, ids, "purpose")
	assert.Contains(t, ids, "api_surface")
	assert.Contains(t, ids, "data_model")

	// Project-only sections filtered out.
	assert.NotContains(t, ids, "services")
	assert.NotContains(t, ids, "roadmap")
	assert.NotContains(t, ids, "runbook")
}

func TestResolveSections_ExplicitListOverridesPreset(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{
		Preset:   "software",
		Sections: []string{"domain", "tags"},
	}, ingestion.ScopeProject)
	require.NoError(t, err)

	assert.Equal(t, []string{"domain", "tags"}, sectionIDs(got))
}

func TestResolveSections_Minimal(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "minimal"}, ingestion.ScopeProject)
	require.NoError(t, err)

	assert.Equal(t, []string{"domain", "architecture", "features", "tags"}, sectionIDs(got))
}

func TestResolveSections_Software(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "software"}, ingestion.ScopeProject)
	require.NoError(t, err)

	ids := sectionIDs(got)
	assert.Contains(t, ids, "patterns")
	assert.Contains(t, ids, "good_practices")
	assert.Contains(t, ids, "testing")
	assert.Contains(t, ids, "runbook")
	assert.Contains(t, ids, "tech_debt")
	assert.NotContains(t, ids, "features")
	assert.NotContains(t, ids, "roadmap")
}

func TestResolveSections_Feature(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "feature"}, ingestion.ScopeProject)
	require.NoError(t, err)

	ids := sectionIDs(got)
	assert.Contains(t, ids, "features")
	assert.Contains(t, ids, "roadmap")
	assert.NotContains(t, ids, "architecture")
	assert.NotContains(t, ids, "patterns")
}

func TestResolveSections_UnknownPreset(t *testing.T) {
	_, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "typo"}, ingestion.ScopeProject)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown preset")
	assert.Contains(t, err.Error(), "typo")
}

func TestResolveSections_UnknownSectionID(t *testing.T) {
	_, err := ingestion.ResolveSections(config.ExtractionConfig{
		Sections: []string{"domain", "nope"},
	}, ingestion.ScopeProject)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown section")
	assert.Contains(t, err.Error(), "nope")
}

func TestResolveSections_EmptyAfterScopeFilterErrors(t *testing.T) {
	// api_surface is service-only; resolving for project scope should error
	// rather than silently returning an empty slice.
	_, err := ingestion.ResolveSections(config.ExtractionConfig{
		Sections: []string{"api_surface"},
	}, ingestion.ScopeProject)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sections resolved")
}

func TestResolveSections_DeduplicatesExplicitList(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{
		Sections: []string{"domain", "tags", "domain"},
	}, ingestion.ScopeProject)
	require.NoError(t, err)
	assert.Equal(t, []string{"domain", "tags"}, sectionIDs(got))
}

func TestResolveSections_FullPresetIncludesAll(t *testing.T) {
	got, err := ingestion.ResolveSections(config.ExtractionConfig{Preset: "full"}, ingestion.ScopeProject)
	require.NoError(t, err)

	ids := sectionIDs(got)
	// "full" is everything in the registry, filtered by scope. Spot-check some.
	assert.Contains(t, ids, "domain")
	assert.Contains(t, ids, "roadmap")
	assert.Contains(t, ids, "tech_debt")
	assert.Contains(t, ids, "runbook")
}

func TestAllSections_UniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, s := range ingestion.AllSections {
		require.False(t, seen[s.ID], "duplicate section ID %q", s.ID)
		seen[s.ID] = true
		// Sanity: every section must have a non-empty title and instruction.
		require.NotEmpty(t, s.Title, "section %q missing Title", s.ID)
		require.NotEmpty(t, s.Instruction, "section %q missing Instruction", s.ID)
		// Every ID must be lowercase snake-style (no spaces, no uppercase).
		require.Equal(t, strings.ToLower(s.ID), s.ID, "section ID must be lowercase: %q", s.ID)
	}
}

func sectionIDs(sections []ingestion.Section) []string {
	ids := make([]string, len(sections))
	for i, s := range sections {
		ids[i] = s.ID
	}
	return ids
}
