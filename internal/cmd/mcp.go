package cmd

import (
	"context"
	"errors"
	"io"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/mcpserver"
	"github.com/emgiezet/llmwiki/internal/wiki"
	"github.com/spf13/cobra"
)

func NewMcpCmd() *cobra.Command {
	var wikiRootFlag string

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP server exposing the wiki for agents (stdio)",
		Long: `Run a Model Context Protocol server over stdio so agents can search and
fetch project information that llmwiki has already extracted, without invoking
an LLM. Exposes two tools:

  search_projects(client?, project?)  list matching projects (both filters optional)
  get_project(project, client?, service?)  fetch the full extracted content

Add to .mcp.json:

  { "mcpServers": { "llmwiki": { "command": "llmwiki", "args": ["mcp"] } } }`,
		Args: cobra.NoArgs,
		// The server runs until the client disconnects; usage spam on the
		// resulting EOF would be noise, so suppress it.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return err
			}
			if wikiRootFlag != "" {
				global.WikiRoot = wikiRootFlag
			}
			err = mcpserver.Serve(cmd.Context(), wiki.NewStore(global.WikiRoot))
			// A client disconnect (stdin EOF) or a cancelled context is a
			// normal shutdown, not a failure.
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		},
	}

	cmd.Flags().StringVar(&wikiRootFlag, "wiki-root", "", "Override wiki root directory")
	return cmd
}
