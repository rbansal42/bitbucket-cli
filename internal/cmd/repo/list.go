package repo

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// ListOptions holds the options for the list command
type ListOptions struct {
	Workspace string
	Limit     int
	Sort      string
	JSON      bool
	Streams   *iostreams.IOStreams
}

// NewCmdList creates the repo list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories in a workspace",
		Long: `List repositories in a Bitbucket workspace.

This command shows repositories you have access to in the specified workspace.
By default, repositories are sorted by last updated time.`,
		Example: `  # List repositories in a workspace
  bb repo list --workspace myworkspace

  # List with a specific limit
  bb repo list -w myworkspace --limit 10

  # Sort by name
  bb repo list -w myworkspace --sort name

  # Output as JSON
  bb repo list -w myworkspace --json`,
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
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of repositories to list")
	cmd.Flags().StringVarP(&opts.Sort, "sort", "s", "-updated_on", "Sort field (name, -updated_on)")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Build list options
	listOpts := &api.RepositoryListOptions{
		Sort:  opts.Sort,
		Limit: opts.Limit,
	}

	// Fetch repositories
	result, err := client.ListRepositories(ctx, opts.Workspace, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No repositories found in workspace %s", opts.Workspace)
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, result.Values)
	}

	return outputTable(opts.Streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, repos []api.RepositoryFull) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(repos))
	for i, repo := range repos {
		output[i] = map[string]interface{}{
			"name":        repo.Name,
			"full_name":   repo.FullName,
			"slug":        repo.Slug,
			"description": repo.Description,
			"is_private":  repo.IsPrivate,
			"language":    repo.Language,
			"updated_on":  repo.UpdatedOn,
			"url":         repo.Links.HTML.Href,
		}
	}

	return cmdutil.PrintJSON(streams, output)
}

func outputTable(streams *iostreams.IOStreams, repos []api.RepositoryFull) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "NAME\tDESCRIPTION\tVISIBILITY\tUPDATED"
	cmdutil.PrintTableHeader(streams, w, header)

	// Print rows
	for _, repo := range repos {
		name := cmdutil.TruncateString(repo.FullName, 40)
		desc := cmdutil.TruncateString(repo.Description, 40)
		visibility := formatVisibility(streams, repo.IsPrivate)
		updated := cmdutil.TimeAgo(repo.UpdatedOn)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, desc, visibility, updated)
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
