package cmd

import (
	"fmt"
	"os"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/memory"
	"github.com/spf13/cobra"
)

func NewDrainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "absorb-drain",
		Short: "Process queued absorb sessions (created when the memory DB was busy)",
		Long: `Drain the absorb queue at <memory_dir>/absorb-queue.jsonl.
Each queued session is handed to graymatter for ingestion. Entries that
fail are re-queued. Safe to run via cron or manually.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return fmt.Errorf("load global config: %w", err)
			}
			cfg := config.Merge(global, config.ProjectConfig{})
			if cfg.AnthropicAPIKey == "" {
				cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			if !cfg.MemoryEnabled {
				fmt.Fprintln(os.Stderr, "warning: memory not enabled — nothing to drain")
				return nil
			}
			mem, err := memory.NewFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("init memory: %w", err)
			}
			defer mem.Close()
			res, derr := memory.DrainAbsorbQueue(cmd.Context(), cfg.MemoryDir, mem)
			if derr != nil {
				return fmt.Errorf("drain: %w", derr)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "processed: %d, requeued: %d, queue: %s\n",
				res.Processed, res.Requeued, res.Path)
			return nil
		},
	}
	return cmd
}
