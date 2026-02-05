package repo

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/config"
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


