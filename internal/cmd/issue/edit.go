package issue

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type editOptions struct {
	streams  *iostreams.IOStreams
	issueID  int
	title    string
	body     string
	kind     string
	priority string
	assignee string
	repo     string

	// Track which flags were explicitly set
	titleSet    bool
	bodySet     bool
	kindSet     bool
	prioritySet bool
	assigneeSet bool
}

// NewCmdEdit creates the issue edit command
func NewCmdEdit(streams *iostreams.IOStreams) *cobra.Command {
	opts := &editOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "edit <issue-id>",
		Short: "Edit an existing issue",
		Long: `Edit an existing issue in a Bitbucket repository.

Only the fields that are explicitly provided will be updated.
Use an empty string for --assignee to clear the assignee.`,
		Example: `  # Update the title
  bb issue edit 123 --title "New title"

  # Update multiple fields
  bb issue edit 123 -t "New title" -b "New description" -p critical

  # Change the kind and priority
  bb issue edit 123 --kind enhancement --priority minor

  # Clear the assignee
  bb issue edit 123 --assignee ""

  # Assign to a user
  bb issue edit 123 -a username

  # Edit in a specific repository
  bb issue edit 123 -t "Fix" --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse issue ID
			issueID, err := parseIssueID(args)
			if err != nil {
				return err
			}
			opts.issueID = issueID

			// Track which flags were explicitly set
			opts.titleSet = cmd.Flags().Changed("title")
			opts.bodySet = cmd.Flags().Changed("body")
			opts.kindSet = cmd.Flags().Changed("kind")
			opts.prioritySet = cmd.Flags().Changed("priority")
			opts.assigneeSet = cmd.Flags().Changed("assignee")

			return runEdit(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.title, "title", "t", "", "New title")
	cmd.Flags().StringVarP(&opts.body, "body", "b", "", "New body/description")
	cmd.Flags().StringVarP(&opts.kind, "kind", "k", "", "New kind (bug, enhancement, proposal, task)")
	cmd.Flags().StringVarP(&opts.priority, "priority", "p", "", "New priority (trivial, minor, major, critical, blocker)")
	cmd.Flags().StringVarP(&opts.assignee, "assignee", "a", "", "New assignee username (use \"\" to clear)")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runEdit(opts *editOptions) error {
	// Check if any fields were provided
	if !opts.titleSet && !opts.bodySet && !opts.kindSet && !opts.prioritySet && !opts.assigneeSet {
		return fmt.Errorf("at least one field must be specified to update")
	}

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

	// Build update options - only include fields that were explicitly set
	updateOpts := &api.IssueUpdateOptions{}

	if opts.titleSet {
		updateOpts.Title = &opts.title
	}

	if opts.bodySet {
		updateOpts.Content = &api.Content{Raw: opts.body}
	}

	if opts.kindSet {
		// Validate kind
		validKinds := map[string]bool{"bug": true, "enhancement": true, "proposal": true, "task": true}
		if !validKinds[opts.kind] {
			return fmt.Errorf("invalid kind %q: must be one of bug, enhancement, proposal, task", opts.kind)
		}
		updateOpts.Kind = &opts.kind
	}

	if opts.prioritySet {
		// Validate priority
		validPriorities := map[string]bool{"trivial": true, "minor": true, "major": true, "critical": true, "blocker": true}
		if !validPriorities[opts.priority] {
			return fmt.Errorf("invalid priority %q: must be one of trivial, minor, major, critical, blocker", opts.priority)
		}
		updateOpts.Priority = &opts.priority
	}

	if opts.assigneeSet {
		if opts.assignee == "" {
			// Clear assignee - set to empty user
			updateOpts.Assignee = &api.User{}
		} else {
			// Resolve assignee username to UUID
			uuid, err := resolveUserUUID(ctx, client, workspace, opts.assignee)
			if err != nil {
				return fmt.Errorf("could not resolve assignee %q: %w", opts.assignee, err)
			}
			updateOpts.Assignee = &api.User{UUID: uuid}
		}
	}

	opts.streams.Info("Updating issue #%d in %s/%s...", opts.issueID, workspace, repoSlug)

	// Update the issue
	issue, err := client.UpdateIssue(ctx, workspace, repoSlug, opts.issueID, updateOpts)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	// Print success message
	opts.streams.Success("Updated issue #%d: %s", issue.ID, issue.Title)
	fmt.Fprintln(opts.streams.Out)
	if issue.Links != nil && issue.Links.HTML != nil {
		fmt.Fprintln(opts.streams.Out, issue.Links.HTML.Href)
	}

	return nil
}
