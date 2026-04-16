package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/ingestion"
	"github.com/mgz/llmwiki/internal/llm"
	"github.com/mgz/llmwiki/internal/scanner"
	"github.com/mgz/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewIngestCmd() *cobra.Command {
	var service string

	cmd := &cobra.Command{
		Use:   "ingest <path>",
		Short: "Scan a project directory and update wiki entries",
		Args:  cobra.ExactArgs(1),
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

			// API key can also come from env
			if cfg.AnthropicAPIKey == "" {
				cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}

			l, err := llm.NewLLM(llm.Config{
				Backend:         cfg.LLM,
				OllamaHost:      cfg.OllamaHost,
				OllamaModel:     cfg.OllamaModel,
				AnthropicAPIKey: cfg.AnthropicAPIKey,
			})
			if err != nil {
				return fmt.Errorf("init LLM: %w", err)
			}

			projectName := filepath.Base(projectDir)

			if service != "" {
				serviceDir := filepath.Join(projectDir, service)
				if _, err := os.Stat(serviceDir); err != nil {
					return fmt.Errorf("service directory %q not found", serviceDir)
				}
				scan, scanErr := scanner.ScanProject(serviceDir)
				if scanErr != nil {
					return fmt.Errorf("scan service: %w", scanErr)
				}

				wikiPath := filepath.Join(cfg.WikiRoot, cfg.Type+"s", cfg.Customer, projectName, service+".md")
				var existingBody string
				if data, readErr := os.ReadFile(wikiPath); readErr == nil {
					if entry, parseErr := wiki.ParseServiceEntry(data); parseErr == nil {
						existingBody = entry.Body
					}
				}

				prompt := ingestion.BuildServicePrompt(service, projectName, scan.Summary, existingBody)
				body, genErr := l.Generate(cmd.Context(), prompt)
				if genErr != nil {
					return genErr
				}
				meta := wiki.ServiceMeta{
					Service:      service,
					Project:      projectName,
					Customer:     cfg.Customer,
					Path:         serviceDir,
					LastIngested: time.Now().UTC(),
				}
				if writeErr := wiki.WriteServiceEntry(wikiPath, meta, "\n"+body+"\n"); writeErr != nil {
					return writeErr
				}
				fmt.Fprintf(os.Stderr, "Done. Service wiki updated at %s\n", wikiPath)
				return nil
			}

			fmt.Fprintf(os.Stderr, "Ingesting %s into %s...\n", projectName, cfg.WikiRoot)
			if err := ingestion.IngestProject(cmd.Context(), projectDir, projectName, cfg, l); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Done. Wiki updated at %s\n", cfg.WikiRoot)
			return nil
		},
	}
	cmd.Flags().StringVar(&service, "service", "", "Ingest only a specific service subdirectory")
	return cmd
}
