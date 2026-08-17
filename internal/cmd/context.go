package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/memory"
	"github.com/emgiezet/llmwiki/internal/validation"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewContextCmd() *cobra.Command {
	var inject string
	var service string
	var noKnowledge bool

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

			// Derive project dir from --inject path (best proxy for project root)
			// or fall back to CWD so memory_mode=project finds the right store.
			projectDir := ""
			if inject != "" {
				projectDir = filepath.Dir(inject)
			} else {
				projectDir, _ = os.Getwd()
			}

			// Load the project (and its client baseline) so the per-project
			// `knowledge:` layer list applies. Missing files are not errors.
			project, err := config.LoadProjectConfig(projectDir)
			if err != nil {
				return fmt.Errorf("load project config: %w", err)
			}
			client, err := config.LoadClientConfig(project.Customer)
			if err != nil {
				return fmt.Errorf("load client config: %w", err)
			}
			cfg := config.Merge(global, client, project)

			layers := cfg.Knowledge
			if noKnowledge {
				layers = nil
			}

			// Initialize memory store if enabled.
			var mem *memory.Store
			if cfg.MemoryEnabled {
				mem, err = memory.NewForProject(cfg, projectDir)
				if err != nil {
					return fmt.Errorf("init memory: %w", err)
				}
				defer mem.Close()
			}

			ctx, err := buildContextOutput(global.WikiRoot, projectName, service, mem, layers)
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
	cmd.Flags().BoolVar(&noKnowledge, "no-knowledge", false, "Omit knowledge layers, printing only project context")
	return cmd
}

// Budgets for the knowledge appended to context output. CLAUDE.md injection
// competes with everything else in the agent's context window, so both a
// per-entry and a total cap apply.
const (
	knowledgeEntryMaxChars = 1500
	knowledgeTotalMaxChars = 6000
)

// knowledgeContextSections are the summary-ish sections preferred when a
// knowledge entry has them (ingest-generated entries do, via the `notes` and
// `research` presets).
var knowledgeContextSections = []string{"## Summary", "## Key Topics", "## Key Points / Findings"}

// renderKnowledgeContext renders the knowledge layers for context injection.
// Each entry is headed by "## Knowledge: <layer>/<topic>", which is what tells
// the agent which layer the content came from.
//
// Layers are read in the given (priority) order and the total budget is spent
// most-specific-first, so when it runs out it's the least specific layer that
// gets dropped.
func renderKnowledgeContext(store *wiki.Store, layers []string) string {
	if len(layers) == 0 {
		return ""
	}
	// A malformed layer name shouldn't kill context generation for the rest,
	// so read layer by layer and skip the ones that error.
	var out strings.Builder
	total := 0
	for _, layer := range layers {
		entries, err := store.SearchKnowledge([]string{layer}, "")
		if err != nil {
			continue
		}
		for _, e := range entries {
			if total >= knowledgeTotalMaxChars {
				return out.String()
			}
			// Prefer the summary sections; hand-authored files have arbitrary
			// headings, so fall back to the whole body.
			text := wiki.ExtractSections(e.Body, knowledgeContextSections)
			if strings.TrimSpace(text) == "" {
				text = e.Body
			}
			text = strings.TrimSpace(wiki.TruncateSection(text, knowledgeEntryMaxChars))
			if text == "" {
				continue
			}
			fmt.Fprintf(&out, "\n## Knowledge: %s/%s\n\n%s\n", e.Layer, e.Topic, text)
			total += len(text)
		}
	}
	return out.String()
}

func buildContextOutput(wikiRoot, projectName, service string, mem *memory.Store, layers []string) (string, error) {
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
				output += renderKnowledgeContext(wiki.NewStore(wikiRoot), layers)
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
			output += renderKnowledgeContext(wiki.NewStore(wikiRoot), layers)
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
