package issue

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type closeOptions struct {
	streams *iostreams.IOStreams
	repo    string
	comment string
}

// NewCmdClose creates the close command
func NewCmdClose(streams *iostreams.IOStreams) *cobra.Command {
	opts := &closeOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "close <issue-id>",
		Short: "Close an issue",
		Long: `Close an issue by setting its state to resolved.

Optionally, you can add a comment explaining why the issue is being closed.`,
		Example: `  # Close issue #42
  bb issue close 42

  # Close with a comment
  bb issue close 42 --comment "Fixed in commit abc123"

  # Close an issue in a specific repository
  bb issue close 42 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClose(opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.comment, "comment", "c", "", "Add a closing comment")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runClose(opts *closeOptions, args []string) error {
	issueID, err := parseIssueID(args)
	if err != nil {
		return err
	}

	workspace, repoSlug, err := cmdutil.ParseRepository(opts.repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// If comment provided, add it first
	if opts.comment != "" {
		_, err := client.CreateIssueComment(ctx, workspace, repoSlug, issueID, opts.comment)
		if err != nil {
			return fmt.Errorf("failed to add comment: %w", err)
		}
	}

	// Update issue state to resolved
	state := "resolved"
	updateOpts := &api.IssueUpdateOptions{
		State: &state,
	}

	_, err = client.UpdateIssue(ctx, workspace, repoSlug, issueID, updateOpts)
	if err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	opts.streams.Success("Closed issue #%d", issueID)
	return nil
}
