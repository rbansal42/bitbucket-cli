package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type editOptions struct {
	streams *iostreams.IOStreams
	repo    string
	prID    int64
	title   string
	body    string
	base    string // destination branch
	jsonOut bool
}

// NewCmdEdit creates the edit command
func NewCmdEdit(streams *iostreams.IOStreams) *cobra.Command {
	opts := &editOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "Edit a pull request",
		Long: `Edit the title, description, or destination branch of a pull request.

At least one of --title, --body, or --base must be specified.`,
		Example: `  # Edit PR title
  bb pr edit 123 --title "New title"

  # Edit PR description
  bb pr edit 123 --body "New description"

  # Edit destination branch
  bb pr edit 123 --base develop

  # Edit multiple fields
  bb pr edit 123 --title "New title" --body "New description"

  # Output as JSON
  bb pr edit 123 --title "New title" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pull request number: %s", args[0])
			}
			opts.prID = id
			return runEdit(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")
	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New title for the pull request")
	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "New description for the pull request")
	cmd.Flags().StringVar(&opts.base, "base", "", "New destination branch")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

func runEdit(ctx context.Context, opts *editOptions) error {
	// Validate - at least one field must be specified
	if opts.title == "" && opts.body == "" && opts.base == "" {
		return fmt.Errorf("nothing to edit: specify --title, --body, or --base")
	}

	// Parse repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.repo)
	if err != nil {
		return err
	}

	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build update options
	updateOpts := &api.PRCreateOptions{
		Title:             opts.title,
		Description:       opts.body,
		DestinationBranch: opts.base,
	}

	// Update PR
	pr, err := client.UpdatePullRequest(ctx, workspace, repoSlug, opts.prID, updateOpts)
	if err != nil {
		return fmt.Errorf("failed to update pull request: %w", err)
	}

	// Handle --json flag
	if opts.jsonOut {
		return outputEditJSON(opts.streams, pr)
	}

	// Output success message
	opts.streams.Success("Edited pull request #%d", opts.prID)
	fmt.Fprintln(opts.streams.Out, pr.Links.HTML.Href)

	return nil
}

func outputEditJSON(streams *iostreams.IOStreams, pr *api.PullRequest) error {
	data, err := json.MarshalIndent(api.PullRequestJSON{PullRequest: pr}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}
