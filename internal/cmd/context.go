package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/wiki"
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
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}

			ctx, err := buildContextOutput(global.WikiRoot, projectName, service)
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

func buildContextOutput(wikiRoot, projectName, service string) (string, error) {
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
				return extractServiceSections(entry.Body), nil
			}
		}
		return "", fmt.Errorf("service %q not found in wiki for project %q", service, projectName)
	}

	patterns := []string{
		filepath.Join(wikiRoot, "clients", "*", projectName+".md"),
		filepath.Join(wikiRoot, "clients", "*", projectName, "_index.md"),
		filepath.Join(wikiRoot, "personal", projectName+".md"),
		filepath.Join(wikiRoot, "opensource", projectName+".md"),
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
			return extractProjectSections(entry.Body), nil
		}
	}
	return "", fmt.Errorf("project %q not found in wiki. Run: llmwiki ingest <path>", projectName)
}

func extractProjectSections(body string) string {
	return extractSections(body, []string{"## Domain", "## Services", "## Flows"})
}

func extractServiceSections(body string) string {
	return extractSections(body, []string{"## Purpose", "## API Surface", "## Integrations"})
}

func extractSections(body string, keep []string) string {
	lines := strings.Split(body, "\n")
	var out []string
	inKept := false
	for _, line := range lines {
		isHeader := strings.HasPrefix(line, "## ")
		if isHeader {
			inKept = false
			for _, s := range keep {
				if strings.HasPrefix(line, s) {
					inKept = true
					break
				}
			}
		}
		if inKept {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
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
	return os.WriteFile(path, buf.Bytes(), 0644)
}
