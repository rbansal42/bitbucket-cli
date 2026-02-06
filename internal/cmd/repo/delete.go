package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type deleteOptions struct {
	streams   *iostreams.IOStreams
	repoArg   string
	yes       bool
	workspace string
	repoSlug  string
}

// NewCmdDelete creates the delete command
func NewCmdDelete(streams *iostreams.IOStreams) *cobra.Command {
	opts := &deleteOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "delete <workspace/repo>",
		Short: "Delete a repository",
		Long: `Delete a repository permanently.

WARNING: This action cannot be undone. The repository and all its data
(commits, branches, pull requests, issues, etc.) will be permanently deleted.

You will be prompted to type the repository name to confirm deletion,
unless the --yes flag is provided.`,
		Example: `  # Delete a repository (will prompt for confirmation)
  bb repo delete myworkspace/myrepo

  # Delete without confirmation prompt
  bb repo delete myworkspace/myrepo --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.repoArg = args[0]
			return runDelete(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(opts *deleteOptions) error {
	// Parse the repository argument
	var err error
	opts.workspace, opts.repoSlug, err = cmdutil.ParseRepository(opts.repoArg)
	if err != nil {
		return err
	}

	// If not auto-confirmed, show warning and prompt
	if !opts.yes {
		// Require TTY for interactive confirmation
		if !opts.streams.IsStdinTTY() {
			return fmt.Errorf("cannot confirm deletion: stdin is not a terminal\nUse --yes flag to skip confirmation in non-interactive mode")
		}

		printDeleteWarning(opts.streams.ErrOut)

		fmt.Fprintf(opts.streams.Out, "Type '%s' to confirm deletion: ", opts.repoSlug)

		if !confirmDeletion(opts.repoSlug, opts.streams.In) {
			return fmt.Errorf("deletion cancelled: repository name did not match")
		}
	}

	// Get authenticated client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Delete the repository
	if err := client.DeleteRepository(ctx, opts.workspace, opts.repoSlug); err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	opts.streams.Success("Deleted repository %s/%s", opts.workspace, opts.repoSlug)
	return nil
}
