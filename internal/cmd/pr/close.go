package pr

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

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
		Use:   "close [<number>]",
		Short: "Close a pull request",
		Long: `Close (decline) a pull request.

This command declines the specified pull request, which closes it without merging.
Optionally, you can add a comment explaining why the PR is being closed.`,
		Example: `  # Close pull request #123
  bb pr close 123

  # Close with a comment
  bb pr close 123 --comment "Closing in favor of #456"

  # Close a PR in a specific repository
  bb pr close 123 --repo workspace/repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClose(opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.comment, "comment", "c", "", "Add a closing comment")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runClose(opts *closeOptions, args []string) error {
	prNum, err := parsePRNumber(args)
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

	ctx := context.Background()

	// If comment provided, add it first
	if opts.comment != "" {
		commentPath := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, prNum)
		commentBody := map[string]interface{}{
			"content": map[string]string{
				"raw": opts.comment,
			},
		}
		if _, err := client.Post(ctx, commentPath, commentBody); err != nil {
			return fmt.Errorf("failed to add comment: %w", err)
		}
	}

	// Decline the PR
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline", workspace, repoSlug, prNum)
	if _, err := client.Post(ctx, path, nil); err != nil {
		return fmt.Errorf("failed to close pull request: %w", err)
	}

	opts.streams.Success("Closed pull request #%d", prNum)
	return nil
}
