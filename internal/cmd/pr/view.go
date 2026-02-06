package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/browser"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/git"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type viewOptions struct {
	streams   *iostreams.IOStreams
	selector  string // PR number, URL, or branch
	repo      string
	web       bool
	jsonOut   bool
	workspace string
	repoSlug  string
}

// NewCmdView creates the view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &viewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view [<number> | <url> | <branch>]",
		Short: "View a pull request",
		Long: `Display the details of a pull request.

With no arguments, the pull request for the current branch is displayed.

You can specify a pull request by number, URL, or branch name.`,
		Example: `  # View the PR for the current branch
  bb pr view

  # View PR by number
  bb pr view 123

  # View PR by URL
  bb pr view https://bitbucket.org/workspace/repo/pull-requests/123

  # View PR by branch
  bb pr view feature/my-branch

  # Open PR in browser
  bb pr view --web

  # Output as JSON
  bb pr view --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.selector = args[0]
			}

			// Get repo from flag or parent flag
			if opts.repo == "" {
				opts.repo, _ = cmd.Flags().GetString("repo")
			}
			if opts.repo == "" {
				opts.repo, _ = cmd.InheritedFlags().GetString("repo")
			}

			return runView(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open the pull request in a web browser")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Select a repository using the WORKSPACE/REPO format")

	return cmd
}

func runView(opts *viewOptions) error {
	// Resolve repository
	var err error
	opts.workspace, opts.repoSlug, err = cmdutil.ParseRepository(opts.repo)
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

	// Resolve PR number from selector
	prNumber, err := resolvePRNumber(ctx, opts)
	if err != nil {
		return err
	}

	// Fetch PR details
	pr, err := getPullRequest(ctx, client, opts.workspace, opts.repoSlug, prNumber)
	if err != nil {
		return err
	}

	// Handle --web flag
	if opts.web {
		if err := browser.Open(pr.Links.HTML.Href); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.streams.Success("Opened %s in your browser", pr.Links.HTML.Href)
		return nil
	}

	// Handle --json flag
	if opts.jsonOut {
		return outputJSON(opts.streams, pr)
	}

	// Display formatted output
	return displayPR(opts.streams, pr)
}

func resolvePRNumber(ctx context.Context, opts *viewOptions) (int, error) {
	// No selector - try to find PR for current branch
	if opts.selector == "" {
		branch, err := git.GetCurrentBranch()
		if err != nil {
			return 0, fmt.Errorf("could not determine current branch: %w", err)
		}
		return findPRForBranch(ctx, opts.workspace, opts.repoSlug, branch)
	}

	// Try as number
	if num, err := strconv.Atoi(opts.selector); err == nil {
		return num, nil
	}

	// Try as URL
	if strings.Contains(opts.selector, "bitbucket.org") {
		return extractPRNumberFromURL(opts.selector)
	}

	// Try as branch name
	return findPRForBranch(ctx, opts.workspace, opts.repoSlug, opts.selector)
}

// extractPRNumberFromURL extracts PR number from a Bitbucket URL
func extractPRNumberFromURL(urlStr string) (int, error) {
	// Pattern: https://bitbucket.org/workspace/repo/pull-requests/123
	pattern := regexp.MustCompile(`/pull-requests/(\d+)`)
	matches := pattern.FindStringSubmatch(urlStr)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not extract PR number from URL: %s", urlStr)
	}
	return strconv.Atoi(matches[1])
}

// findPRForBranch finds an open PR for the given source branch
func findPRForBranch(ctx context.Context, workspace, repoSlug, branch string) (int, error) {
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return 0, err
	}

	// Use Bitbucket's query parameter to filter by source branch
	query := url.Values{}
	query.Set("q", fmt.Sprintf(`source.branch.name="%s" AND state="OPEN"`, branch))
	query.Set("pagelen", "1")

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repoSlug)
	resp, err := client.Get(ctx, path, query)
	if err != nil {
		return 0, fmt.Errorf("failed to search for pull request: %w", err)
	}

	var result struct {
		Values []PullRequest `json:"values"`
		Size   int           `json:"size"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Size == 0 || len(result.Values) == 0 {
		return 0, fmt.Errorf("no open pull request found for branch %q", branch)
	}

	return result.Values[0].ID, nil
}

func outputJSON(streams *iostreams.IOStreams, pr *PullRequest) error {
	return cmdutil.PrintJSON(streams, pr)
}

func displayPR(streams *iostreams.IOStreams, pr *PullRequest) error {
	// Title and state
	fmt.Fprintf(streams.Out, "Title: %s\n", pr.Title)
	fmt.Fprintf(streams.Out, "State: %s\n", strings.ToUpper(pr.State))

	// Author
	authorName := getUserDisplayName(pr.Author)
	fmt.Fprintf(streams.Out, "Author: %s\n", authorName)

	// Description
	fmt.Fprintln(streams.Out)
	if pr.Description != "" {
		fmt.Fprintln(streams.Out, pr.Description)
	} else {
		fmt.Fprintln(streams.Out, "(No description)")
	}
	fmt.Fprintln(streams.Out)

	// Reviewers with approval status
	if len(pr.Participants) > 0 {
		fmt.Fprintln(streams.Out, "Reviewers:")
		for _, p := range pr.Participants {
			if p.Role == "REVIEWER" {
				name := getUserDisplayName(p.User)
				status := "pending"
				if p.Approved {
					status = "approved"
				} else if p.State == "changes_requested" {
					status = "changes requested"
				}
				fmt.Fprintf(streams.Out, "  @%s (%s)\n", name, status)
			}
		}
		fmt.Fprintln(streams.Out)
	}

	// Branch info
	fmt.Fprintf(streams.Out, "Base: %s <- %s\n",
		pr.Destination.Branch.Name,
		pr.Source.Branch.Name)

	// Comments
	fmt.Fprintf(streams.Out, "Comments: %d\n", pr.CommentCount)

	// Created date
	createdAt, err := time.Parse(time.RFC3339, pr.CreatedOn)
	if err == nil {
		fmt.Fprintf(streams.Out, "Created: %s\n", cmdutil.TimeAgo(createdAt))
	}

	return nil
}

// getUserDisplayName returns the best available display name for a user
func getUserDisplayName(user PRUser) string {
	if user.DisplayName != "" {
		return user.DisplayName
	}
	if user.Username != "" {
		return user.Username
	}
	if user.Nickname != "" {
		return user.Nickname
	}
	return "unknown"
}
