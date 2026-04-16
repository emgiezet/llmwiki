package cmd

import "github.com/spf13/cobra"

func NewIngestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ingest <path>",
		Short: "Scan a project directory and update wiki entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: wire ingestion
		},
	}
}
