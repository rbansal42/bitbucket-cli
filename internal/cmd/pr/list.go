package pr

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// ListOptions holds the options for the list command
type ListOptions struct {
	State   string
	Author  string
	Limit   int
	JSON    bool
	Repo    string
	Streams *iostreams.IOStreams
}

// NewCmdList creates the pr list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests in a repository",
		Long: `List pull requests in a Bitbucket repository.

By default, this shows open pull requests. Use the --state flag to filter
by state (OPEN, MERGED, DECLINED).`,
		Example: `  # List open pull requests
  bb pr list

  # List merged pull requests
  bb pr list --state MERGED

  # List pull requests by a specific author
  bb pr list --author johndoe

  # List pull requests with limit
  bb pr list --limit 10

  # Output as JSON
  bb pr list --json

  # List PRs for a specific repository
  bb pr list --repo workspace/repo`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.State, "state", "s", "OPEN", "Filter by state: OPEN, MERGED, DECLINED")
	cmd.Flags().StringVarP(&opts.Author, "author", "a", "", "Filter by author username")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of pull requests to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

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

	// Validate state
	state := strings.ToUpper(opts.State)
	if state != "OPEN" && state != "MERGED" && state != "DECLINED" {
		return fmt.Errorf("invalid state: %s (must be OPEN, MERGED, or DECLINED)", opts.State)
	}

	// Build list options
	listOpts := &api.PRListOptions{
		State:  api.PRState(state),
		Author: opts.Author,
		Limit:  opts.Limit,
	}

	// Fetch pull requests
	result, err := client.ListPullRequests(ctx, workspace, repoSlug, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list pull requests: %w", err)
	}

	if len(result.Values) == 0 {
		if opts.Author != "" {
			opts.Streams.Info("No %s pull requests found by %s in %s/%s", strings.ToLower(state), opts.Author, workspace, repoSlug)
		} else {
			opts.Streams.Info("No %s pull requests found in %s/%s", strings.ToLower(state), workspace, repoSlug)
		}
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, result.Values)
	}

	return outputTable(opts.Streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, prs []api.PullRequest) error {
	// Create simplified JSON output
	output := make([]api.PullRequestJSON, len(prs))
	for i := range prs {
		output[i] = api.PullRequestJSON{PullRequest: &prs[i]}
	}

	return cmdutil.PrintJSON(streams, output)
}

func outputTable(streams *iostreams.IOStreams, prs []api.PullRequest) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "ID\tTITLE\tBRANCH\tAUTHOR\tSTATUS"
	cmdutil.PrintTableHeader(streams, w, header)

	// Print rows
	for _, pr := range prs {
		title := cmdutil.TruncateString(pr.Title, 50)
		branch := cmdutil.TruncateString(pr.Source.Branch.Name, 30)
		author := cmdutil.TruncateString(pr.Author.DisplayName, 20)
		status := formatStatus(streams, string(pr.State))

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			pr.ID, title, branch, author, status)
	}

	return w.Flush()
}

func formatStatus(streams *iostreams.IOStreams, state string) string {
	if !streams.ColorEnabled() {
		return state
	}

	switch state {
	case "OPEN":
		return iostreams.Green + state + iostreams.Reset
	case "MERGED":
		return iostreams.Magenta + state + iostreams.Reset
	case "DECLINED":
		return iostreams.Red + state + iostreams.Reset
	default:
		return state
	}
}
