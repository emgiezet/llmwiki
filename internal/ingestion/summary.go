package ingestion

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/emgiezet/llmwiki/internal/wiki"
)

// ProjectSummary holds extracted sections from a wiki entry for index prompts.
type ProjectSummary struct {
	Name           string
	Tags           []string
	IsMultiService bool
	Services       []ServiceSummary
	Domain         string
	Architecture   string
	Integrations   string
	TechStack      string
}

// ServiceSummary holds extracted sections from a service wiki entry.
type ServiceSummary struct {
	Name         string
	Tags         []string
	Purpose      string
	Integrations string
}

// ReadProjectSummaries reads all project wiki files for a customer and extracts key sections.
func ReadProjectSummaries(wikiRoot, customer string) ([]ProjectSummary, error) {
	customerDir := filepath.Join(wikiRoot, "clients", customer)
	entries, err := os.ReadDir(customerDir)
	if err != nil {
		return nil, nil // no customer dir yet
	}

	var summaries []ProjectSummary
	for _, e := range entries {
		if e.Name() == "_index.md" {
			continue
		}
		path := filepath.Join(customerDir, e.Name())
		if e.IsDir() {
			// Multi-service project directory
			s, err := readMultiServiceSummary(path, e.Name())
			if err != nil {
				continue
			}
			summaries = append(summaries, s)
		} else if strings.HasSuffix(e.Name(), ".md") {
			// Single-service project file
			s, err := readSingleProjectSummary(path, strings.TrimSuffix(e.Name(), ".md"))
			if err != nil {
				continue
			}
			summaries = append(summaries, s)
		}
	}
	return summaries, nil
}

// ReadServiceSummaries reads all service files in a multi-service project directory.
func ReadServiceSummaries(projectDir string) ([]ServiceSummary, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}
	var summaries []ServiceSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "_index.md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(projectDir, e.Name()))
		if err != nil {
			continue
		}
		entry, err := wiki.ParseServiceEntry(data)
		if err != nil {
			continue
		}
		summaries = append(summaries, ServiceSummary{
			Name:         entry.Meta.Service,
			Tags:         entry.Meta.Tags,
			Purpose:      wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Purpose"), 500),
			Integrations: wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Integrations"), 500),
		})
	}
	return summaries, nil
}

func readSingleProjectSummary(path, name string) (ProjectSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectSummary{}, err
	}
	entry, err := wiki.ParseProjectEntry(data)
	if err != nil {
		return ProjectSummary{}, err
	}
	return ProjectSummary{
		Name:           name,
		Tags:           entry.Meta.Tags,
		IsMultiService: false,
		Domain:         wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Domain"), 500),
		Architecture:   wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Architecture"), 500),
		Integrations:   wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Integrations"), 500),
		TechStack:      wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Tech Stack"), 500),
	}, nil
}

func readMultiServiceSummary(dirPath, name string) (ProjectSummary, error) {
	services, err := ReadServiceSummaries(dirPath)
	if err != nil {
		return ProjectSummary{}, err
	}
	// Also try reading project _index.md if it exists
	var domain, arch, integrations, techStack string
	indexPath := filepath.Join(dirPath, "_index.md")
	if data, err := os.ReadFile(indexPath); err == nil {
		if entry, err := wiki.ParseMultiProjectEntry(data); err == nil {
			domain = wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Domain"), 500)
			arch = wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Architecture"), 500)
			integrations = wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Integrations"), 500)
			techStack = wiki.TruncateSection(wiki.ExtractSection(entry.Body, "## Tech Stack"), 500)
			_ = techStack // may be empty, that's fine
		}
	}
	// Aggregate tags from all services
	tagSet := make(map[string]bool)
	for _, svc := range services {
		for _, t := range svc.Tags {
			tagSet[t] = true
		}
	}
	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}

	return ProjectSummary{
		Name:           name,
		Tags:           tags,
		IsMultiService: true,
		Services:       services,
		Domain:         domain,
		Architecture:   arch,
		Integrations:   integrations,
		TechStack:      techStack,
	}, nil
}
