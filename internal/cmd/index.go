package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/ingestion"
	"github.com/mgz/llmwiki/internal/llm"
	"github.com/mgz/llmwiki/internal/validation"
	"github.com/spf13/cobra"
)

func NewIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index [customer]",
		Short: "Generate client and project index files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				if err := validation.NameComponent("customer", args[0]); err != nil {
					return err
				}
			}

			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}
			if global.AnthropicAPIKey == "" {
				global.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			l, err := llm.NewLLM(llm.Config{
				Backend:         global.LLM,
				AnthropicAPIKey: global.AnthropicAPIKey,
				OllamaHost:      global.OllamaHost,
			})
			if err != nil {
				return err
			}

			clientsDir := filepath.Join(global.WikiRoot, "clients")
			var customers []string

			if len(args) == 1 {
				customers = []string{args[0]}
			} else {
				// Discover all customers
				entries, err := os.ReadDir(clientsDir)
				if err != nil {
					return fmt.Errorf("read clients dir: %w", err)
				}
				for _, e := range entries {
					if e.IsDir() {
						customers = append(customers, e.Name())
					}
				}
			}

			for _, customer := range customers {
				// First generate multi-project indexes
				customerDir := filepath.Join(clientsDir, customer)
				entries, err := os.ReadDir(customerDir)
				if err != nil {
					continue
				}
				for _, e := range entries {
					if e.IsDir() {
						fmt.Fprintf(os.Stderr, "Generating project index for %s/%s...\n", customer, e.Name())
						if err := ingestion.GenerateMultiProjectIndex(cmd.Context(), global.WikiRoot, "client", customer, e.Name(), l); err != nil {
							fmt.Fprintf(os.Stderr, "  warning: %v\n", err)
						}
					}
				}

				// Then generate client index
				fmt.Fprintf(os.Stderr, "Generating client index for %s...\n", customer)
				if err := ingestion.GenerateClientIndex(cmd.Context(), global.WikiRoot, customer, l); err != nil {
					fmt.Fprintf(os.Stderr, "  warning: %v\n", err)
				}
			}

			fmt.Fprintf(os.Stderr, "Done.\n")
			return nil
		},
	}
}
