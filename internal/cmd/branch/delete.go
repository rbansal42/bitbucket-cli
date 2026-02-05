package branch

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// DeleteOptions holds the options for the delete command
type DeleteOptions struct {
	BranchName string
	Repo       string
	Force      bool
	Streams    *iostreams.IOStreams
}

// NewCmdDelete creates the branch delete command
func NewCmdDelete(streams *iostreams.IOStreams) *cobra.Command {
	opts := &DeleteOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "delete <branch-name>",
		Short: "Delete a branch",
		Long: `Delete a branch from a Bitbucket repository.

By default, you will be prompted to confirm the deletion.
Use --force to skip the confirmation prompt.

By default, this command detects the repository from your git remote.`,
		Example: `  # Delete a branch (will prompt for confirmation)
  bb branch delete feature-branch

  # Delete without confirmation
  bb branch delete feature-branch --force

  # Delete a branch in a specific repository
  bb branch delete feature-branch --repo myworkspace/myrepo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BranchName = args[0]
			return runDelete(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format (detects from git remote if not specified)")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(ctx context.Context, opts *DeleteOptions) error {
	// Parse repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.Repo)
	if err != nil {
		return err
	}

	// If not forced, prompt for confirmation
	if !opts.Force {
		// Require TTY for interactive confirmation
		if !opts.Streams.IsStdinTTY() {
			return fmt.Errorf("cannot confirm deletion in non-interactive mode\nUse --force flag to skip confirmation")
		}

		fmt.Fprintf(opts.Streams.Out, "Delete branch %s from %s/%s? [y/N]: ", opts.BranchName, workspace, repoSlug)

		reader := bufio.NewReader(opts.Streams.In)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("deletion cancelled")
		}
	}

	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Delete the branch
	if err := client.DeleteBranch(ctx, workspace, repoSlug, opts.BranchName); err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	opts.Streams.Success("Deleted branch %s from %s/%s", opts.BranchName, workspace, repoSlug)
	return nil
}
