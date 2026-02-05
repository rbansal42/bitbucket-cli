package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Remote represents a git remote
type Remote struct {
	Name      string
	FetchURL  string
	PushURL   string
	Workspace string
	RepoSlug  string
}

// BitbucketRemote extracts workspace and repo from a Bitbucket remote URL
type BitbucketRemote struct {
	Workspace string
	RepoSlug  string
}

var (
	// SSH URL pattern: git@bitbucket.org:workspace/repo.git
	sshPattern = regexp.MustCompile(`^git@bitbucket\.org:([^/]+)/([^/]+?)(?:\.git)?$`)

	// HTTPS URL pattern: https://bitbucket.org/workspace/repo.git
	httpsPattern = regexp.MustCompile(`^https://(?:[^@]+@)?bitbucket\.org/([^/]+)/([^/]+?)(?:\.git)?$`)
)

// ParseBitbucketURL parses a Bitbucket remote URL and extracts workspace and repo
func ParseBitbucketURL(url string) (*BitbucketRemote, error) {
	url = strings.TrimSpace(url)

	// Try SSH pattern
	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		return &BitbucketRemote{
			Workspace: matches[1],
			RepoSlug:  matches[2],
		}, nil
	}

	// Try HTTPS pattern
	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 3 {
		return &BitbucketRemote{
			Workspace: matches[1],
			RepoSlug:  matches[2],
		}, nil
	}

	return nil, fmt.Errorf("not a valid Bitbucket URL: %s", url)
}

// IsBitbucketURL checks if a URL points to Bitbucket
func IsBitbucketURL(url string) bool {
	return strings.Contains(url, "bitbucket.org")
}

// GetRemotes returns all git remotes for the current repository
func GetRemotes() ([]Remote, error) {
	cmd := exec.Command("git", "remote", "-v")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get git remotes: %w", err)
	}

	return parseRemotes(stdout.String())
}

func parseRemotes(output string) ([]Remote, error) {
	remotes := make(map[string]*Remote)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		url := parts[1]
		urlType := strings.Trim(parts[2], "()")

		if _, ok := remotes[name]; !ok {
			remotes[name] = &Remote{Name: name}
		}

		switch urlType {
		case "fetch":
			remotes[name].FetchURL = url
		case "push":
			remotes[name].PushURL = url
		}

		// Extract workspace and repo for Bitbucket URLs
		if IsBitbucketURL(url) {
			if bbRemote, err := ParseBitbucketURL(url); err == nil {
				remotes[name].Workspace = bbRemote.Workspace
				remotes[name].RepoSlug = bbRemote.RepoSlug
			}
		}
	}

	result := make([]Remote, 0, len(remotes))
	for _, r := range remotes {
		result = append(result, *r)
	}

	return result, nil
}

// GetBitbucketRemotes returns only Bitbucket remotes
func GetBitbucketRemotes() ([]Remote, error) {
	allRemotes, err := GetRemotes()
	if err != nil {
		return nil, err
	}

	var bbRemotes []Remote
	for _, r := range allRemotes {
		if r.Workspace != "" && r.RepoSlug != "" {
			bbRemotes = append(bbRemotes, r)
		}
	}

	return bbRemotes, nil
}

// GetDefaultRemote returns the default Bitbucket remote (prefers "origin")
func GetDefaultRemote() (*Remote, error) {
	remotes, err := GetBitbucketRemotes()
	if err != nil {
		return nil, err
	}

	if len(remotes) == 0 {
		return nil, fmt.Errorf("no Bitbucket remotes found")
	}

	// Prefer "origin"
	for _, r := range remotes {
		if r.Name == "origin" {
			return &r, nil
		}
	}

	// Return first remote
	return &remotes[0], nil
}

// IsGitRepository checks if the current directory is a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetCurrentBranch returns the current git branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Checkout checks out a branch
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}
	return nil
}

// Fetch fetches from a remote
func Fetch(remote string, refspec string) error {
	args := []string{"fetch", remote}
	if refspec != "" {
		args = append(args, refspec)
	}

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch from %s: %w", remote, err)
	}
	return nil
}

// Clone clones a repository
func Clone(url string, dest string) error {
	args := []string{"clone", url}
	if dest != "" {
		args = append(args, dest)
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	return nil
}
