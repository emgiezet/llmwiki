package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emgiezet/llmwiki/internal/config"
	"github.com/emgiezet/llmwiki/internal/validation"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewClientCmd returns the `llmwiki client` subtree. v1.3.0 introduces
// per-customer config at ~/.llmwiki/clients/<customer>.yaml — these
// subcommands help scaffold, inspect, and enumerate those files.
func NewClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Manage per-client baseline configs (~/.llmwiki/clients/<customer>.yaml)",
	}
	cmd.AddCommand(newClientInitCmd(), newClientShowCmd(), newClientListCmd())
	return cmd
}

// clientInitTemplate is the commented YAML scaffold written by
// `client init`. Every field is present-but-blank so users see the
// full shape at a glance.
const clientInitTemplate = `# llmwiki client baseline for %s
# Every project with customer: %s inherits these defaults.
# Remove or comment out any block you don't want to set client-wide.

# status: production            # production | poc | discovery
# llm: codex                    # claude-code | claude-api | ollama | codex | opencode | gemini-cli | pi

# links:
#   github: https://github.com/%s
#   gitlab: https://gitlab.com/%s
#   jira: https://%s.atlassian.net/jira/software/c/projects
#   confluence: https://%s.atlassian.net/wiki/spaces/
#   slack: https://%s.slack.com/archives/
#   # Unknown keys also work — they render as generic links.

# team:
#   lead: "owner@%s.com"
#   oncall_channel: "#%s-ops"
#   escalation: "ops-manager@%s.com"
#   notes: "Additional team-wide context here."

# cost:
#   # Team rate applies to every project unless that project overrides it.
#   team_fte_rate_usd_monthly: 18000
#   notes: "Fully loaded (salary + benefits + overhead)."

# extraction:
#   preset: software            # or minimal | feature | full | status-<x>
#   max_tokens: 4000
`

func newClientInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init <customer>",
		Short: "Scaffold ~/.llmwiki/clients/<customer>.yaml with commented-out defaults",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			customer := args[0]
			if err := validation.NameComponent("customer", customer); err != nil {
				return err
			}
			target := config.DefaultClientConfigPath(customer)
			if _, err := os.Stat(target); err == nil && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", target)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { // #nosec G301 -- user-owned dir
				return err
			}
			content := fmt.Sprintf(clientInitTemplate,
				customer, customer, customer, customer, customer,
				customer, customer, customer, customer, customer)
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil { // #nosec G306 -- user-owned file
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", target)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing client config")
	return cmd
}

func newClientShowCmd() *cobra.Command {
	var projectDir string
	cmd := &cobra.Command{
		Use:   "show <customer>",
		Short: "Print the effective merged config for a customer (global + client + optional project)",
		Long: `Loads global config + the customer's client baseline. When --project is set,
also loads that project's llmwiki.yaml and prints the full 3-way merged
Merged struct as YAML. Useful for verifying what effective settings a
project will see.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			customer := args[0]
			if err := validation.NameComponent("customer", customer); err != nil {
				return err
			}
			global, err := config.LoadGlobalConfig(config.DefaultGlobalConfigPath())
			if err != nil {
				return fmt.Errorf("load global config: %w", err)
			}
			client, err := config.LoadClientConfig(customer)
			if err != nil {
				return fmt.Errorf("load client config: %w", err)
			}
			var project config.ProjectConfig
			if projectDir != "" {
				project, err = config.LoadProjectConfig(projectDir)
				if err != nil {
					return fmt.Errorf("load project config: %w", err)
				}
			} else {
				// Show merged client config against an empty project so users
				// see the baseline.
				project = config.ProjectConfig{Customer: customer}
			}
			merged := config.Merge(global, client, project)

			data, err := yaml.Marshal(merged)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&projectDir, "project", "", "Project directory to merge against (default: none — shows client baseline only)")
	return cmd
}

func newClientListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all customers that have a client config file under ~/.llmwiki/clients/",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			dir := filepath.Join(home, ".llmwiki", "clients")
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintln(cmd.OutOrStdout(), "(no client configs yet — run `llmwiki client init <customer>` to create one)")
					return nil
				}
				return err
			}
			var names []string
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
					continue
				}
				names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
			}
			sort.Strings(names)
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no client configs yet)")
				return nil
			}
			for _, n := range names {
				fmt.Fprintln(cmd.OutOrStdout(), n)
			}
			return nil
		},
	}
}
