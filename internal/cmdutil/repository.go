package cmdutil

import (
	"fmt"
	"strings"

	"github.com/rbansal42/bitbucket-cli/internal/git"
)

// ParseRepository parses a repository string in WORKSPACE/REPO format,
// or detects the repository from the current git remote if not specified.
func ParseRepository(repoFlag string) (workspace, repoSlug string, err error) {
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid repository format: %s (expected workspace/repo)", repoFlag)
		}
		// Validate both parts are non-empty
		if parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid repository format: %s (workspace and repo cannot be empty)", repoFlag)
		}
		return parts[0], parts[1], nil
	}

	// Detect from git
	remote, err := git.GetDefaultRemote()
	if err != nil {
		return "", "", fmt.Errorf("could not detect repository: %w\nUse --repo WORKSPACE/REPO to specify", err)
	}

	return remote.Workspace, remote.RepoSlug, nil
}

// ParseWorkspace validates a workspace string.
// Returns the trimmed workspace or an error if empty.
func ParseWorkspace(workspace string) (string, error) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return "", fmt.Errorf("workspace is required. Use --workspace/-w to specify")
	}
	return workspace, nil
}
