package repo

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
)

type syncOptions struct {
	streams   *iostreams.IOStreams
	branch    string
	force     bool
	workspace string
	repoSlug  string
}

// NewCmdSync creates the sync command
func NewCmdSync(streams *iostreams.IOStreams) *cobra.Command {
	opts := &syncOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync fork with upstream repository",
		Long: `Sync the current fork with its upstream (parent) repository.

This command fetches changes from the upstream repository and updates
the local branch. The repository must be a fork.

By default, the main branch is synced. Use --branch to specify a different branch.`,
		Example: `  # Sync the default branch with upstream
  bb repo sync

  # Sync a specific branch
  bb repo sync --branch develop

  # Force sync (reset to upstream, discarding local changes)
  bb repo sync --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "Branch to sync (default: main branch)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Force update (reset to upstream, discarding local changes)")

	return cmd
}

func runSync(opts *syncOptions) error {
	// Detect current repository from git
	remote, err := git.GetDefaultRemote()
	if err != nil {
		return fmt.Errorf("could not detect repository: %w", err)
	}
	opts.workspace = remote.Workspace
	opts.repoSlug = remote.RepoSlug

	// Get authenticated client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get repository info to check if it's a fork
	repo, err := client.GetRepository(ctx, opts.workspace, opts.repoSlug)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w", err)
	}

	// Check if repo has a parent (is a fork)
	if repo.Parent == nil {
		return fmt.Errorf("this repository is not a fork; nothing to sync with")
	}

	// Determine branch to sync
	branch := detectDefaultBranch(getMainBranchName(repo), opts.branch)

	// Build parent repository URL
	if repo.Parent.Workspace == nil {
		return fmt.Errorf("parent repository has no workspace information")
	}
	parentWorkspace := repo.Parent.Workspace.Slug
	parentSlug := repo.Parent.Slug
	parentFullName := fmt.Sprintf("%s/%s", parentWorkspace, parentSlug)

	// Setup upstream remote if needed
	upstreamRemote := getUpstreamRemoteName()
	parentURL := buildParentURL(repo.Parent)

	// Add upstream remote if it doesn't exist
	if err := ensureUpstreamRemote(upstreamRemote, parentURL); err != nil {
		return fmt.Errorf("failed to set up upstream remote: %w", err)
	}

	// Fetch from upstream
	opts.streams.Info("Fetching from upstream %s...", parentFullName)
	refspec := buildFetchRefspec(upstreamRemote, branch)
	if err := fetchUpstream(upstreamRemote, refspec); err != nil {
		return fmt.Errorf("failed to fetch from upstream: %w", err)
	}

	// Merge or reset
	if opts.force {
		// Require confirmation for force reset (destructive operation)
		if !opts.streams.IsStdinTTY() {
			return fmt.Errorf("cannot confirm force sync: stdin is not a terminal\nForce sync requires interactive confirmation as it discards local changes")
		}

		opts.streams.Warning("This will discard ALL local changes on branch '%s'", branch)
		fmt.Fprintf(opts.streams.Out, "Are you sure you want to force sync? [y/N] ")

		if !confirmForceSync(opts.streams.In) {
			return fmt.Errorf("force sync cancelled")
		}

		if err := resetToUpstream(upstreamRemote, branch); err != nil {
			return fmt.Errorf("failed to reset to upstream: %w", err)
		}
	} else {
		if err := mergeUpstream(upstreamRemote, branch); err != nil {
			return fmt.Errorf("failed to merge upstream changes: %w", err)
		}
	}

	opts.streams.Success("Synced with upstream %s", parentFullName)
	fmt.Fprintf(opts.streams.Out, "  %s is now up to date\n", branch)
	return nil
}

// detectDefaultBranch determines which branch to sync
func detectDefaultBranch(mainBranch, flagBranch string) string {
	if flagBranch != "" {
		return flagBranch
	}
	if mainBranch != "" {
		return mainBranch
	}
	return "main"
}

// getUpstreamRemoteName returns the name to use for the upstream remote
func getUpstreamRemoteName() string {
	return "upstream"
}

// buildFetchRefspec builds a refspec for fetching a branch from upstream
func buildFetchRefspec(remote, branch string) string {
	return fmt.Sprintf("refs/heads/%s:refs/remotes/%s/%s", branch, remote, branch)
}

// getMainBranchName extracts the main branch name from repository info
func getMainBranchName(repo *api.RepositoryFull) string {
	if repo.MainBranch != nil {
		return repo.MainBranch.Name
	}
	return ""
}

// buildParentURL builds the git URL for the parent repository
func buildParentURL(parent *api.ParentRepository) string {
	// Build HTTPS URL
	return fmt.Sprintf("https://bitbucket.org/%s/%s.git", parent.Workspace.Slug, parent.Slug)
}

// ensureUpstreamRemote ensures the upstream remote exists
func ensureUpstreamRemote(remoteName, url string) error {
	// Check if remote exists
	cmd := exec.Command("git", "remote", "get-url", remoteName)
	if err := cmd.Run(); err != nil {
		// Remote doesn't exist, add it
		cmd = exec.Command("git", "remote", "add", remoteName, url)
		return cmd.Run()
	}
	// Remote exists, update URL
	cmd = exec.Command("git", "remote", "set-url", remoteName, url)
	return cmd.Run()
}

// fetchUpstream fetches from the upstream remote
func fetchUpstream(remote, refspec string) error {
	cmd := exec.Command("git", "fetch", remote, refspec)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}
	return nil
}

// mergeUpstream merges changes from upstream
func mergeUpstream(remote, branch string) error {
	ref := fmt.Sprintf("%s/%s", remote, branch)
	cmd := exec.Command("git", "merge", ref, "--ff-only")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}
	return nil
}

// resetToUpstream resets the current branch to upstream
func resetToUpstream(remote, branch string) error {
	ref := fmt.Sprintf("%s/%s", remote, branch)
	cmd := exec.Command("git", "reset", "--hard", ref)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}
	return nil
}

// confirmForceSync prompts the user to confirm force sync operation
func confirmForceSync(in interface{}) bool {
	var reader *bufio.Reader

	// Handle different input types
	switch r := in.(type) {
	case *bufio.Reader:
		reader = r
	case io.Reader:
		reader = bufio.NewReader(r)
	default:
		return false
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
