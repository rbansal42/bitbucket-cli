package repo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/git"
)

// getAPIClient creates an authenticated API client
func getAPIClient() (*api.Client, error) {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load hosts config: %w", err)
	}

	user := hosts.GetActiveUser(config.DefaultHost)
	if user == "" {
		return nil, fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
	}

	tokenData, _, err := config.GetTokenFromEnvOrKeyring(config.DefaultHost, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Try to parse as JSON (OAuth token) or use as plain token
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	token := tokenData
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		token = tokenResp.AccessToken
	}

	return api.NewClient(api.WithToken(token)), nil
}

// parseRepository parses a repository string or detects from git remote
func parseRepository(repoFlag string) (workspace, repoSlug string, err error) {
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


