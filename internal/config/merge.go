package config

import "path/filepath"

// Merge resolves global defaults, a per-client baseline, and a per-project
// override into a single Merged result. Precedence per field:
//
//	project > client > global
//
// Scalars use first-non-zero-wins (so an empty field at one layer falls
// through to the next). Structured fields (Extraction) merge field by
// field with the same rule.
//
// v1.3.0 lands the 3-arg signature plus LLM / Extraction inheritance from
// client. Status / links / team / cost block merging arrives in a
// follow-up commit that adds those fields to ProjectConfig + ClientConfig.
func Merge(g GlobalConfig, c ClientConfig, p ProjectConfig) Merged {
	memDir := g.MemoryDir
	if memDir == "" {
		memDir = filepath.Join(homeDir(), ".llmwiki", "memory")
	}

	m := Merged{
		// Global-only fields.
		WikiRoot:          g.WikiRoot,
		OllamaHost:        g.OllamaHost,
		AllowRemoteOllama: g.AllowRemoteOllama,
		AnthropicAPIKey:   g.AnthropicAPIKey,
		MemoryEnabled:     g.MemoryEnabled,
		MemoryDir:         memDir,
		ClaudeBinaryPath:  g.ClaudeBinaryPath,

		// Project-only fields.
		Customer:    p.Customer,
		Type:        p.Type,
		OllamaModel: p.OllamaModel,

		// Start LLM at the global default; client and project override below.
		LLM: g.LLM,

		// Extraction: three-way merge field by field (see mergeExtraction).
		Extraction: mergeExtraction(g, c, p),
	}

	// LLM: project > client > global.
	if c.LLM != "" {
		m.LLM = c.LLM
	}
	if p.LLM != "" {
		m.LLM = p.LLM
	}

	return m
}

// mergeExtraction resolves ExtractionConfig across the three layers,
// field by field, under the same precedence as Merge itself. Empty-string
// / zero-int / nil-slice counts as "unset" for the purposes of override.
//
// Global doesn't carry ExtractionConfig today, but the signature accepts
// GlobalConfig so the function stays future-proof when/if that changes.
func mergeExtraction(_ GlobalConfig, c ClientConfig, p ProjectConfig) ExtractionConfig {
	out := ExtractionConfig{}

	// Preset — scalar override.
	if c.Extraction.Preset != "" {
		out.Preset = c.Extraction.Preset
	}
	if p.Extraction.Preset != "" {
		out.Preset = p.Extraction.Preset
	}

	// Sections — slice override (non-empty wins, no concat).
	if len(c.Extraction.Sections) > 0 {
		out.Sections = c.Extraction.Sections
	}
	if len(p.Extraction.Sections) > 0 {
		out.Sections = p.Extraction.Sections
	}

	// MaxTokens — scalar override (non-zero wins).
	if c.Extraction.MaxTokens > 0 {
		out.MaxTokens = c.Extraction.MaxTokens
	}
	if p.Extraction.MaxTokens > 0 {
		out.MaxTokens = p.Extraction.MaxTokens
	}

	return out
}
