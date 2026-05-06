package cmd

import (
	"fmt"
	"os"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/ingestion"
	"github.com/emgiezet/llmwiki/internal/llm"
	"github.com/emgiezet/llmwiki/internal/memory"
	"github.com/emgiezet/llmwiki/internal/validation"
	"github.com/spf13/cobra"
)

func NewMaterializeCmd() *cobra.Command {
	var customer, projectType string

	cmd := &cobra.Command{
		Use:   "materialize <project>",
		Short: "Rebuild wiki from accumulated memory facts (no file scanning, ~10x cheaper)",
		Long: `Recalls all accumulated facts for a project from graymatter memory and
generates or updates the wiki entry. ~5-15K tokens vs 50-100K for ingest.
Requires prior 'absorb' sessions or 'ingest' to populate memory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			if err := validation.NameComponent("project", projectName); err != nil {
				return err
			}
			if err := validation.NameComponentOptional("customer", customer); err != nil {
				return err
			}
			if err := validation.NameComponentOptional("type", projectType); err != nil {
				return err
			}

			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return fmt.Errorf("load global config: %w", err)
			}
			// materialize reconstructs a wiki entry from memory for a
			// specific customer; loading the client baseline lets customer-
			// wide LLM / extraction defaults apply.
			clientCfg, err := config.LoadClientConfig(customer)
			if err != nil {
				return fmt.Errorf("load client config: %w", err)
			}
			cfg := config.Merge(global, clientCfg, config.ProjectConfig{
				Customer: customer,
				Type:     projectType,
			})
			if cfg.AnthropicAPIKey == "" {
				cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}

			if !cfg.MemoryEnabled {
				return fmt.Errorf("memory is not enabled — enable memory_enabled: true in ~/.llmwiki/config.yaml")
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

			// Use CWD as the project dir hint so memory_mode=project opens the
			// right .graymatter/ store when the user is in the project directory.
			cwd, _ := os.Getwd()
			mem, err := memory.NewForProject(cfg, cwd)
			if err != nil {
				return fmt.Errorf("init memory: %w", err)
			}
			defer mem.Close()

			fmt.Fprintf(os.Stderr, "Materializing wiki for %q from memory...\n", projectName)
			if err := ingestion.MaterializeFromMemory(cmd.Context(), projectName, cfg, l, mem); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Done.\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&customer, "customer", "", "Customer name")
	cmd.Flags().StringVar(&projectType, "type", "client", "Project type (client, personal, oss)")
	return cmd
}
