package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/ingestion"
	"github.com/emgiezet/llmwiki/internal/llm"
	"github.com/emgiezet/llmwiki/internal/memory"
	"github.com/emgiezet/llmwiki/internal/scanner"
	"github.com/emgiezet/llmwiki/internal/validation"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewIngestCmd() *cobra.Command {
	var service string
	var noMemory bool
	var preset string
	var sectionsFlag []string
	var maxTokens int

	cmd := &cobra.Command{
		Use:   "ingest <path>",
		Short: "Scan a project directory and update wiki entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validation.NameComponentOptional("service", service); err != nil {
				return err
			}

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

			// CLI flag overrides for extraction (highest precedence).
			if cmd.Flags().Changed("preset") {
				cfg.Extraction.Preset = preset
			}
			if cmd.Flags().Changed("sections") {
				cfg.Extraction.Sections = sectionsFlag
			}
			if cmd.Flags().Changed("max-tokens") {
				cfg.Extraction.MaxTokens = maxTokens
			}

			// API key can also come from env
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
				MaxTokens:         cfg.Extraction.MaxTokens,
			})
			if err != nil {
				return fmt.Errorf("init LLM: %w", err)
			}

			// Initialize memory store.
			var mem *memory.Store
			if cfg.MemoryEnabled && !noMemory {
				mem, err = memory.NewFromConfig(cfg)
				if err != nil {
					return fmt.Errorf("init memory: %w", err)
				}
				defer mem.Close()
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

				wikiPath := filepath.Join(cfg.WikiRoot, ingestion.TypeToDir(cfg.Type), cfg.Customer, projectName, service+".md")
				var existingBody string
				if data, readErr := os.ReadFile(wikiPath); readErr == nil {
					if entry, parseErr := wiki.ParseServiceEntry(data); parseErr == nil {
						existingBody = entry.Body
					}
				}

				// Recall previous knowledge for prompt enrichment.
				var recalled string
				if mem != nil {
					recalled, _ = mem.RecallForProject(cmd.Context(), projectName, cfg.Customer)
				}

				serviceSections, err := ingestion.ResolveSections(cfg.Extraction, ingestion.ScopeService)
				if err != nil {
					return fmt.Errorf("resolve service sections: %w", err)
				}

				prompt := ingestion.BuildServicePrompt(service, projectName, scan.Summary, existingBody, recalled, serviceSections, cfg.Extraction.MaxTokens)
				body, genErr := l.Generate(cmd.Context(), prompt)
				if genErr != nil {
					return genErr
				}
				tags, body := ingestion.ParseTagsFromBody(body)
				meta := wiki.ServiceMeta{
					Service:      service,
					Project:      projectName,
					Customer:     cfg.Customer,
					Path:         serviceDir,
					Tags:         tags,
					LastIngested: time.Now().UTC(),
				}
				if writeErr := wiki.WriteServiceEntry(wikiPath, meta, "\n"+body+"\n"); writeErr != nil {
					return writeErr
				}

				// Store facts from this service ingestion.
				if mem != nil {
					_ = mem.RememberServiceIngestion(cmd.Context(), projectName, service, cfg.Customer, body, tags)
				}

				fmt.Fprintf(os.Stderr, "Done. Service wiki updated at %s\n", wikiPath)
				return nil
			}

			fmt.Fprintf(os.Stderr, "Ingesting %s into %s...\n", projectName, cfg.WikiRoot)
			if err := ingestion.IngestProject(cmd.Context(), projectDir, projectName, cfg, l, mem); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Done. Wiki updated at %s\n", cfg.WikiRoot)
			return nil
		},
	}
	cmd.Flags().StringVar(&service, "service", "", "Ingest only a specific service subdirectory")
	cmd.Flags().BoolVar(&noMemory, "no-memory", false, "Disable memory recall/storage for this run")
	cmd.Flags().StringVar(&preset, "preset", "", "Extraction preset (default|minimal|software|feature|full) — overrides llmwiki.yaml")
	cmd.Flags().StringSliceVar(&sectionsFlag, "sections", nil, "Comma-separated section IDs to extract — overrides llmwiki.yaml and --preset")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 0, "Cap LLM output tokens per call (0 = backend default)")
	return cmd
}
