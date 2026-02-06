package workspace

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// ListOptions holds the options for the list command
type ListOptions struct {
	Role    string
	Limit   int
	JSON    bool
	Streams *iostreams.IOStreams
}

// NewCmdList creates the workspace list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces you have access to",
		Long: `List Bitbucket workspaces that you are a member of.

You can filter by your role in the workspace (owner, collaborator, or member).`,
		Example: `  # List all workspaces
  bb workspace list

  # List workspaces where you are an owner
  bb workspace list --role owner

  # List with a specific limit
  bb workspace list --limit 10

  # Output as JSON
  bb workspace list --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Role, "role", "r", "", "Filter by role (owner, collaborator, member)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of workspaces to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build list options
	listOpts := &api.WorkspaceListOptions{
		Role:  opts.Role,
		Limit: opts.Limit,
	}

	// Fetch workspaces
	result, err := client.ListWorkspaces(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No workspaces found")
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, result.Values)
	}

	return outputListTable(opts.Streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, memberships []api.WorkspaceMembership) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(memberships))
	for i, m := range memberships {
		ws := m.Workspace
		output[i] = map[string]interface{}{
			"slug":       ws.Slug,
			"name":       ws.Name,
			"uuid":       ws.UUID,
			"role":       m.Permission,
			"is_private": ws.IsPrivate,
		}
		if ws.Links.HTML.Href != "" {
			output[i]["url"] = ws.Links.HTML.Href
		}
	}

	return cmdutil.PrintJSON(streams, output)
}

func outputListTable(streams *iostreams.IOStreams, memberships []api.WorkspaceMembership) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "SLUG\tNAME\tROLE"
	cmdutil.PrintTableHeader(streams, w, header)

	// Print rows
	for _, m := range memberships {
		ws := m.Workspace
		role := formatRole(streams, m.Permission)
		fmt.Fprintf(w, "%s\t%s\t%s\n", ws.Slug, ws.Name, role)
	}

	return w.Flush()
}

func formatRole(streams *iostreams.IOStreams, role string) string {
	if !streams.ColorEnabled() {
		return role
	}

	switch role {
	case "owner":
		return iostreams.Yellow + role + iostreams.Reset
	case "collaborator":
		return iostreams.Cyan + role + iostreams.Reset
	default:
		return role
	}
}
