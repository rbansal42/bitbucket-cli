package pr

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/git"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type commentListOptions struct {
	streams  *iostreams.IOStreams
	selector string // PR number or empty for current branch
	repo     string
	jsonOut  bool
}

// newCmdCommentList creates the "pr comment list" command
func newCmdCommentList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &commentListOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:     "list [<number>]",
		Aliases: []string{"ls", "view"},
		Short:   "View comments on a pull request",
		Long: `View the comments on a pull request, including threaded replies.

With no argument, the pull request for the current branch is used. Replies are
shown indented beneath the comment they respond to, and inline (code) comments
show their file path and line.`,
		Example: `  # View comments on pull request #123
  bb pr comment list 123

  # View comments for the current branch's PR
  bb pr comment list

  # View comments in a specific repository
  bb pr comment list 123 --repo workspace/repo

  # Output as JSON
  bb pr comment list 123 --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.selector = args[0]
			}
			if opts.repo == "" {
				opts.repo, _ = cmd.InheritedFlags().GetString("repo")
			}
			return runCommentList(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	cmd.ValidArgsFunction = cmdutil.CompletePRNumbers
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)

	return cmd
}

func runCommentList(opts *commentListOptions) error {
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

	prNum, err := resolveCommentPRNumber(ctx, workspace, repoSlug, opts.selector)
	if err != nil {
		return err
	}

	comments, err := client.ListAllPRComments(ctx, workspace, repoSlug, int64(prNum))
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	if opts.jsonOut {
		return cmdutil.PrintJSON(opts.streams, comments)
	}

	return displayComments(opts.streams, comments, prNum)
}

// resolveCommentPRNumber resolves a PR number from an explicit selector or, when
// empty, from the open pull request for the current branch.
func resolveCommentPRNumber(ctx context.Context, workspace, repoSlug, selector string) (int, error) {
	if selector == "" {
		branch, err := git.GetCurrentBranch()
		if err != nil {
			return 0, fmt.Errorf("could not determine current branch: %w", err)
		}
		return findPRForBranch(ctx, workspace, repoSlug, branch)
	}

	num, err := strconv.Atoi(selector)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request number: %s", selector)
	}
	if num <= 0 {
		return 0, fmt.Errorf("invalid pull request number: must be a positive integer")
	}
	return num, nil
}

// displayComments renders comments as a thread: top-level comments in
// chronological order, each followed by its (recursively nested) replies.
func displayComments(streams *iostreams.IOStreams, comments []api.PRComment, prNum int) error {
	if len(comments) == 0 {
		fmt.Fprintf(streams.Out, "No comments on pull request #%d\n", prNum)
		return nil
	}

	// Group replies by their parent comment ID; collect top-level comments.
	childrenByParent := make(map[int64][]api.PRComment)
	var roots []api.PRComment
	for _, c := range comments {
		if c.Parent != nil && c.Parent.ID != 0 {
			childrenByParent[c.Parent.ID] = append(childrenByParent[c.Parent.ID], c)
		} else {
			roots = append(roots, c)
		}
	}

	byCreated := func(cs []api.PRComment) func(i, j int) bool {
		return func(i, j int) bool { return cs[i].CreatedOn.Before(cs[j].CreatedOn) }
	}
	sort.SliceStable(roots, byCreated(roots))
	for id := range childrenByParent {
		sort.SliceStable(childrenByParent[id], byCreated(childrenByParent[id]))
	}

	bold := streams.ColorFunc(iostreams.Bold)
	cyan := streams.ColorFunc(iostreams.Cyan)
	yellow := streams.ColorFunc(iostreams.Yellow)

	fmt.Fprintf(streams.Out, "%s\n\n", bold(fmt.Sprintf("Comments on pull request #%d (%d total)", prNum, len(comments))))

	var render func(c api.PRComment, depth int)
	render = func(c api.PRComment, depth int) {
		indent := strings.Repeat("  ", depth)

		author := cmdutil.GetUserDisplayName(&c.User)
		fmt.Fprintf(streams.Out, "%s%s\n", indent, cyan(fmt.Sprintf("@%s · %s · id %d", author, cmdutil.TimeAgo(c.CreatedOn), c.ID)))

		if c.Inline != nil && c.Inline.Path != "" {
			loc := c.Inline.Path
			if c.Inline.To > 0 {
				loc = fmt.Sprintf("%s:%d", c.Inline.Path, c.Inline.To)
			}
			fmt.Fprintf(streams.Out, "%s%s\n", indent, yellow(loc))
		}

		body := strings.TrimSpace(c.Content.Raw)
		if body == "" {
			body = "(no content)"
		}
		for _, line := range strings.Split(body, "\n") {
			fmt.Fprintf(streams.Out, "%s  %s\n", indent, line)
		}
		fmt.Fprintln(streams.Out)

		for _, child := range childrenByParent[c.ID] {
			render(child, depth+1)
		}
	}

	for _, r := range roots {
		render(r, 0)
	}

	return nil
}
