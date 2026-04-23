package ingestion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/llm"
	"github.com/emgiezet/llmwiki/internal/memory"
	"github.com/emgiezet/llmwiki/internal/scanner"
	"github.com/emgiezet/llmwiki/internal/wiki"
)

// IngestProject scans projectDir and writes wiki entries to cfg.WikiRoot.
// mem may be nil to disable memory recall/storage.
func IngestProject(ctx context.Context, projectDir, projectName string, cfg config.Merged, l llm.LLM, mem *memory.Store) error {
	// Resolve extraction sections up front so a bad preset/section ID fails the
	// run before we touch the scanner or the LLM.
	projectSections, err := ResolveSections(cfg.Extraction, cfg.Status, ScopeProject)
	if err != nil {
		return fmt.Errorf("resolve project sections: %w", err)
	}
	serviceSections, err := ResolveSections(cfg.Extraction, cfg.Status, ScopeService)
	if err != nil {
		return fmt.Errorf("resolve service sections: %w", err)
	}

	services, err := scanner.DetectServices(projectDir)
	if err != nil {
		return err
	}

	if len(services) == 0 {
		if err := ingestSingleService(ctx, projectDir, projectName, cfg, l, mem, projectSections); err != nil {
			return err
		}
	} else {
		if err := ingestMultiService(ctx, projectDir, projectName, services, cfg, l, mem, serviceSections); err != nil {
			return err
		}
		// Generate project-level index for multi-service projects
		if err := GenerateMultiProjectIndex(ctx, cfg.WikiRoot, cfg.Type, cfg.Customer, projectName, l); err != nil {
			return fmt.Errorf("generate project index: %w", err)
		}
	}

	// Cross-link wiki files after writing
	if err := wiki.LinkWikiFiles(cfg.WikiRoot); err != nil {
		return err
	}

	// Generate client index (only for client-type projects)
	if cfg.Type == "client" && cfg.Customer != "" {
		if err := GenerateClientIndex(ctx, cfg.WikiRoot, cfg.Customer, l); err != nil {
			return fmt.Errorf("generate client index: %w", err)
		}
	}

	return nil
}

func ingestSingleService(ctx context.Context, projectDir, projectName string, cfg config.Merged, l llm.LLM, mem *memory.Store, sections []Section) error {
	scan, err := scanner.ScanProject(projectDir)
	if err != nil {
		return err
	}

	wikiPath := wikiFilePath(cfg.WikiRoot, cfg.Type, cfg.Customer, projectName, "")
	existing := loadExistingBody(wikiPath)

	// Recall previous knowledge for prompt enrichment.
	recalled, _ := mem.RecallForProject(ctx, projectName, cfg.Customer)

	prompt := BuildProjectPrompt(projectName, scan.Summary, existing, recalled, sections, cfg.Extraction.MaxTokens)
	body, err := l.Generate(ctx, prompt)
	if err != nil {
		return err
	}
	body = scrubLLMResponse(body)
	tags, body := ParseTagsFromBody(body)

	meta := wiki.ProjectMeta{
		Name:         projectName,
		Customer:     cfg.Customer,
		Type:         cfg.Type,
		Status:       statusString(cfg.Status),
		Path:         projectDir,
		LLM:          cfg.LLM,
		OllamaModel:  cfg.OllamaModel,
		Tags:         tags,
		Links:        map[string]string(cfg.Links),
		LastIngested: time.Now().UTC(),
	}
	body += renderMetadata(cfg)
	if err := wiki.WriteProjectEntry(wikiPath, meta, "\n"+body+"\n"); err != nil {
		return err
	}

	// Store facts from this ingestion.
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

func ingestMultiService(ctx context.Context, projectDir, projectName string, services []scanner.ServiceDir, cfg config.Merged, l llm.LLM, mem *memory.Store, sections []Section) error {
	// Recall project-level knowledge once for all services.
	recalled, _ := mem.RecallForProject(ctx, projectName, cfg.Customer)

	for _, svc := range services {
		scan, err := scanner.ScanProject(svc.Path)
		if err != nil {
			return err
		}

		wikiPath := wikiFilePath(cfg.WikiRoot, cfg.Type, cfg.Customer, projectName, svc.Name)
		existing := loadExistingServiceBody(wikiPath)

		prompt := BuildServicePrompt(svc.Name, projectName, scan.Summary, existing, recalled, sections, cfg.Extraction.MaxTokens)
		body, err := l.Generate(ctx, prompt)
		if err != nil {
			return err
		}
		body = scrubLLMResponse(body)
		tags, body := ParseTagsFromBody(body)

		meta := wiki.ServiceMeta{
			Service:      svc.Name,
			Project:      projectName,
			Customer:     cfg.Customer,
			Path:         svc.Path,
			Tags:         tags,
			Links:        map[string]string(cfg.Links),
			LastIngested: time.Now().UTC(),
		}
		body += renderMetadata(cfg)
		if err := wiki.WriteServiceEntry(wikiPath, meta, "\n"+body+"\n"); err != nil {
			return err
		}

		// Store facts from this service ingestion.
		_ = mem.RememberServiceIngestion(ctx, projectName, svc.Name, cfg.Customer, body, tags)
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

// statusString resolves the ProjectStatus into the string stored in wiki
// front matter. Empty status defaults to "production" — this preserves
// the convention that "no status set" means "normal production project",
// which matches pre-v1.3.0 behaviour where Status was always "active".
func statusString(s config.ProjectStatus) string {
	if s == "" {
		return string(config.StatusProduction)
	}
	return string(s)
}

// renderMetadata translates the merged config's v1.3.0 metadata into the
// wiki package's plain render types and returns the markdown for the
// Links / Team / Cost sections. Empty blocks return "", so unused metadata
// leaves no empty headers in the output.
func renderMetadata(cfg config.Merged) string {
	links := make([]wiki.LinkEntry, 0, len(cfg.Links))
	for k, v := range cfg.Links {
		links = append(links, wiki.LinkEntry{
			Key:        k,
			URL:        v,
			FromClient: cfg.Source.LinksFromClient[k],
		})
	}
	team := wiki.TeamData{
		Lead:             cfg.Team.Lead,
		LeadFromClient:   cfg.Source.TeamLeadFromClient,
		OncallChannel:    cfg.Team.OncallChannel,
		OncallFromClient: cfg.Source.TeamOncallFromClient,
		Escalation:       cfg.Team.Escalation,
		EscFromClient:    cfg.Source.TeamEscFromClient,
		Notes:            cfg.Team.Notes,
		NotesFromClient:  cfg.Source.TeamNotesFromClient,
	}
	cost := wiki.CostData{
		InfraMonthlyUSD:       cfg.Cost.InfraMonthlyUSD,
		TeamFTE:               cfg.Cost.TeamFTE,
		TeamFTERateMonthlyUSD: cfg.Cost.TeamFTERateMonthlyUSD,
		Notes:                 cfg.Cost.Notes,
		FromClient:            cfg.Source.CostFromClient,
	}
	return wiki.RenderMetadataSections(links, team, cost)
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
