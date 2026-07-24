package pr

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type commentReplyOptions struct {
	streams   *iostreams.IOStreams
	repo      string
	body      string
	commentID int64
}

// newCmdCommentReply creates the "pr comment reply" command
func newCmdCommentReply(streams *iostreams.IOStreams) *cobra.Command {
	opts := &commentReplyOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "reply <number> --comment <id>",
		Short: "Reply to a comment on a pull request",
		Long: `Reply to an existing comment on a pull request, creating a threaded reply.

The parent comment is identified by its ID (shown by "bb pr comment list").
If the reply body is not provided via --body, an editor is opened for you to
enter the reply text.`,
		Example: `  # Reply to comment 456 on pull request #123 (opens editor)
  bb pr comment reply 123 --comment 456

  # Reply with an inline body
  bb pr comment reply 123 --comment 456 --body "Good point, fixed."

  # Reply to a comment in a specific repository
  bb pr comment reply 123 --comment 456 --repo workspace/repo --body "Done"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.repo == "" {
				opts.repo, _ = cmd.InheritedFlags().GetString("repo")
			}
			return runCommentReply(opts, args)
		},
	}

	cmd.Flags().Int64VarP(&opts.commentID, "comment", "c", 0, "ID of the comment to reply to (required)")
	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "Reply body text")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	_ = cmd.MarkFlagRequired("comment")

	cmd.ValidArgsFunction = cmdutil.CompletePRNumbers
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)

	return cmd
}

func runCommentReply(opts *commentReplyOptions, args []string) error {
	prNum, err := parsePRNumber(args)
	if err != nil {
		return err
	}

	if opts.commentID <= 0 {
		return fmt.Errorf("a valid --comment <id> is required")
	}

	workspace, repoSlug, err := cmdutil.ParseRepository(opts.repo)
	if err != nil {
		return err
	}

	// If no body provided, open editor
	if opts.body == "" {
		body, err := openEditor("")
		if err != nil {
			return fmt.Errorf("failed to get reply: %w", err)
		}
		if body == "" {
			return fmt.Errorf("reply body is required")
		}
		opts.body = body
	}

	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	comment, err := client.ReplyToPRComment(ctx, workspace, repoSlug, int64(prNum), opts.commentID, opts.body)
	if err != nil {
		return fmt.Errorf("failed to reply to comment: %w", err)
	}

	// Print the URL to the new reply
	if comment.Links.HTML.Href != "" {
		fmt.Fprintln(opts.streams.Out, comment.Links.HTML.Href)
	} else {
		fmt.Fprintf(opts.streams.Out, "https://bitbucket.org/%s/%s/pull-requests/%d#comment-%d\n",
			workspace, repoSlug, prNum, comment.ID)
	}

	return nil
}
