package issue

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type createOptions struct {
	streams  *iostreams.IOStreams
	title    string
	body     string
	kind     string
	priority string
	assignee string
	repo     string
}

// NewCmdCreate creates the issue create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &createOptions{
		streams:  streams,
		kind:     "bug",
		priority: "major",
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long: `Create a new issue in a Bitbucket repository.

If --title is not provided and stdin is a TTY, you will be prompted
to enter a title interactively.`,
		Example: `  # Create an issue interactively
  bb issue create

  # Create an issue with title and body
  bb issue create --title "Bug in login" --body "Users cannot log in"

  # Create an enhancement with specific priority
  bb issue create -t "Add dark mode" -k enhancement -p minor

  # Create and assign to a user
  bb issue create -t "Fix crash" -a username

  # Create in a specific repository
  bb issue create -t "New feature" --repo workspace/repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "Issue title (required if not interactive)")
	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "Issue body/description")
	cmd.Flags().StringVarP(&opts.kind, "kind", "k", "bug", "Issue kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVarP(&opts.priority, "priority", "p", "major", "Priority (trivial, minor, major, critical, blocker)")
	cmd.Flags().StringVarP(&opts.assignee, "assignee", "a", "", "Assignee username")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runCreate(opts *createOptions) error {
	// Resolve repository
	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	// Get authenticated client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Interactive mode: prompt for title if not provided
	if opts.title == "" {
		if !opts.streams.IsStdinTTY() {
			return fmt.Errorf("--title flag is required when not running interactively")
		}

		title, err := promptForTitle(opts.streams)
		if err != nil {
			return err
		}
		if title == "" {
			return fmt.Errorf("title is required")
		}
		opts.title = title
	}

	// Validate kind
	validKinds := map[string]bool{"bug": true, "enhancement": true, "proposal": true, "task": true}
	if !validKinds[opts.kind] {
		return fmt.Errorf("invalid kind %q: must be one of bug, enhancement, proposal, task", opts.kind)
	}

	// Validate priority
	validPriorities := map[string]bool{"trivial": true, "minor": true, "major": true, "critical": true, "blocker": true}
	if !validPriorities[opts.priority] {
		return fmt.Errorf("invalid priority %q: must be one of trivial, minor, major, critical, blocker", opts.priority)
	}

	// Build create options
	createOpts := &api.IssueCreateOptions{
		Title:    opts.title,
		Kind:     opts.kind,
		Priority: opts.priority,
	}

	if opts.body != "" {
		createOpts.Content = &api.Content{Raw: opts.body}
	}

	// Resolve assignee if provided
	if opts.assignee != "" {
		uuid, err := resolveUserUUID(ctx, client, workspace, opts.assignee)
		if err != nil {
			return fmt.Errorf("could not resolve assignee %q: %w", opts.assignee, err)
		}
		createOpts.Assignee = &api.User{UUID: uuid}
	}

	opts.streams.Info("Creating issue in %s/%s...", workspace, repoSlug)

	// Create the issue
	issue, err := client.CreateIssue(ctx, workspace, repoSlug, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// Print success message and URL
	opts.streams.Success("Created issue #%d: %s", issue.ID, issue.Title)
	fmt.Fprintln(opts.streams.Out)
	if issue.Links != nil && issue.Links.HTML != nil {
		fmt.Fprintln(opts.streams.Out, issue.Links.HTML.Href)
	}

	return nil
}


