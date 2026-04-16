package cmd

import "github.com/spf13/cobra"

func NewContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "context <project>",
		Short: "Print wiki context for a project (pipe into CLAUDE.md)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
