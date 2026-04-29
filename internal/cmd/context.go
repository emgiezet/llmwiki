package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/memory"
	"github.com/emgiezet/llmwiki/internal/validation"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewContextCmd() *cobra.Command {
	var inject string
	var service string

	cmd := &cobra.Command{
		Use:   "context <project>",
		Short: "Print wiki context for a project (pipe into CLAUDE.md)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			if err := validation.NameComponent("project", projectName); err != nil {
				return err
			}
			if err := validation.NameComponentOptional("service", service); err != nil {
				return err
			}
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}

			// Initialize memory store if enabled.
			cfg := config.Merge(global, config.ClientConfig{}, config.ProjectConfig{})
			var mem *memory.Store
			if cfg.MemoryEnabled {
				mem, err = memory.NewFromConfig(cfg)
				if err != nil {
					return fmt.Errorf("init memory: %w", err)
				}
				defer mem.Close()
			}

			ctx, err := buildContextOutput(global.WikiRoot, projectName, service, mem)
			if err != nil {
				return err
			}

			if inject == "" {
				fmt.Print(ctx)
				return nil
			}
			return injectIntoFile(inject, ctx)
		},
	}
	cmd.Flags().StringVar(&inject, "inject", "", "Inject into file, replacing <!-- llmwiki:start --> ... <!-- llmwiki:end --> markers")
	cmd.Flags().StringVar(&service, "service", "", "Output context for a specific service only")
	return cmd
}

func buildContextOutput(wikiRoot, projectName, service string, mem *memory.Store) (string, error) {
	if service != "" {
		patterns := []string{
			filepath.Join(wikiRoot, "clients", "*", projectName, service+".md"),
			filepath.Join(wikiRoot, "personal", projectName, service+".md"),
		}
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(pattern)
			for _, p := range matches {
				data, err := os.ReadFile(p)
				if err != nil {
					continue
				}
				entry, err := wiki.ParseServiceEntry(data)
				if err != nil {
					continue
				}
				output := extractServiceSections(entry.Body)
				if recalled, _ := mem.RecallForContext(context.Background(), projectName, entry.Meta.Customer); recalled != "" {
					output += recalled
				}
				return output, nil
			}
		}
		return "", fmt.Errorf("service %q not found in wiki for project %q", service, projectName)
	}

	// The second pattern uses *_index.md so we match both the v1.1.1+
	// "{customer}_{project}_index.md" and the legacy "_index.md" in the
	// project directory.
	patterns := []string{
		filepath.Join(wikiRoot, "clients", "*", projectName+".md"),
		filepath.Join(wikiRoot, "clients", "*", projectName, "*_index.md"),
		filepath.Join(wikiRoot, "personal", projectName+".md"),
		filepath.Join(wikiRoot, "personal", projectName, "*_index.md"),
		filepath.Join(wikiRoot, "opensource", projectName+".md"),
		filepath.Join(wikiRoot, "opensource", projectName, "*_index.md"),
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, p := range matches {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			entry, err := wiki.ParseProjectEntry(data)
			if err != nil {
				continue
			}
			output := extractProjectSections(entry.Body)
			if recalled, _ := mem.RecallForContext(context.Background(), projectName, entry.Meta.Customer); recalled != "" {
				output += recalled
			}
			return output, nil
		}
	}
	return "", fmt.Errorf("project %q not found in wiki. Run: llmwiki ingest <path>", projectName)
}

// extractProjectSections returns the key sections for CLAUDE.md injection.
// Intentionally excludes: System Diagram, Data Model Diagram, Tags (too verbose for context injection).
func extractProjectSections(body string) string {
	return wiki.ExtractSections(body, []string{"## Domain", "## Architecture", "## Services", "## Features", "## Flows"})
}

// extractServiceSections returns the key sections for CLAUDE.md injection.
// Intentionally excludes: System Diagram, Data Model Diagram, Tags (too verbose for context injection).
func extractServiceSections(body string) string {
	return wiki.ExtractSections(body, []string{"## Purpose", "## Architecture", "## API Surface", "## Integrations"})
}

func injectIntoFile(path, content string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	start := []byte("<!-- llmwiki:start -->")
	end := []byte("<!-- llmwiki:end -->")

	si := bytes.Index(data, start)
	ei := bytes.Index(data, end)
	if si == -1 || ei == -1 || si > ei {
		return fmt.Errorf("markers <!-- llmwiki:start --> and <!-- llmwiki:end --> not found in %s", path)
	}

	var buf bytes.Buffer
	buf.Write(data[:si+len(start)])
	buf.WriteString("\n")
	buf.WriteString(content)
	buf.WriteString("\n")
	buf.Write(data[ei:])
	return os.WriteFile(path, buf.Bytes(), 0644) // #nosec G306 -- preserves existing file permissions; CLAUDE.md is world-readable by design
}
