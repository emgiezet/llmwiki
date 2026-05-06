package cmd

import (
	"fmt"
	"os"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/memory"
	"github.com/spf13/cobra"
)

func NewDrainCmd() *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "absorb-drain",
		Short: "Process queued absorb sessions (created when the memory DB was busy)",
		Long: `Drain the absorb queue at <store_dir>/absorb-queue.jsonl.
Each queued session is handed to graymatter for ingestion. Entries that
fail are re-queued. Safe to run via cron or manually.

In memory_mode=project (default), run from the project directory or
pass --project-dir so the correct .graymatter/ store is targeted.
In memory_mode=global, the global memory_dir is used regardless.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return fmt.Errorf("load global config: %w", err)
			}
			cfg := config.Merge(global, config.ClientConfig{}, config.ProjectConfig{})
			if cfg.AnthropicAPIKey == "" {
				cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			if !cfg.MemoryEnabled {
				fmt.Fprintln(os.Stderr, "warning: memory not enabled — nothing to drain")
				return nil
			}

			// Resolve the store dir consistently with how absorb queued the sessions.
			if projectDir == "" {
				projectDir, _ = os.Getwd()
			}
			memDir := memory.ResolveDir(cfg, projectDir)

			mem, err := memory.NewForProject(cfg, projectDir)
			if err != nil {
				return fmt.Errorf("init memory: %w", err)
			}
			defer mem.Close()
			res, derr := memory.DrainAbsorbQueue(cmd.Context(), memDir, mem)
			if derr != nil {
				return fmt.Errorf("drain: %w", derr)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "processed: %d, requeued: %d, queue: %s\n",
				res.Processed, res.Requeued, res.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", "", "Project directory (default: CWD); used to locate the .graymatter/ store in project mode")
	return cmd
}
