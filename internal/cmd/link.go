package cmd

import (
	"fmt"
	"os"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link",
		Short: "Add cross-reference links between wiki files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Linking wiki files in %s...\n", global.WikiRoot)
			if err := wiki.LinkWikiFiles(global.WikiRoot); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Done.\n")
			return nil
		},
	}
}
