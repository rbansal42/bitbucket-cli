package project

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// listOptions holds the options for the list command
type listOptions struct {
	Workspace string
	Limit     int
	JSON      bool
	Streams   *iostreams.IOStreams
}

// NewCmdList creates the project list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &listOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects in a workspace",
		Long: `List projects in a Bitbucket workspace.

This command shows projects you have access to in the specified workspace.`,
		Example: `  # List projects in a workspace
  bb project list --workspace myworkspace

  # List with a specific limit
  bb project list -w myworkspace --limit 10

  # Output as JSON
  bb project list -w myworkspace --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Workspace == "" {
				defaultWs, err := config.GetDefaultWorkspace()
				if err == nil && defaultWs != "" {
					opts.Workspace = defaultWs
				}
			}
			if opts.Workspace == "" {
				return fmt.Errorf("workspace is required. Use --workspace or -w to specify, or set a default with 'bb workspace set-default'")
			}
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of projects to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Build list options
	listOpts := &api.ProjectListOptions{
		Limit: opts.Limit,
	}

	// Fetch projects
	result, err := client.ListProjects(ctx, opts.Workspace, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No projects found in workspace %s", opts.Workspace)
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, result.Values)
	}

	return outputListTable(opts.Streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, projects []api.ProjectFull) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(projects))
	for i, proj := range projects {
		output[i] = map[string]interface{}{
			"key":         proj.Key,
			"name":        proj.Name,
			"description": proj.Description,
			"is_private":  proj.IsPrivate,
			"uuid":        proj.UUID,
			"created_on":  proj.CreatedOn,
			"updated_on":  proj.UpdatedOn,
			"url":         proj.Links.HTML.Href,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputListTable(streams *iostreams.IOStreams, projects []api.ProjectFull) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "KEY\tNAME\tDESCRIPTION\tVISIBILITY"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	// Print rows
	for _, proj := range projects {
		key := proj.Key
		name := truncateString(proj.Name, 30)
		desc := truncateString(proj.Description, 40)
		visibility := formatVisibility(streams, proj.IsPrivate)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", key, name, desc, visibility)
	}

	return w.Flush()
}

func formatVisibility(streams *iostreams.IOStreams, isPrivate bool) string {
	if isPrivate {
		if streams.ColorEnabled() {
			return iostreams.Yellow + "private" + iostreams.Reset
		}
		return "private"
	}

	if streams.ColorEnabled() {
		return iostreams.Green + "public" + iostreams.Reset
	}
	return "public"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
