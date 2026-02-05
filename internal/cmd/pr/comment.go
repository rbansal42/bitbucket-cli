package pr

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
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
		Use:   "comment [<number>]",
		Short: "Add a comment to a pull request",
		Long: `Add a comment to a pull request.

If the comment body is not provided via --body, an editor will be opened
for you to enter the comment text.`,
		Example: `  # Add a comment to pull request #123 (opens editor)
  bb pr comment 123

  # Add a comment with body
  bb pr comment 123 --body "This looks great!"

  # Add a comment to a PR in a specific repository
  bb pr comment 123 --repo workspace/repo --body "LGTM"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComment(opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "Comment body text")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runComment(opts *commentOptions, args []string) error {
	prNum, err := parsePRNumber(args)
	if err != nil {
		return err
	}

	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	// If no body provided, open editor
	if opts.body == "" {
		body, err := openEditor("")
		if err != nil {
			return fmt.Errorf("failed to get comment: %w", err)
		}
		if body == "" {
			return fmt.Errorf("comment body is required")
		}
		opts.body = body
	}

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Add the comment
	commentPath := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, prNum)
	commentBody := map[string]interface{}{
		"content": map[string]string{
			"raw": opts.body,
		},
	}

	resp, err := client.Post(ctx, commentPath, commentBody)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	// Parse response to get comment ID
	comment, err := api.ParseResponse[*PRComment](resp)
	if err != nil {
		// Still print success even if we can't parse the comment ID
		opts.streams.Success("Added comment to pull request #%d", prNum)
		return nil
	}

	// Print the URL to the comment
	if comment.Links.HTML.Href != "" {
		fmt.Fprintln(opts.streams.Out, comment.Links.HTML.Href)
	} else {
		// Construct the URL manually
		fmt.Fprintf(opts.streams.Out, "https://bitbucket.org/%s/%s/pull-requests/%d#comment-%d\n",
			workspace, repoSlug, prNum, comment.ID)
	}

	return nil
}
