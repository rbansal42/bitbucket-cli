package issue

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type reopenOptions struct {
	streams *iostreams.IOStreams
	repo    string
}

// NewCmdReopen creates the reopen command
func NewCmdReopen(streams *iostreams.IOStreams) *cobra.Command {
	opts := &reopenOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "reopen <issue-id>",
		Short: "Reopen a closed issue",
		Long: `Reopen a previously closed issue by setting its state to open.

This command is useful when an issue was closed prematurely or when 
additional work is needed.`,
		Example: `  # Reopen issue #42
  bb issue reopen 42

  # Reopen an issue in a specific repository
  bb issue reopen 42 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReopen(opts, args)
		},
	}

	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runReopen(opts *reopenOptions, args []string) error {
	issueID, err := parseIssueID(args)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Update issue state to open
	state := "open"
	updateOpts := &api.IssueUpdateOptions{
		State: &state,
	}

	_, err = client.UpdateIssue(ctx, workspace, repoSlug, issueID, updateOpts)
	if err != nil {
		return fmt.Errorf("failed to reopen issue: %w", err)
	}

	opts.streams.Success("Reopened issue #%d", issueID)
	return nil
}
