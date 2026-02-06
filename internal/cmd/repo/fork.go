package repo

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/git"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type forkOptions struct {
	streams    *iostreams.IOStreams
	sourceRepo string
	workspace  string
	name       string
	clone      bool
	remoteName string
}

// NewCmdFork creates the repo fork command
func NewCmdFork(streams *iostreams.IOStreams) *cobra.Command {
	opts := &forkOptions{
		streams:    streams,
		remoteName: "fork",
	}

	cmd := &cobra.Command{
		Use:   "fork [<workspace/repo>]",
		Short: "Fork a repository",
		Long: `Create a fork of a repository.

If no repository is specified, the current repository is forked (detected
from git remote).

By default, the fork is created in your personal workspace with the same
name as the original repository.

If you're in an existing clone of the repository, the fork will be added
as a new remote (default name: "fork").`,
		Example: `  # Fork the current repository
  bb repo fork

  # Fork a specific repository
  bb repo fork myworkspace/repo

  # Fork to a different workspace
  bb repo fork myworkspace/repo --workspace otherworkspace

  # Fork with a different name
  bb repo fork myworkspace/repo --name my-fork

  # Fork and clone the result
  bb repo fork myworkspace/repo --clone

  # Fork and add as remote with custom name
  bb repo fork --remote-name upstream`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.sourceRepo = args[0]
			}

			return runFork(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.workspace, "workspace", "w", "", "Destination workspace (default: your personal workspace)")
	cmd.Flags().StringVar(&opts.name, "name", "", "Name for the forked repository (default: same as original)")
	cmd.Flags().BoolVarP(&opts.clone, "clone", "c", false, "Clone the fork after creation")
	cmd.Flags().StringVar(&opts.remoteName, "remote-name", "fork", "Name for the new remote when in an existing clone")

	return cmd
}

func runFork(opts *forkOptions) error {
	// Get authenticated client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Parse source repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.sourceRepo)
	if err != nil {
		return err
	}

	// Check if we're in an existing git repository
	inExistingRepo := git.IsGitRepository()

	// Determine destination workspace
	destWorkspace := opts.workspace
	if destWorkspace == "" {
		// Try to get default workspace from config
		defaultWs, err := config.GetDefaultWorkspace()
		if err == nil && defaultWs != "" {
			destWorkspace = defaultWs
		}
	}
	if destWorkspace == "" {
		// Try to get current user's workspace
		user, err := client.GetCurrentUser(ctx)
		if err != nil {
			return fmt.Errorf("could not determine destination workspace: %w\nUse --workspace to specify or run 'bb workspace set-default'", err)
		}
		destWorkspace = user.Username
	}

	// Determine fork name
	forkName := opts.name
	if forkName == "" {
		forkName = repoSlug
	}

	opts.streams.Info("Forking %s/%s to %s/%s...", workspace, repoSlug, destWorkspace, forkName)

	// Create the fork
	fork, err := client.ForkRepository(ctx, workspace, repoSlug, destWorkspace, forkName)
	if err != nil {
		return fmt.Errorf("failed to fork repository: %w", err)
	}

	// Success message
	opts.streams.Success("Forked %s/%s to %s", workspace, repoSlug, fork.FullName)
	fmt.Fprintln(opts.streams.Out)
	fmt.Fprintf(opts.streams.Out, "%s\n", fork.Links.HTML.Href)

	// Handle post-fork actions
	if opts.clone {
		// Clone the fork
		fmt.Fprintln(opts.streams.Out)
		opts.streams.Info("Cloning fork...")

		protocol := getPreferredProtocol()
		cloneURL := getCloneURL(fork.Links, protocol)

		if err := git.Clone(cloneURL, forkName); err != nil {
			return fmt.Errorf("failed to clone fork: %w", err)
		}

		opts.streams.Success("Cloned to %s/", forkName)

		// Optionally add the original repo as upstream remote
		if err := addUpstreamRemote(forkName, workspace, repoSlug); err != nil {
			opts.streams.Warning("Could not add upstream remote: %v", err)
		} else {
			opts.streams.Success("Added upstream remote for %s/%s", workspace, repoSlug)
		}

	} else if inExistingRepo && opts.remoteName != "" {
		// Add the fork as a new remote in the existing repo
		protocol := getPreferredProtocol()
		cloneURL := getCloneURL(fork.Links, protocol)

		fmt.Fprintln(opts.streams.Out)
		opts.streams.Info("Adding fork as remote '%s'...", opts.remoteName)

		if err := addRemote(opts.remoteName, cloneURL); err != nil {
			// Remote addition failure shouldn't fail the entire fork operation
			opts.streams.Warning("Could not add remote '%s': %v", opts.remoteName, err)
			opts.streams.Info("You can add the remote manually with: git remote add %s %s", opts.remoteName, cloneURL)
		} else {
			opts.streams.Success("Added remote '%s' pointing to %s", opts.remoteName, fork.FullName)
		}
	}

	return nil
}

// addUpstreamRemote adds the original repository as an "upstream" remote
func addUpstreamRemote(repoDir, workspace, repoSlug string) error {
	protocol := getPreferredProtocol()
	var upstreamURL string
	if protocol == "ssh" {
		upstreamURL = fmt.Sprintf("git@bitbucket.org:%s/%s.git", workspace, repoSlug)
	} else {
		upstreamURL = fmt.Sprintf("https://bitbucket.org/%s/%s.git", workspace, repoSlug)
	}

	cmd := exec.Command("git", "-C", repoDir, "remote", "add", "upstream", upstreamURL)
	return cmd.Run()
}

// addRemote adds a new remote to the current repository
func addRemote(name, url string) error {
	cmd := exec.Command("git", "remote", "add", name, url)
	return cmd.Run()
}
