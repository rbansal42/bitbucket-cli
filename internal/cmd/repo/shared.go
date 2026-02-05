package repo

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
)

// getCloneURL returns the appropriate clone URL based on protocol preference
func getCloneURL(links api.RepositoryLinks, protocol string) string {
	for _, clone := range links.Clone {
		if clone.Name == protocol {
			return clone.Href
		}
	}

	// Fallback: return any available URL
	if len(links.Clone) > 0 {
		return links.Clone[0].Href
	}

	return ""
}

// getPreferredProtocol returns the user's preferred git protocol
func getPreferredProtocol() string {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "https" // default to https
	}

	if cfg.GitProtocol != "" {
		return cfg.GitProtocol
	}

	return "https"
}

// parseRepoArg parses a repository argument in workspace/repo format
// This is used when a repository is provided as a command argument (not a flag)
func parseRepoArg(arg string) (workspace, repoSlug string, err error) {
	if arg == "" {
		return "", "", fmt.Errorf("repository argument is required (workspace/repo)")
	}

	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected workspace/repo)", arg)
	}

	// Validate both parts are non-empty
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format: %s (workspace and repo cannot be empty)", arg)
	}

	return parts[0], parts[1], nil
}



// confirmDeletion prompts the user to confirm deletion by typing the repository name
func confirmDeletion(repoName string, reader io.Reader) bool {
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		return input == repoName
	}
	return false
}

// printDeleteWarning prints a warning message about repository deletion
func printDeleteWarning(w io.Writer) {
	fmt.Fprintln(w, "! Deleting a repository cannot be undone.")
}


