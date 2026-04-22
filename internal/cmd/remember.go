package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/memory"
	"github.com/mgz/llmwiki/internal/validation"
	"github.com/spf13/cobra"
)

func NewRememberCmd() *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "remember <fact>",
		Short: "Store a fact in memory for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fact := args[0]

			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}
			if global.AnthropicAPIKey == "" {
				global.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			cfg := config.Merge(global, config.ProjectConfig{})
			if !cfg.MemoryEnabled {
				return fmt.Errorf("memory is not enabled; set memory_enabled: true in %s", config.DefaultGlobalConfigPath())
			}

			mem, err := memory.NewFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("init memory: %w", err)
			}
			defer mem.Close()

			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if err := validation.NameComponent("project", project); err != nil {
				return err
			}

			if err := mem.RememberIngestion(cmd.Context(), project, "", fact, nil); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Remembered fact for project %q\n", project)
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Project to associate the fact with (required)")
	return cmd
}

func NewRecallCmd() *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "recall <query>",
		Short: "Recall facts from memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}
			if global.AnthropicAPIKey == "" {
				global.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			cfg := config.Merge(global, config.ProjectConfig{})
			if !cfg.MemoryEnabled {
				return fmt.Errorf("memory is not enabled; set memory_enabled: true in %s", config.DefaultGlobalConfigPath())
			}

			mem, err := memory.NewFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("init memory: %w", err)
			}
			defer mem.Close()

			if err := validation.NameComponentOptional("project", project); err != nil {
				return err
			}

			if project != "" {
				result, recallErr := mem.RecallForProject(cmd.Context(), project, "")
				if recallErr != nil {
					return recallErr
				}
				if result == "" {
					fmt.Println("No facts found.")
					return nil
				}
				fmt.Println(result)
				return nil
			}

			facts, recallErr := mem.RecallForQuery(cmd.Context(), query)
			if recallErr != nil {
				return recallErr
			}
			if len(facts) == 0 {
				fmt.Println("No facts found.")
				return nil
			}
			fmt.Println(strings.Join(facts, "\n"))
			return nil
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "Recall facts for a specific project")
	return cmd
}
