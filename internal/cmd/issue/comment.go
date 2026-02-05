package issue

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type commentOptions struct {
	streams *iostreams.IOStreams
	repo    string
	body    string
}

// NewCmdComment creates the comment command
func NewCmdComment(streams *iostreams.IOStreams) *cobra.Command {
	opts := &commentOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "comment <issue-id>",
		Short: "Add a comment to an issue",
		Long:  `Add a comment to an issue.`,
		Example: `  # Add a comment to issue #123
  bb issue comment 123 --body "This is a comment"

  # Add a comment to an issue in a specific repository
  bb issue comment 123 --repo workspace/repo --body "Working on this"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComment(opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runComment(opts *commentOptions, args []string) error {
	issueID, err := parseIssueID(args)
	if err != nil {
		return err
	}

	workspace, repoSlug, err := cmdutil.ParseRepository(opts.repo)
	if err != nil {
		return err
	}

	// If no body provided, require --body flag
	if opts.body == "" {
		return fmt.Errorf("comment body required, use --body flag")
	}

	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Add the comment
	_, err = client.CreateIssueComment(ctx, workspace, repoSlug, issueID, opts.body)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	opts.streams.Success("Added comment to issue #%d", issueID)
	return nil
}
