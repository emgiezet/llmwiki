package ingestion

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/extractor"
	"github.com/emgiezet/llmwiki/internal/llm"
	"github.com/emgiezet/llmwiki/internal/scanner"
	"github.com/emgiezet/llmwiki/internal/validation"
	"github.com/emgiezet/llmwiki/internal/wiki"
)

// IngestKnowledge scans srcDir and writes the result into a knowledge layer at
// <WikiRoot>/knowledge/<layer>/<topic>.md. An empty topic defaults to the base
// name of srcDir.
//
// This reuses the project pipeline (scan → prompt → generate → write) but skips
// everything that belongs to the project axis: no service detection, no
// _index.md upsert, no client/multi-project index generation. The knowledge tree
// is read by walking the filesystem (see wiki.Store.SearchKnowledge), so there
// is no index to maintain.
func IngestKnowledge(ctx context.Context, srcDir, layer, topic string, cfg config.Merged, l llm.LLM) error {
	if err := validation.NameComponent("knowledge layer", layer); err != nil {
		return err
	}
	if topic == "" {
		topic = filepath.Base(srcDir)
	}
	if err := validation.NameComponent("topic", topic); err != nil {
		return err
	}

	sections, err := ResolveSections(cfg.Extraction, cfg.Status, ScopeProject)
	if err != nil {
		return fmt.Errorf("resolve sections: %w", err)
	}

	scan, err := scanner.ScanProject(ctx, srcDir, scanner.WithExtractor(extractor.New(cfg.Extractors)))
	if err != nil {
		return err
	}

	wikiPath := KnowledgeFilePath(cfg.WikiRoot, layer, topic)
	existing := loadExistingBody(wikiPath)

	// Knowledge entries are not project-scoped, so there is no memory recall
	// to enrich the prompt with.
	prompt := BuildProjectPrompt(topic, scan.Summary, existing, "", sections, cfg.Extraction.MaxTokens)
	body, err := l.Generate(ctx, prompt)
	if err != nil {
		return err
	}
	body = scrubLLMResponse(body)
	tags, body := ParseTagsFromBody(body)

	meta := wiki.ProjectMeta{
		Name:            topic,
		Type:            "knowledge",
		Path:            srcDir,
		LLM:             cfg.LLM,
		OllamaModel:     cfg.OllamaModel,
		Tags:            tags,
		LastIngested:    time.Now().UTC(),
		LLMWikiTracking: buildTracking(srcDir, nil),
	}
	return wiki.WriteProjectEntry(wikiPath, meta, "\n"+body+"\n")
}

// KnowledgeFilePath returns the on-disk path of a knowledge entry. Exported so
// the CLI can report where it wrote.
func KnowledgeFilePath(wikiRoot, layer, topic string) string {
	return filepath.Join(wikiRoot, wiki.KnowledgeDir, layer, topic+".md")
}
