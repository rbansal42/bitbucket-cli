package pr

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
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
		Use:   "reopen <number>",
		Short: "Reopen a declined pull request",
		Long: `Reopen a pull request that was previously declined.

Only declined pull requests can be reopened. Merged pull requests cannot be reopened.`,
		Example: `  # Reopen pull request #123
  bb pr reopen 123

  # Reopen a PR in a specific repository
  bb pr reopen 123 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReopen(opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runReopen(opts *reopenOptions, args []string) error {
	prNum, err := parsePRNumber(args)
	if err != nil {
		return err
	}

	workspace, repoSlug, err := cmdutil.ParseRepository(opts.repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// First, check if PR is declined
	pr, err := client.GetPullRequest(ctx, workspace, repoSlug, int64(prNum))
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	if pr.State != api.PRStateDeclined {
		return fmt.Errorf("pull request #%d is not declined (current state: %s)", prNum, pr.State)
	}

	// Reopen the PR by updating its state to OPEN
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", workspace, repoSlug, prNum)
	body := map[string]interface{}{
		"state": "OPEN",
	}
	if _, err := client.Put(ctx, path, body); err != nil {
		return fmt.Errorf("failed to reopen pull request: %w", err)
	}

	opts.streams.Success("Reopened pull request #%d", prNum)
	return nil
}
