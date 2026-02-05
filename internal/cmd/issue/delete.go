package issue

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

type deleteOptions struct {
	streams *iostreams.IOStreams
	repo    string
	yes     bool
}

// NewCmdDelete creates the delete command
func NewCmdDelete(streams *iostreams.IOStreams) *cobra.Command {
	opts := &deleteOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "delete <issue-id>",
		Short: "Delete an issue",
		Long: `Delete an issue permanently.

WARNING: This action cannot be undone. The issue and all its comments
will be permanently deleted.

You will be prompted to confirm deletion unless the --yes flag is provided.`,
		Example: `  # Delete issue #42 (will prompt for confirmation)
  bb issue delete 42

  # Delete without confirmation prompt
  bb issue delete 42 --yes

  # Delete an issue in a specific repository
  bb issue delete 42 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(opts, args)
		},
	}

	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runDelete(opts *deleteOptions, args []string) error {
	issueID, err := parseIssueID(args)
	if err != nil {
		return err
	}

	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	// If not auto-confirmed, show warning and prompt
	if !opts.yes {
		// Require TTY for interactive confirmation
		if !opts.streams.IsStdinTTY() {
			return fmt.Errorf("cannot confirm deletion: stdin is not a terminal\nUse --yes flag to skip confirmation in non-interactive mode")
		}

		fmt.Fprintf(opts.streams.Out, "Are you sure you want to delete issue #%d? [y/N] ", issueID)

		if !confirmPrompt(opts.streams.In) {
			return fmt.Errorf("deletion cancelled")
		}
	}

	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.DeleteIssue(ctx, workspace, repoSlug, issueID)
	if err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}

	opts.streams.Success("Deleted issue #%d", issueID)
	return nil
}
