package pr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/browser"
	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
)

type createOptions struct {
	streams          *iostreams.IOStreams
	title            string
	body             string
	baseBranch       string
	headBranch       string
	reviewers        []string
	fill             bool
	draft            bool
	web              bool
	noMaintainerEdit bool
	repo             string
}

// NewCmdCreate creates the create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &createOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		Long: `Create a pull request from the current branch.

The current branch will be used as the source branch. By default, the destination
branch is the repository's default branch (usually main or master).

If --title is not provided, you will be prompted to enter a title interactively.
If --body is not provided, an editor will open for you to write the description.`,
		Example: `  # Create a pull request interactively
  bb pr create

  # Create a pull request with title and body
  bb pr create --title "Add new feature" --body "Description of changes"

  # Create a pull request with auto-filled title from commits
  bb pr create --fill

  # Create a pull request to a specific base branch
  bb pr create --base develop

  # Create a pull request with reviewers
  bb pr create --title "My PR" --reviewer user1 --reviewer user2

  # Create and open in browser
  bb pr create --title "My PR" --web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Title of the pull request")
	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "Body/description of the pull request")
	cmd.Flags().StringVar(&opts.baseBranch, "base", "", "Base branch (destination). Defaults to repository's default branch")
	cmd.Flags().StringVar(&opts.headBranch, "head", "", "Head branch (source). Defaults to current branch")
	cmd.Flags().StringArrayVarP(&opts.reviewers, "reviewer", "r", nil, "Add reviewer by username (can be repeated)")
	cmd.Flags().BoolVar(&opts.fill, "fill", false, "Auto-fill title and body from commits")
	cmd.Flags().BoolVarP(&opts.draft, "draft", "d", false, "Create as draft (adds [DRAFT] prefix to title)")
	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open the created pull request in the browser")
	cmd.Flags().BoolVar(&opts.noMaintainerEdit, "no-maintainer-edit", false, "Disable maintainer edits (not supported by Bitbucket)")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runCreate(opts *createOptions) error {
	// Resolve repository
	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	// Get current branch as head if not specified
	if opts.headBranch == "" {
		opts.headBranch, err = git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("could not determine current branch: %w", err)
		}
	}

	// Prevent creating PR from main/master
	if opts.headBranch == "main" || opts.headBranch == "master" {
		return fmt.Errorf("cannot create a pull request from branch %q - please switch to a feature branch", opts.headBranch)
	}

	// Get authenticated client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get default branch if base not specified
	if opts.baseBranch == "" {
		defaultBranch, err := getDefaultBranch(ctx, client, workspace, repoSlug)
		if err != nil {
			opts.streams.Warning("Could not determine default branch, using 'main': %v", err)
			opts.baseBranch = "main"
		} else {
			opts.baseBranch = defaultBranch
		}
	}

	// Check if PR already exists for this branch
	existingPR, _ := findExistingPR(ctx, client, workspace, repoSlug, opts.headBranch)
	if existingPR != nil {
		return fmt.Errorf("a pull request already exists for branch %q: %s", opts.headBranch, existingPR.Links.HTML.Href)
	}

	// Handle --fill flag
	if opts.fill {
		fillFromCommits(opts)
	}

	// Interactive mode: prompt for title if not provided
	if opts.title == "" {
		title, err := promptForTitle(opts.streams)
		if err != nil {
			return err
		}
		if title == "" {
			return fmt.Errorf("title is required")
		}
		opts.title = title
	}

	// Handle draft
	if opts.draft {
		if !strings.HasPrefix(opts.title, "[DRAFT]") && !strings.HasPrefix(opts.title, "[WIP]") {
			opts.title = "[DRAFT] " + opts.title
		}
	}

	// Interactive mode: open editor for body if not provided and stdin is TTY
	if opts.body == "" && opts.streams.IsStdinTTY() && !opts.fill {
		body, err := openEditor(getBodyTemplate(opts))
		if err != nil {
			opts.streams.Warning("Could not open editor: %v", err)
		} else {
			opts.body = cleanupBody(body)
		}
	}

	// Display what we're about to do
	opts.streams.Info("Creating pull request for %s into %s\n", opts.headBranch, opts.baseBranch)

	// Resolve reviewer UUIDs
	var reviewerUUIDs []string
	if len(opts.reviewers) > 0 {
		reviewerUUIDs, err = resolveReviewers(ctx, client, workspace, opts.reviewers)
		if err != nil {
			opts.streams.Warning("Could not resolve some reviewers: %v", err)
		}
	}

	// Create the PR
	createOpts := &api.PRCreateOptions{
		Title:             opts.title,
		Description:       opts.body,
		SourceBranch:      opts.headBranch,
		DestinationBranch: opts.baseBranch,
		CloseSourceBranch: false,
		Reviewers:         reviewerUUIDs,
	}

	pr, err := client.CreatePullRequest(ctx, workspace, repoSlug, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	// Print success message
	fmt.Fprintln(opts.streams.Out)
	fmt.Fprintln(opts.streams.Out, pr.Links.HTML.Href)

	// Open in browser if requested
	if opts.web {
		if err := browser.Open(pr.Links.HTML.Href); err != nil {
			opts.streams.Warning("Could not open browser: %v", err)
		}
	}

	return nil
}

// getDefaultBranch fetches the repository's default branch
func getDefaultBranch(ctx context.Context, client *api.Client, workspace, repoSlug string) (string, error) {
	path := fmt.Sprintf("/repositories/%s/%s", workspace, repoSlug)
	resp, err := client.Get(ctx, path, nil)
	if err != nil {
		return "", err
	}

	var repo struct {
		MainBranch struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
	}
	if err := parseResponseBody(resp.Body, &repo); err != nil {
		return "", err
	}

	if repo.MainBranch.Name == "" {
		return "main", nil
	}
	return repo.MainBranch.Name, nil
}

// parseResponseBody parses JSON response body
func parseResponseBody(body []byte, v interface{}) error {
	return json.Unmarshal(body, v)
}

// findExistingPR checks if there's already an open PR for the given branch
func findExistingPR(ctx context.Context, client *api.Client, workspace, repoSlug, branch string) (*api.PullRequest, error) {
	opts := &api.PRListOptions{
		State: api.PRStateOpen,
		Limit: 100,
	}

	result, err := client.ListPullRequests(ctx, workspace, repoSlug, opts)
	if err != nil {
		return nil, err
	}

	for _, pr := range result.Values {
		if pr.Source.Branch.Name == branch {
			return &pr, nil
		}
	}

	return nil, nil
}

// fillFromCommits fills title and body from git commits
func fillFromCommits(opts *createOptions) {
	// Get commits between base and head
	commits, err := getCommitMessages(opts.baseBranch, opts.headBranch)
	if err != nil {
		return
	}

	if len(commits) == 0 {
		return
	}

	// Use first commit as title
	if opts.title == "" && len(commits) > 0 {
		opts.title = commits[0]
	}

	// Use remaining commits as body
	if opts.body == "" && len(commits) > 1 {
		var bodyLines []string
		for _, commit := range commits[1:] {
			bodyLines = append(bodyLines, "- "+commit)
		}
		opts.body = strings.Join(bodyLines, "\n")
	}
}

// getCommitMessages returns commit messages between base and head
func getCommitMessages(base, head string) ([]string, error) {
	// Try to get commits
	cmd := exec.Command("git", "log", "--format=%s", fmt.Sprintf("origin/%s..%s", base, head))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Fallback: try without origin/ prefix
		cmd = exec.Command("git", "log", "--format=%s", fmt.Sprintf("%s..%s", base, head))
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, err
		}
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var commits []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
	}

	return commits, nil
}

// promptForTitle prompts the user to enter a title
func promptForTitle(streams *iostreams.IOStreams) (string, error) {
	if !streams.IsStdinTTY() {
		return "", fmt.Errorf("--title flag is required when not running interactively")
	}

	fmt.Fprint(streams.Out, "Title: ")

	reader := bufio.NewReader(os.Stdin)
	title, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(title), nil
}

// getBodyTemplate returns a template for the PR body
func getBodyTemplate(opts *createOptions) string {
	return fmt.Sprintf(`
<!-- Describe your changes here -->

## Summary


## Related Issues


---
Branch: %s → %s
`, opts.headBranch, opts.baseBranch)
}

// cleanupBody removes comment lines and trims whitespace
func cleanupBody(body string) string {
	lines := strings.Split(body, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "-->") {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

// resolveReviewers resolves usernames to UUIDs
func resolveReviewers(ctx context.Context, client *api.Client, workspace string, usernames []string) ([]string, error) {
	var uuids []string

	for _, username := range usernames {
		// Try to get user by username
		uuid, err := getUserUUID(ctx, client, workspace, username)
		if err != nil {
			continue // Skip failed lookups
		}
		uuids = append(uuids, uuid)
	}

	return uuids, nil
}

// getUserUUID looks up a user's UUID by username
func getUserUUID(ctx context.Context, client *api.Client, workspace, username string) (string, error) {
	// First try as workspace member
	path := fmt.Sprintf("/workspaces/%s/members", workspace)
	resp, err := client.Get(ctx, path, nil)
	if err == nil {
		var members struct {
			Values []struct {
				User struct {
					UUID     string `json:"uuid"`
					Username string `json:"username"`
					Nickname string `json:"nickname"`
				} `json:"user"`
			} `json:"values"`
		}
		if parseErr := parseJSONResponse(resp.Body, &members); parseErr == nil {
			for _, m := range members.Values {
				if m.User.Username == username || m.User.Nickname == username {
					return m.User.UUID, nil
				}
			}
		}
	}

	// Fallback: try to get user directly
	userPath := fmt.Sprintf("/users/%s", username)
	resp, err = client.Get(ctx, userPath, nil)
	if err != nil {
		return "", fmt.Errorf("user not found: %s", username)
	}

	var user struct {
		UUID string `json:"uuid"`
	}
	if err := parseJSONResponse(resp.Body, &user); err != nil {
		return "", err
	}

	return user.UUID, nil
}

// parseJSONResponse parses JSON response body into the given interface
func parseJSONResponse(body []byte, v interface{}) error {
	return json.Unmarshal(body, v)
}
