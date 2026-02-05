package pr

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

type reviewOptions struct {
	streams        *iostreams.IOStreams
	repo           string
	approve        bool
	requestChanges bool
	comment        bool
	body           string
}

// NewCmdReview creates the review command
func NewCmdReview(streams *iostreams.IOStreams) *cobra.Command {
	opts := &reviewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "review [<number>]",
		Short: "Review a pull request",
		Long: `Add a review to a pull request.

You can approve a pull request, request changes, or just add a review comment.
At least one action flag (--approve, --request-changes, or --comment) must be specified.`,
		Example: `  # Approve a pull request
  bb pr review 123 --approve

  # Request changes with a comment
  bb pr review 123 --request-changes --body "Please fix the tests"

  # Add a review comment (opens editor if no body provided)
  bb pr review 123 --comment

  # Add a review comment with body
  bb pr review 123 --comment --body "Looks good overall"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview(opts, args)
		},
	}

	cmd.Flags().BoolVarP(&opts.approve, "approve", "a", false, "Approve the pull request")
	cmd.Flags().BoolVarP(&opts.requestChanges, "request-changes", "r", false, "Request changes on the pull request")
	cmd.Flags().BoolVarP(&opts.comment, "comment", "c", false, "Add a review comment")
	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "Review comment body")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runReview(opts *reviewOptions, args []string) error {
	// Validate that at least one action is specified
	if !opts.approve && !opts.requestChanges && !opts.comment {
		return fmt.Errorf("please specify an action: --approve, --request-changes, or --comment")
	}

	// Can't approve and request changes at the same time
	if opts.approve && opts.requestChanges {
		return fmt.Errorf("cannot use --approve and --request-changes together")
	}

	prNum, err := parsePRNumber(args)
	if err != nil {
		return err
	}

	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// If comment flag is set and no body provided, open editor
	if opts.comment && opts.body == "" {
		body, err := openEditor("")
		if err != nil {
			return fmt.Errorf("failed to get comment: %w", err)
		}
		if body == "" {
			return fmt.Errorf("comment body is required")
		}
		opts.body = body
	}

	// Add comment if body is provided (for any action)
	if opts.body != "" {
		commentPath := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, prNum)
		commentBody := map[string]interface{}{
			"content": map[string]string{
				"raw": opts.body,
			},
		}
		if _, err := client.Post(ctx, commentPath, commentBody); err != nil {
			return fmt.Errorf("failed to add comment: %w", err)
		}
	}

	// Handle approve or request changes
	if opts.approve {
		path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve", workspace, repoSlug, prNum)
		if _, err := client.Post(ctx, path, nil); err != nil {
			return fmt.Errorf("failed to approve pull request: %w", err)
		}
		opts.streams.Success("Approved pull request #%d", prNum)
	} else if opts.requestChanges {
		path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/request-changes", workspace, repoSlug, prNum)
		if _, err := client.Post(ctx, path, nil); err != nil {
			return fmt.Errorf("failed to request changes: %w", err)
		}
		opts.streams.Success("Requested changes on pull request #%d", prNum)
	} else if opts.comment {
		opts.streams.Success("Added review comment to pull request #%d", prNum)
	}

	return nil
}
