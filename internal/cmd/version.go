package cmd

import (
	"fmt"

	"github.com/emgiezet/llmwiki/internal/version"
	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the llmwiki version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "llmwiki %s (%s, built %s)\n",
				version.Version, version.Commit, version.Date)
			return nil
		},
	}
}
