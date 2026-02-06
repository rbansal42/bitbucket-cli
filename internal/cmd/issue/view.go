package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/browser"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type viewOptions struct {
	streams  *iostreams.IOStreams
	repo     string
	web      bool
	comments bool
	jsonOut  bool
}

// NewCmdView creates the issue view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &viewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view <issue-id>",
		Short: "View an issue",
		Long: `Display the details of an issue.

Shows the issue title, state, kind, priority, reporter, assignee,
content, and other metadata. Use --comments to also show comments.`,
		Example: `  # View issue #123
  bb issue view 123

  # View issue with comments
  bb issue view 123 --comments

  # Open issue in browser
  bb issue view 123 --web

  # Output as JSON
  bb issue view 123 --json

  # View issue in a specific repository
  bb issue view 123 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runView(opts, args)
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open the issue in a web browser")
	cmd.Flags().BoolVarP(&opts.comments, "comments", "c", false, "Show issue comments")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runView(opts *viewOptions, args []string) error {
	// Parse issue ID
	issueID, err := parseIssueID(args)
	if err != nil {
		return err
	}

	// Resolve repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.repo)
	if err != nil {
		return err
	}

	// Get authenticated client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch issue details
	issue, err := client.GetIssue(ctx, workspace, repoSlug, issueID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Handle --web flag
	if opts.web {
		if issue.Links != nil && issue.Links.HTML != nil {
			if err := browser.Open(issue.Links.HTML.Href); err != nil {
				return fmt.Errorf("could not open browser: %w", err)
			}
			opts.streams.Success("Opened %s in your browser", issue.Links.HTML.Href)
			return nil
		}
		return fmt.Errorf("no URL available for this issue")
	}

	// Fetch comments if requested
	var comments []api.IssueComment
	if opts.comments || opts.jsonOut {
		commentsResult, err := client.ListIssueComments(ctx, workspace, repoSlug, issueID)
		if err == nil {
			comments = commentsResult.Values
		}
	}

	// Handle --json flag
	if opts.jsonOut {
		return outputViewJSON(opts.streams, issue, comments)
	}

	// Display formatted output
	return displayIssue(opts.streams, issue, comments, opts.comments)
}

func outputViewJSON(streams *iostreams.IOStreams, issue *api.Issue, comments []api.IssueComment) error {
	output := map[string]interface{}{
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

	if issue.Content != nil {
		output["content"] = issue.Content.Raw
	}

	if issue.Links != nil && issue.Links.HTML != nil {
		output["url"] = issue.Links.HTML.Href
	}

	if len(comments) > 0 {
		commentList := make([]map[string]interface{}, len(comments))
		for i, c := range comments {
			commentList[i] = map[string]interface{}{
				"id":         c.ID,
				"user":       cmdutil.GetUserDisplayName(c.User),
				"created_on": c.CreatedOn,
				"updated_on": c.UpdatedOn,
			}
			if c.Content != nil {
				commentList[i]["content"] = c.Content.Raw
			}
		}
		output["comments"] = commentList
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func displayIssue(streams *iostreams.IOStreams, issue *api.Issue, comments []api.IssueComment, showComments bool) error {
	// Title with ID
	fmt.Fprintf(streams.Out, "#%d: %s\n", issue.ID, issue.Title)
	fmt.Fprintln(streams.Out)

	// State, Kind, Priority
	fmt.Fprintf(streams.Out, "State:    %s\n", formatIssueState(streams, issue.State))
	fmt.Fprintf(streams.Out, "Kind:     %s\n", formatIssueKind(streams, issue.Kind))
	fmt.Fprintf(streams.Out, "Priority: %s\n", formatIssuePriority(streams, issue.Priority))
	fmt.Fprintln(streams.Out)

	// Reporter and Assignee
	fmt.Fprintf(streams.Out, "Reporter: %s\n", cmdutil.GetUserDisplayName(issue.Reporter))
	fmt.Fprintf(streams.Out, "Assignee: %s\n", cmdutil.GetUserDisplayName(issue.Assignee))
	fmt.Fprintln(streams.Out)

	// Votes
	if issue.Votes > 0 {
		fmt.Fprintf(streams.Out, "Votes:    %d\n", issue.Votes)
		fmt.Fprintln(streams.Out)
	}

	// Content/Description
	if issue.Content != nil && issue.Content.Raw != "" {
		fmt.Fprintln(streams.Out, "Description:")
		fmt.Fprintln(streams.Out, issue.Content.Raw)
		fmt.Fprintln(streams.Out)
	}

	// Timestamps
	fmt.Fprintf(streams.Out, "Created:  %s\n", cmdutil.TimeAgo(issue.CreatedOn))
	fmt.Fprintf(streams.Out, "Updated:  %s\n", cmdutil.TimeAgo(issue.UpdatedOn))

	// URL
	if issue.Links != nil && issue.Links.HTML != nil {
		fmt.Fprintln(streams.Out)
		fmt.Fprintf(streams.Out, "View in browser: %s\n", issue.Links.HTML.Href)
	}

	// Comments
	if showComments && len(comments) > 0 {
		fmt.Fprintln(streams.Out)
		fmt.Fprintf(streams.Out, "--- Comments (%d) ---\n", len(comments))
		fmt.Fprintln(streams.Out)

		for _, comment := range comments {
			author := cmdutil.GetUserDisplayName(comment.User)
			timestamp := cmdutil.TimeAgo(comment.CreatedOn)

			if streams.ColorEnabled() {
				fmt.Fprintf(streams.Out, "%s%s%s commented %s:\n", iostreams.Bold, author, iostreams.Reset, timestamp)
			} else {
				fmt.Fprintf(streams.Out, "%s commented %s:\n", author, timestamp)
			}

			if comment.Content != nil && comment.Content.Raw != "" {
				fmt.Fprintln(streams.Out, comment.Content.Raw)
			}
			fmt.Fprintln(streams.Out)
		}
	}

	return nil
}
