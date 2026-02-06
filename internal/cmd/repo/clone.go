package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type cloneOptions struct {
	streams   *iostreams.IOStreams
	repoArg   string
	directory string
	depth     int
	branch    string
}

// NewCmdClone creates the repo clone command
func NewCmdClone(streams *iostreams.IOStreams) *cobra.Command {
	opts := &cloneOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "clone <workspace/repo> [<directory>]",
		Short: "Clone a repository",
		Long: `Clone a Bitbucket repository to your local machine.

You can specify a repository using the workspace/repo format, or provide
a full Bitbucket URL (SSH or HTTPS).

The clone URL protocol (SSH or HTTPS) is determined by the git_protocol
setting in your configuration. Use 'bb config set git_protocol <ssh|https>'
to change this preference.`,
		Example: `  # Clone a repository
  bb repo clone myworkspace/myrepo

  # Clone to a specific directory
  bb repo clone myworkspace/myrepo my-local-folder

  # Clone a specific branch
  bb repo clone myworkspace/myrepo -b develop

  # Shallow clone (only latest commit)
  bb repo clone myworkspace/myrepo --depth 1

  # Clone using a full URL
  bb repo clone https://bitbucket.org/myworkspace/myrepo.git
  bb repo clone git@bitbucket.org:myworkspace/myrepo.git`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.repoArg = args[0]
			if len(args) > 1 {
				opts.directory = args[1]
			}

			return runClone(opts)
		},
	}

	cmd.Flags().IntVar(&opts.depth, "depth", 0, "Create a shallow clone with a limited number of commits")
	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "Clone a specific branch")

	return cmd
}

func runClone(opts *cloneOptions) error {
	var cloneURL string
	var destDir string

	// Check if the argument is already a URL
	if isURL(opts.repoArg) {
		cloneURL = opts.repoArg
		// Extract repo slug from URL for default directory name
		destDir = extractRepoNameFromURL(opts.repoArg)
		if destDir == "" {
			return fmt.Errorf("could not determine repository name from URL: %s", opts.repoArg)
		}
	} else {
		// Parse workspace/repo format
		workspace, repoSlug, err := cmdutil.ParseRepository(opts.repoArg)
		if err != nil {
			return err
		}

		// Get authenticated client
		client, err := cmdutil.GetAPIClient()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Fetch repository details to get clone URLs
		repo, err := client.GetRepository(ctx, workspace, repoSlug)
		if err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}

		// Get preferred protocol and clone URL
		protocol := getPreferredProtocol()
		cloneURL = getCloneURL(repo.Links, protocol)
		if cloneURL == "" {
			return fmt.Errorf("no clone URL found for repository")
		}

		destDir = repoSlug
	}

	// Use custom directory if specified
	if opts.directory != "" {
		destDir = opts.directory
	}

	// Check if destination already exists
	if destDir != "" {
		if _, err := os.Stat(destDir); err == nil {
			return fmt.Errorf("destination path '%s' already exists", destDir)
		}
	}

	// Build git clone command
	args := []string{"clone"}

	// Add depth flag if specified
	if opts.depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.depth))
	}

	// Add branch flag if specified
	if opts.branch != "" {
		args = append(args, "--branch", opts.branch)
	}

	// Add progress flag for better UX
	args = append(args, "--progress")

	// Add clone URL
	args = append(args, cloneURL)

	// Add destination directory
	if destDir != "" {
		args = append(args, destDir)
	}

	// Execute git clone
	opts.streams.Info("Cloning into '%s'...", destDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = opts.streams.Out
	cmd.Stderr = opts.streams.ErrOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Print success message with cd hint
	fmt.Fprintln(opts.streams.Out)
	opts.streams.Success("Cloned repository to %s/", destDir)

	// Get absolute path for cd hint
	absPath, err := filepath.Abs(destDir)
	if err == nil {
		fmt.Fprintf(opts.streams.Out, "\nTo get started, run:\n  cd %s\n", absPath)
	}

	return nil
}

// isURL checks if the given string looks like a URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "git@") ||
		strings.HasPrefix(s, "ssh://")
}

// extractRepoNameFromURL extracts the repository name from a clone URL
func extractRepoNameFromURL(url string) string {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH URLs like git@bitbucket.org:workspace/repo
	if strings.HasPrefix(url, "git@") {
		// Find the part after the colon
		if colonIdx := strings.Index(url, ":"); colonIdx != -1 {
			path := url[colonIdx+1:]
			parts := strings.Split(path, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}

	// Handle HTTPS/SSH URLs
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}


