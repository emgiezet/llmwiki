package ingestion

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/llm"
	"github.com/mgz/llmwiki/internal/memory"
	"github.com/mgz/llmwiki/internal/wiki"
)

// MaterializeFromMemory rebuilds or updates a project wiki entry using only accumulated
// graymatter facts — no file scanning. Cost: ~5-15K tokens vs 50-100K for full ingest.
func MaterializeFromMemory(ctx context.Context, projectName string, cfg config.Merged, l llm.LLM, mem *memory.Store) error {
	if mem == nil || !mem.Enabled() {
		return fmt.Errorf("memory is not enabled — run 'llmwiki absorb' during sessions first and enable memory_enabled: true in config")
	}

	facts, err := mem.RecallForProject(ctx, projectName, cfg.Customer)
	if err != nil {
		return fmt.Errorf("recall facts: %w", err)
	}

	wikiPath := wikiFilePath(cfg.WikiRoot, cfg.Type, cfg.Customer, projectName, "")
	existingWiki := loadExistingBody(wikiPath)

	if facts == "" && existingWiki == "" {
		return fmt.Errorf("no memory facts found for %q — run 'llmwiki ingest' first or use 'llmwiki absorb' during sessions", projectName)
	}

	prompt := BuildMaterializePrompt(projectName, facts, existingWiki)
	body, err := l.Generate(ctx, prompt)
	if err != nil {
		return fmt.Errorf("generate wiki: %w", err)
	}

	tags, body := ParseTagsFromBody(body)

	meta := wiki.ProjectMeta{
		Name:         projectName,
		Customer:     cfg.Customer,
		Type:         cfg.Type,
		Status:       "active",
		LLM:          cfg.LLM,
		OllamaModel:  cfg.OllamaModel,
		Tags:         tags,
		LastIngested: time.Now().UTC(),
		// Path intentionally omitted — MaterializeFromMemory has no projectDir context.
	}
	if err := wiki.WriteProjectEntry(wikiPath, meta, "\n"+body+"\n"); err != nil {
		return fmt.Errorf("write wiki entry: %w", err)
	}

	_ = mem.RememberIngestion(ctx, projectName, cfg.Customer, body, tags)

	relPath, _ := filepath.Rel(cfg.WikiRoot, wikiPath)
	return wiki.UpsertIndex(filepath.Join(cfg.WikiRoot, "_index.md"), wiki.IndexEntry{
		Name:     projectName,
		Customer: cfg.Customer,
		Type:     cfg.Type,
		Status:   "active",
		WikiPath: relPath,
	})
}
