package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/ingestion"
	"github.com/mgz/llmwiki/internal/memory"
	"github.com/mgz/llmwiki/internal/validation"
	"github.com/spf13/cobra"
)

func NewAbsorbCmd() *cobra.Command {
	var project, customer, note string
	var noteStdin bool
	var fastFail bool

	cmd := &cobra.Command{
		Use:   "absorb <dir>",
		Short: "Extract session facts into memory (no wiki entry, near-zero cost)",
		Long: `Reads recent git commits and an optional note, then stores atomic facts
into graymatter memory. Run at end of session or wire as a CLAUDE.md hook.

Facts accumulate over time. Materialize them into a wiki entry with:
  llmwiki materialize <project>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validation.NameComponentOptional("project", project); err != nil {
				return err
			}
			if err := validation.NameComponentOptional("customer", customer); err != nil {
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
			projCfg, err := config.LoadProjectConfig(projectDir)
			if err != nil {
				return fmt.Errorf("load project config: %w", err)
			}
			cfg := config.Merge(global, projCfg)
			// AnthropicAPIKey is forwarded to memory.NewFromConfig so graymatter can use
			// the Anthropic embedding backend for semantic fact storage.
			if cfg.AnthropicAPIKey == "" {
				cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
			}

			projectName := project
			if projectName == "" {
				projectName = filepath.Base(projectDir)
			}
			resolvedCustomer := customer
			if resolvedCustomer == "" {
				resolvedCustomer = cfg.Customer
			}

			if note != "" && noteStdin {
				return fmt.Errorf("cannot combine --note and --note-stdin")
			}
			if noteStdin {
				const maxNoteSize = 1 << 20 // 1 MiB
				data, err := io.ReadAll(io.LimitReader(os.Stdin, maxNoteSize))
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				if int64(len(data)) >= maxNoteSize {
					return fmt.Errorf("note from stdin exceeds %d bytes", maxNoteSize)
				}
				note = strings.TrimSpace(string(data))
			}

			if !cfg.MemoryEnabled {
				fmt.Fprintln(os.Stderr, "warning: memory not enabled — facts not stored. Set memory_enabled: true in ~/.llmwiki/config.yaml")
				return nil
			}

			if fastFail {
				if err := memory.ProbeLock(cfg.MemoryDir); errors.Is(err, memory.ErrLockBusy) {
					fmt.Fprintln(os.Stderr, "warning: memory db busy — skipping absorb (another llmwiki or graymatter process holds the lock)")
					return nil
				}
			}

			mem, err := memory.NewFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("init memory: %w", err)
			}
			defer mem.Close()

			if err := ingestion.AbsorbSession(cmd.Context(), projectDir, projectName, resolvedCustomer, note, mem); err != nil {
				if errors.Is(err, ingestion.ErrNothingToAbsorb) {
					fmt.Fprintln(os.Stderr, "warning: nothing to absorb — no git history found and no --note provided")
					return nil
				}
				return err
			}

			fmt.Fprintf(os.Stderr, "Absorbed session for %q into memory.\n", projectName)
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Project name (defaults to directory basename)")
	cmd.Flags().StringVar(&customer, "customer", "", "Customer name (defaults to llmwiki.yaml customer)")
	cmd.Flags().StringVar(&note, "note", "", "Free-form description of what was worked on this session")
	cmd.Flags().BoolVar(&noteStdin, "note-stdin", false, "Read the session note from stdin (use when piping from hook scripts)")
	cmd.Flags().BoolVar(&fastFail, "fast-fail", false, "If the memory DB is held by another process, skip instead of waiting (used by the Stop hook)")
	return cmd
}
