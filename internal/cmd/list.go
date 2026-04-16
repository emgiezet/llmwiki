package cmd

import "github.com/spf13/cobra"

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tracked projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
