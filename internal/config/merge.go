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

	memMode := g.MemoryMode
	if memMode == "" {
		memMode = MemoryModeProject
	}

	m := Merged{
		// Global-only fields.
		WikiRoot:           g.WikiRoot,
		OllamaHost:         g.OllamaHost,
		AllowRemoteOllama:  g.AllowRemoteOllama,
		AnthropicAPIKey:    g.AnthropicAPIKey,
		MemoryEnabled:      g.MemoryEnabled,
		MemoryMode:         memMode,
		MemoryDir:          memDir,
		ProjectMemoryDir:   p.MemoryDir,
		ClaudeBinaryPath:   g.ClaudeBinaryPath,
		GeminiBinaryPath:   g.GeminiBinaryPath,
		CodexBinaryPath:    g.CodexBinaryPath,
		OpencodeBinaryPath: g.OpencodeBinaryPath,
		PiBinaryPath:       g.PiBinaryPath,

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

	// Status: scalar override, project > client. Global has no status.
	if c.Status != "" {
		m.Status = c.Status
	}
	if p.Status != "" {
		m.Status = p.Status
	}

	// Links: key-by-key override. Client keys pass through; project keys
	// override on collision. We track which keys came from the client so
	// the wiki renderer can annotate them.
	m.Links, m.Source.LinksFromClient = mergeLinks(c.Links, p.Links)

	// Team: field-by-field override, track client provenance.
	m.Team, m.Source.TeamLeadFromClient, m.Source.TeamOncallFromClient, m.Source.TeamEscFromClient, m.Source.TeamNotesFromClient = mergeTeam(c.Team, p.Team)

	// Cost: field-by-field override, flag if any field came from client.
	m.Cost, m.Source.CostFromClient = mergeCost(c.Cost, p.Cost)

	return m
}

// mergeLinks key-by-key-overrides project's entries onto client's, and
// returns a companion map flagging which keys in the result came from the
// client (for the inherited-from-client annotation in the wiki).
func mergeLinks(c, p LinksConfig) (LinksConfig, map[string]bool) {
	if len(c) == 0 && len(p) == 0 {
		return nil, nil
	}
	out := make(LinksConfig, len(c)+len(p))
	inherited := make(map[string]bool, len(c))
	for k, v := range c {
		out[k] = v
		inherited[k] = true
	}
	for k, v := range p {
		out[k] = v
		delete(inherited, k) // project override — no longer inherited
	}
	if len(inherited) == 0 {
		inherited = nil
	}
	return out, inherited
}

// mergeTeam applies the non-empty-wins rule per field, returning per-field
// flags for which fields came from the client baseline.
func mergeTeam(c, p TeamConfig) (out TeamConfig, leadFromClient, oncallFromClient, escFromClient, notesFromClient bool) {
	out = c
	if p.Lead != "" {
		out.Lead = p.Lead
	} else if c.Lead != "" {
		leadFromClient = true
	}
	if p.OncallChannel != "" {
		out.OncallChannel = p.OncallChannel
	} else if c.OncallChannel != "" {
		oncallFromClient = true
	}
	if p.Escalation != "" {
		out.Escalation = p.Escalation
	} else if c.Escalation != "" {
		escFromClient = true
	}
	if p.Notes != "" {
		out.Notes = p.Notes
	} else if c.Notes != "" {
		notesFromClient = true
	}
	return out, leadFromClient, oncallFromClient, escFromClient, notesFromClient
}

// mergeCost applies the non-zero-wins rule per numeric field and non-empty-
// wins for the notes string. fromClient is true when any field's merged
// value came from client (so the rendered total shows the inherited flag).
func mergeCost(c, p CostConfig) (out CostConfig, fromClient bool) {
	out = p
	if out.InfraMonthlyUSD == 0 && c.InfraMonthlyUSD != 0 {
		out.InfraMonthlyUSD = c.InfraMonthlyUSD
		fromClient = true
	}
	if out.TeamFTE == 0 && c.TeamFTE != 0 {
		out.TeamFTE = c.TeamFTE
		fromClient = true
	}
	if out.TeamFTERateMonthlyUSD == 0 && c.TeamFTERateMonthlyUSD != 0 {
		out.TeamFTERateMonthlyUSD = c.TeamFTERateMonthlyUSD
		fromClient = true
	}
	if out.Notes == "" && c.Notes != "" {
		out.Notes = c.Notes
		fromClient = true
	}
	return out, fromClient
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
