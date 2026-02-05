package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
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
				return fmt.Errorf("workspace is required. Use --workspace or -w to specify")
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
	client, err := getAPIClient()
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

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputTable(streams *iostreams.IOStreams, repos []api.RepositoryFull) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "NAME\tDESCRIPTION\tVISIBILITY\tUPDATED"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	// Print rows
	for _, repo := range repos {
		name := truncateString(repo.FullName, 40)
		desc := truncateString(repo.Description, 40)
		visibility := formatVisibility(streams, repo.IsPrivate)
		updated := formatUpdated(repo.UpdatedOn)

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

func formatUpdated(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 30*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
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
