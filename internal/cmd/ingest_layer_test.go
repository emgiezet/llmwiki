package cmd

import (
	"testing"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestKnowledgeExtraction_DefaultsToNotesPreset(t *testing.T) {
	// A folder of company docs shouldn't be asked for "## Services" or
	// "## System Diagram" — with nothing configured, default to `notes`.
	got := knowledgeExtraction(config.Merged{})
	assert.Equal(t, "notes", got.Preset)
}

func TestKnowledgeExtraction_RespectsExplicitConfig(t *testing.T) {
	cases := []struct {
		name       string
		cfg        config.Merged
		wantPreset string
		wantSects  []string
	}{
		{
			name:       "explicit preset wins",
			cfg:        config.Merged{Extraction: config.ExtractionConfig{Preset: "research"}},
			wantPreset: "research",
		},
		{
			name:      "explicit sections win and leave preset alone",
			cfg:       config.Merged{Extraction: config.ExtractionConfig{Sections: []string{"summary", "tags"}}},
			wantSects: []string{"summary", "tags"},
		},
		{
			name:       "status-driven resolution is left to ResolveSections",
			cfg:        config.Merged{Status: config.StatusDiscovery},
			wantPreset: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := knowledgeExtraction(tc.cfg)
			assert.Equal(t, tc.wantPreset, got.Preset)
			assert.Equal(t, tc.wantSects, got.Sections)
		})
	}
}

func TestKnowledgeExtraction_PreservesMaxTokens(t *testing.T) {
	got := knowledgeExtraction(config.Merged{Extraction: config.ExtractionConfig{MaxTokens: 2000}})
	assert.Equal(t, 2000, got.MaxTokens)
	assert.Equal(t, "notes", got.Preset, "max_tokens alone doesn't count as configured sections")
}
