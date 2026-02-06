package issue

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// ListOptions holds the options for the list command
type ListOptions struct {
	State    string
	Kind     string
	Priority string
	Assignee string
	Limit    int
	JSON     bool
	Repo     string
	Streams  *iostreams.IOStreams
}

// NewCmdList creates the issue list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues in a repository",
		Long: `List issues in a Bitbucket repository.

By default, this shows all issues. Use flags to filter by state, kind,
priority, or assignee.`,
		Example: `  # List all issues
  bb issue list

  # List open issues
  bb issue list --state open

  # List bugs
  bb issue list --kind bug

  # List critical issues
  bb issue list --priority critical

  # List issues assigned to a user
  bb issue list --assignee johndoe

  # Limit results
  bb issue list --limit 10

  # Output as JSON
  bb issue list --json

  # List issues in a specific repository
  bb issue list --repo workspace/repo`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "", "Filter by state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed)")
	cmd.Flags().StringVarP(&opts.Kind, "kind", "k", "", "Filter by kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVarP(&opts.Priority, "priority", "p", "", "Filter by priority (trivial, minor, major, critical, blocker)")
	cmd.Flags().StringVarP(&opts.Assignee, "assignee", "a", "", "Filter by assignee username")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of issues to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Parse repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.Repo)
	if err != nil {
		return err
	}

	// Build list options
	listOpts := &api.IssueListOptions{
		State:    opts.State,
		Kind:     opts.Kind,
		Priority: opts.Priority,
		Assignee: opts.Assignee,
		Limit:    opts.Limit,
	}

	// Fetch issues
	result, err := client.ListIssues(ctx, workspace, repoSlug, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No issues found in %s/%s", workspace, repoSlug)
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, result.Values)
	}

	return outputIssueTable(opts.Streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, issues []api.Issue) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(issues))
	for i, issue := range issues {
		output[i] = map[string]interface{}{
			"id":         issue.ID,
			"title":      issue.Title,
			"state":      issue.State,
			"kind":       issue.Kind,
			"priority":   issue.Priority,
			"reporter":   cmdutil.GetUserDisplayName(issue.Reporter),
			"assignee":   cmdutil.GetUserDisplayName(issue.Assignee),
			"votes":      issue.Votes,
			"created_on": issue.CreatedOn,
			"updated_on": issue.UpdatedOn,
		}
		if issue.Links != nil && issue.Links.HTML != nil {
			output[i]["url"] = issue.Links.HTML.Href
		}
	}

	return cmdutil.PrintJSON(streams, output)
}

func outputIssueTable(streams *iostreams.IOStreams, issues []api.Issue) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "#\tTITLE\tSTATE\tKIND\tPRIORITY\tASSIGNEE\tUPDATED"
	cmdutil.PrintTableHeader(streams, w, header)

	// Print rows
	for _, issue := range issues {
		id := fmt.Sprintf("%d", issue.ID)
		title := cmdutil.TruncateString(issue.Title, 40)
		state := formatIssueState(streams, issue.State)
		kind := formatIssueKind(streams, issue.Kind)
		priority := formatIssuePriority(streams, issue.Priority)
		assignee := cmdutil.TruncateString(cmdutil.GetUserDisplayName(issue.Assignee), 15)
		updated := cmdutil.TimeAgo(issue.UpdatedOn)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			id, title, state, kind, priority, assignee, updated)
	}

	return w.Flush()
}
