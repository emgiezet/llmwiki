package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/mgz/llmwiki/internal/config"
	"github.com/mgz/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tracked projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}
			indexPath := filepath.Join(global.WikiRoot, "_index.md")
			entries, err := wiki.ReadIndex(indexPath)
			if err != nil {
				return fmt.Errorf("read index: %w", err)
			}
			if len(entries) == 0 {
				fmt.Println("No projects tracked yet. Run: llmwiki ingest <path>")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tCUSTOMER\tTYPE\tSTATUS\tWIKI")
			fmt.Fprintln(w, "-------\t--------\t----\t------\t----")
			for _, e := range entries {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.Name, e.Customer, e.Type, e.Status, e.WikiPath)
			}
			return w.Flush()
		},
	}
}
