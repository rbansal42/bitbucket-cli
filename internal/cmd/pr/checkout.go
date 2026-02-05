package pr

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
)

type checkoutOptions struct {
	streams  *iostreams.IOStreams
	prNumber int
	repo     string
	force    bool
}

// NewCmdCheckout creates the checkout command
func NewCmdCheckout(streams *iostreams.IOStreams) *cobra.Command {
	opts := &checkoutOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "checkout <number>",
		Short: "Check out a pull request locally",
		Long: `Check out a pull request branch locally.

This command fetches the pull request's source branch from the remote
and creates a local branch to track it. If the local branch already exists,
use --force to overwrite it.`,
		Example: `  # Check out pull request #123
  bb pr checkout 123

  # Force overwrite existing local branch
  bb pr checkout 123 --force

  # Check out from a specific repository
  bb pr checkout 123 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			opts.prNumber, err = parsePRNumber(args)
			if err != nil {
				return err
			}

			// Get repo from flag or inherit from parent
			if opts.repo == "" {
				opts.repo, _ = cmd.Flags().GetString("repo")
			}

			return runCheckout(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Overwrite existing local branch")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runCheckout(opts *checkoutOptions) error {
	// Resolve repository
	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	opts.streams.Info("Fetching pull request #%d...", opts.prNumber)

	// Get authenticated API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Get PR details
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := getPullRequest(ctx, client, workspace, repoSlug, opts.prNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	sourceBranch := pr.Source.Branch.Name
	if sourceBranch == "" {
		return fmt.Errorf("pull request has no source branch")
	}

	// Check if local branch exists
	localBranchExists := branchExists(sourceBranch)

	if localBranchExists && !opts.force {
		return fmt.Errorf("branch '%s' already exists locally. Use --force to overwrite", sourceBranch)
	}

	// Determine remote name (default to origin)
	remote, err := git.GetDefaultRemote()
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	// Fetch the branch
	if localBranchExists && opts.force {
		// Delete the existing branch first (if not currently checked out)
		currentBranch, _ := git.GetCurrentBranch()
		if currentBranch == sourceBranch {
			return fmt.Errorf("cannot overwrite branch '%s' while it is checked out", sourceBranch)
		}
		if err := deleteBranch(sourceBranch, true); err != nil {
			return fmt.Errorf("failed to delete existing branch: %w", err)
		}
	}

	// Fetch and create tracking branch
	refspec := fmt.Sprintf("%s:%s", sourceBranch, sourceBranch)
	if err := git.Fetch(remote.Name, refspec); err != nil {
		return fmt.Errorf("failed to fetch branch: %w", err)
	}

	// Set up tracking
	if err := setUpstreamTracking(sourceBranch, remote.Name); err != nil {
		// Non-fatal, just warn
		opts.streams.Warning("Could not set upstream tracking: %v", err)
	}

	// Checkout the branch
	if err := git.Checkout(sourceBranch); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	opts.streams.Success("Switched to branch '%s'", sourceBranch)
	return nil
}

// branchExists checks if a local branch exists
func branchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	return cmd.Run() == nil
}

// deleteBranch deletes a local branch
func deleteBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, branch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

// setUpstreamTracking sets the upstream tracking branch
func setUpstreamTracking(branch, remote string) error {
	cmd := exec.Command("git", "branch", "--set-upstream-to="+remote+"/"+branch, branch)
	return cmd.Run()
}
