package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/ingestion"
	"github.com/mgz/llmwiki/internal/llm"
	"github.com/mgz/llmwiki/internal/memory"
	"github.com/mgz/llmwiki/internal/scanner"
	"github.com/mgz/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewDocsCmd() *cobra.Command {
	var write bool
	var target string

	cmd := &cobra.Command{
		Use:   "docs <path>",
		Short: "Generate or update project documentation from wiki knowledge and memory",
		Long: `Reads wiki entries, recalled memory facts, and a fresh code scan to produce
up-to-date documentation. By default updates README.md. Use --target to pick
a different file.

Without --write, prints the generated content to stdout for review.
With --write, overwrites the target file in the project directory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return fmt.Errorf("load global config: %w", err)
			}
			project, err := config.LoadProjectConfig(projectDir)
			if err != nil {
				return fmt.Errorf("load project config: %w", err)
			}
			cfg := config.Merge(global, project)

			if cfg.AnthropicAPIKey == "" {
				cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}

			l, err := llm.NewLLM(llm.Config{
				Backend:           cfg.LLM,
				OllamaHost:        cfg.OllamaHost,
				OllamaModel:       cfg.OllamaModel,
				AllowRemoteOllama: cfg.AllowRemoteOllama,
				AnthropicAPIKey:   cfg.AnthropicAPIKey,
				ClaudeBinaryPath:  cfg.ClaudeBinaryPath,
			})
			if err != nil {
				return fmt.Errorf("init LLM: %w", err)
			}

			projectName := filepath.Base(projectDir)

			// 1. Scan the current project state.
			scan, err := scanner.ScanProject(projectDir)
			if err != nil {
				return fmt.Errorf("scan project: %w", err)
			}

			// 2. Load wiki body if available.
			wikiBody := loadWikiBody(cfg.WikiRoot, cfg.Type, cfg.Customer, projectName)

			// 3. Recall from memory if enabled.
			var recalled string
			if cfg.MemoryEnabled {
				mem, memErr := memory.NewFromConfig(cfg)
				if memErr == nil {
					defer mem.Close()
					recalled, _ = mem.RecallForProject(cmd.Context(), projectName, cfg.Customer)
				}
			}

			// 4. Read existing target file.
			targetFile := target
			targetPath := filepath.Join(projectDir, targetFile)
			var existingDoc string
			if data, readErr := os.ReadFile(targetPath); readErr == nil {
				existingDoc = string(data)
			}

			// 5. Build prompt and generate.
			prompt := ingestion.BuildDocsPrompt(projectName, scan.Summary, wikiBody, recalled, existingDoc, targetFile)
			result, err := l.Generate(cmd.Context(), prompt)
			if err != nil {
				return fmt.Errorf("generate docs: %w", err)
			}
			result = ingestion.ScrubLLMResponse(result)

			if !write {
				fmt.Print(result)
				return nil
			}

			if err := os.WriteFile(targetPath, []byte(result), 0644); err != nil { // #nosec G306 G703 -- target is user-supplied CLI flag; 0644 is intentional for doc files
				return fmt.Errorf("write %s: %w", targetPath, err)
			}
			fmt.Fprintf(os.Stderr, "Updated %s\n", targetPath)
			return nil
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "Write the generated content to the target file (default: print to stdout)")
	cmd.Flags().StringVar(&target, "target", "README.md", "Documentation file to generate/update")
	return cmd
}

// loadWikiBody searches for the project's wiki entry and returns its body.
func loadWikiBody(wikiRoot, projectType, customer, projectName string) string {
	patterns := []string{
		filepath.Join(wikiRoot, "clients", "*", projectName+".md"),
		filepath.Join(wikiRoot, "clients", "*", projectName, "_index.md"),
		filepath.Join(wikiRoot, "personal", projectName+".md"),
		filepath.Join(wikiRoot, "opensource", projectName+".md"),
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, p := range matches {
			data, readErr := os.ReadFile(p)
			if readErr != nil {
				continue
			}
			entry, parseErr := wiki.ParseProjectEntry(data)
			if parseErr != nil {
				continue
			}
			return entry.Body
		}
	}
	return ""
}
