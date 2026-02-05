package snippet

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
	Role      string // owner, contributor, member
	Limit     int
	JSON      bool
	Streams   *iostreams.IOStreams
}

// NewCmdList creates the snippet list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snippets in a workspace",
		Long: `List snippets in a Bitbucket workspace.

Snippets are workspace-scoped and can be filtered by your role.`,
		Example: `  # List all snippets in a workspace
  bb snippet list --workspace myworkspace

  # List only your snippets
  bb snippet list --workspace myworkspace --role owner

  # Limit the number of snippets shown
  bb snippet list --workspace myworkspace --limit 10

  # Output as JSON
  bb snippet list --workspace myworkspace --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().StringVar(&opts.Role, "role", "", "Filter by role: owner, contributor, member")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of snippets to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	cmd.MarkFlagRequired("workspace")

	return cmd
}

// validRoles are the valid values for the --role flag
var validRoles = map[string]bool{
	"owner":       true,
	"contributor": true,
	"member":      true,
}

func runList(ctx context.Context, opts *ListOptions) error {
	// Validate workspace
	if err := parseWorkspace(opts.Workspace); err != nil {
		return err
	}

	// Validate role if provided
	if opts.Role != "" && !validRoles[opts.Role] {
		return fmt.Errorf("invalid role %q: must be one of owner, contributor, member", opts.Role)
	}

	// Get API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build list options
	listOpts := &api.SnippetListOptions{
		Role:  opts.Role,
		Limit: opts.Limit,
	}

	// Fetch snippets
	result, err := client.ListSnippets(ctx, opts.Workspace, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list snippets: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No snippets found in workspace %s", opts.Workspace)
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, result.Values)
	}

	return outputListTable(opts.Streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, snippets []api.Snippet) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(snippets))
	for i, snippet := range snippets {
		output[i] = map[string]interface{}{
			"id":         fmt.Sprintf("%d", snippet.ID),
			"title":      snippet.Title,
			"is_private": snippet.IsPrivate,
			"updated_on": snippet.UpdatedOn,
		}
		if snippet.Owner != nil {
			output[i]["owner"] = snippet.Owner.DisplayName
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputListTable(streams *iostreams.IOStreams, snippets []api.Snippet) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "ID\tTITLE\tVISIBILITY\tUPDATED"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	// Print rows
	for _, snippet := range snippets {
		id := fmt.Sprintf("%d", snippet.ID)
		title := truncateString(snippet.Title, 40)
		if title == "" {
			title = "(untitled)"
		}

		visibility := "public"
		if snippet.IsPrivate {
			visibility = "private"
		}

		updated := formatTime(snippet.UpdatedOn)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, title, visibility, updated)
	}

	return w.Flush()
}

// formatTime formats an ISO 8601 timestamp to a human-readable format
func formatTime(isoTime string) string {
	if isoTime == "" {
		return ""
	}

	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		// Try alternative format
		t, err = time.Parse("2006-01-02T15:04:05.000000-07:00", isoTime)
		if err != nil {
			return isoTime
		}
	}

	// Format as relative time or date
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins <= 1 {
			return "just now"
		}
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}
