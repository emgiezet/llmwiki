package ingestion

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/llm"
	"github.com/mgz/llmwiki/internal/scanner"
	"github.com/mgz/llmwiki/internal/wiki"
)

// IngestProject scans projectDir and writes wiki entries to cfg.WikiRoot.
func IngestProject(ctx context.Context, projectDir, projectName string, cfg config.Merged, l llm.LLM) error {
	services, err := scanner.DetectServices(projectDir)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		if err := ingestSingleService(ctx, projectDir, projectName, cfg, l); err != nil {
			return err
		}
	} else {
		if err := ingestMultiService(ctx, projectDir, projectName, services, cfg, l); err != nil {
			return err
		}
	}

	// Cross-link wiki files after writing
	return wiki.LinkWikiFiles(cfg.WikiRoot)
}

func ingestSingleService(ctx context.Context, projectDir, projectName string, cfg config.Merged, l llm.LLM) error {
	scan, err := scanner.ScanProject(projectDir)
	if err != nil {
		return err
	}

	wikiPath := wikiFilePath(cfg.WikiRoot, cfg.Type, cfg.Customer, projectName, "")
	existing := loadExistingBody(wikiPath)

	prompt := BuildProjectPrompt(projectName, scan.Summary, existing)
	body, err := l.Generate(ctx, prompt)
	if err != nil {
		return err
	}
	tags, body := ParseTagsFromBody(body)

	meta := wiki.ProjectMeta{
		Name:         projectName,
		Customer:     cfg.Customer,
		Type:         cfg.Type,
		Status:       "active",
		Path:         projectDir,
		LLM:          cfg.LLM,
		OllamaModel:  cfg.OllamaModel,
		Tags:         tags,
		LastIngested: time.Now().UTC(),
	}
	if err := wiki.WriteProjectEntry(wikiPath, meta, "\n"+body+"\n"); err != nil {
		return err
	}

	relPath, _ := filepath.Rel(cfg.WikiRoot, wikiPath)
	return wiki.UpsertIndex(filepath.Join(cfg.WikiRoot, "_index.md"), wiki.IndexEntry{
		Name:     projectName,
		Customer: cfg.Customer,
		Type:     cfg.Type,
		Status:   "active",
		WikiPath: relPath,
	})
}

func ingestMultiService(ctx context.Context, projectDir, projectName string, services []scanner.ServiceDir, cfg config.Merged, l llm.LLM) error {
	for _, svc := range services {
		scan, err := scanner.ScanProject(svc.Path)
		if err != nil {
			return err
		}

		wikiPath := wikiFilePath(cfg.WikiRoot, cfg.Type, cfg.Customer, projectName, svc.Name)
		existing := loadExistingServiceBody(wikiPath)

		prompt := BuildServicePrompt(svc.Name, projectName, scan.Summary, existing)
		body, err := l.Generate(ctx, prompt)
		if err != nil {
			return err
		}
		tags, body := ParseTagsFromBody(body)

		meta := wiki.ServiceMeta{
			Service:      svc.Name,
			Project:      projectName,
			Customer:     cfg.Customer,
			Path:         svc.Path,
			Tags:         tags,
			LastIngested: time.Now().UTC(),
		}
		if err := wiki.WriteServiceEntry(wikiPath, meta, "\n"+body+"\n"); err != nil {
			return err
		}
	}

	indexPath := filepath.Join(cfg.WikiRoot, TypeToDir(cfg.Type), cfg.Customer, projectName, "_index.md")
	relPath, _ := filepath.Rel(cfg.WikiRoot, indexPath)
	return wiki.UpsertIndex(filepath.Join(cfg.WikiRoot, "_index.md"), wiki.IndexEntry{
		Name:     projectName,
		Customer: cfg.Customer,
		Type:     cfg.Type,
		Status:   "active",
		WikiPath: relPath,
	})
}

// TypeToDir maps project type to wiki directory name.
// personal and opensource don't follow the simple "type+s" pattern.
func TypeToDir(projectType string) string {
	switch projectType {
	case "personal":
		return "personal"
	case "oss":
		return "opensource"
	default:
		return projectType + "s" // "client" → "clients"
	}
}

// wikiFilePath returns the absolute path for a wiki entry.
func wikiFilePath(wikiRoot, projectType, customer, project, service string) string {
	typeDir := TypeToDir(projectType)
	if service == "" {
		return filepath.Join(wikiRoot, typeDir, customer, project+".md")
	}
	return filepath.Join(wikiRoot, typeDir, customer, project, service+".md")
}

func loadExistingBody(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	entry, err := wiki.ParseProjectEntry(data)
	if err != nil {
		return ""
	}
	return entry.Body
}

func loadExistingServiceBody(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	entry, err := wiki.ParseServiceEntry(data)
	if err != nil {
		return ""
	}
	return entry.Body
}
