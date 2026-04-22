package ingestion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emgiezet/llmwiki/internal/llm"
	"github.com/emgiezet/llmwiki/internal/wiki"
)

// GenerateMultiProjectIndex generates a project-level _index.md for multi-service projects.
func GenerateMultiProjectIndex(ctx context.Context, wikiRoot, projectType, customer, projectName string, l llm.LLM) error {
	projectDir := filepath.Join(wikiRoot, TypeToDir(projectType), customer, projectName)
	services, err := ReadServiceSummaries(projectDir)
	if err != nil || len(services) == 0 {
		return err
	}

	// Build summaries text for the prompt
	var sb strings.Builder
	for _, svc := range services {
		fmt.Fprintf(&sb, "### %s\n", svc.Name)
		if svc.Purpose != "" {
			fmt.Fprintf(&sb, "Purpose: %s\n", svc.Purpose)
		}
		if svc.Integrations != "" {
			fmt.Fprintf(&sb, "Integrations: %s\n", svc.Integrations)
		}
		if len(svc.Tags) > 0 {
			fmt.Fprintf(&sb, "Tags: %s\n", strings.Join(svc.Tags, ", "))
		}
		sb.WriteString("\n")
	}

	indexPath := filepath.Join(projectDir, "_index.md")
	existing := loadExistingMultiProjectBody(indexPath)

	prompt := BuildMultiProjectIndexPrompt(projectName, sb.String(), existing)
	body, err := l.Generate(ctx, prompt)
	if err != nil {
		return fmt.Errorf("generate project index: %w", err)
	}
	body = scrubLLMResponse(body)

	tags, body := ParseTagsFromBody(body)

	// Collect service names
	var serviceNames []string
	for _, svc := range services {
		serviceNames = append(serviceNames, svc.Name)
	}

	meta := wiki.MultiProjectMeta{
		Name:          projectName,
		Customer:      customer,
		Type:          projectType,
		Status:        "active",
		Services:      serviceNames,
		Tags:          tags,
		LastGenerated: time.Now().UTC(),
	}
	return wiki.WriteMultiProjectEntry(indexPath, meta, "\n"+body+"\n")
}

// GenerateClientIndex generates a client-level _index.md from all project wiki files.
func GenerateClientIndex(ctx context.Context, wikiRoot, customer string, l llm.LLM) error {
	summaries, err := ReadProjectSummaries(wikiRoot, customer)
	if err != nil || len(summaries) == 0 {
		return err
	}

	// Build summaries text for the prompt
	var sb strings.Builder
	for _, proj := range summaries {
		fmt.Fprintf(&sb, "### %s", proj.Name)
		if proj.IsMultiService {
			fmt.Fprintf(&sb, " (multi-service, %d services)", len(proj.Services))
		}
		sb.WriteString("\n")
		if proj.Domain != "" {
			fmt.Fprintf(&sb, "Domain: %s\n", proj.Domain)
		}
		if proj.Architecture != "" {
			fmt.Fprintf(&sb, "Architecture: %s\n", proj.Architecture)
		}
		if proj.TechStack != "" {
			fmt.Fprintf(&sb, "Tech Stack: %s\n", proj.TechStack)
		}
		if len(proj.Tags) > 0 {
			fmt.Fprintf(&sb, "Tags: %s\n", strings.Join(proj.Tags, ", "))
		}
		if proj.IsMultiService {
			fmt.Fprintf(&sb, "Services:\n")
			for _, svc := range proj.Services {
				fmt.Fprintf(&sb, "  - %s: %s\n", svc.Name, svc.Purpose)
			}
		}
		sb.WriteString("\n")
	}

	indexPath := filepath.Join(wikiRoot, "clients", customer, "_index.md")
	existing := loadExistingClientBody(indexPath)

	prompt := BuildClientIndexPrompt(customer, sb.String())
	if existing != "" {
		// Prepend update note manually since BuildClientIndexPrompt doesn't take existing
		prompt = fmt.Sprintf("%s\n\nEXISTING INDEX (update this — preserve accurate information):\n%s", prompt, existing)
	}

	body, err := l.Generate(ctx, prompt)
	if err != nil {
		return fmt.Errorf("generate client index: %w", err)
	}
	body = scrubLLMResponse(body)

	tags, body := ParseTagsFromBody(body)

	var projectNames []string
	for _, p := range summaries {
		projectNames = append(projectNames, p.Name)
	}

	meta := wiki.ClientMeta{
		Customer:      customer,
		Projects:      projectNames,
		Tags:          tags,
		LastGenerated: time.Now().UTC(),
	}
	return wiki.WriteClientEntry(indexPath, meta, "\n"+body+"\n")
}

func loadExistingMultiProjectBody(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	entry, err := wiki.ParseMultiProjectEntry(data)
	if err != nil {
		return ""
	}
	return entry.Body
}

func loadExistingClientBody(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	entry, err := wiki.ParseClientEntry(data)
	if err != nil {
		return ""
	}
	return entry.Body
}
