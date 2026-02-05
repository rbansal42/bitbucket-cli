package issue

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
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

// parseIssueID parses an issue ID from args or returns an error
func parseIssueID(args []string) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("issue ID is required")
	}

	issueID, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, fmt.Errorf("invalid issue ID: %s", args[0])
	}

	// Validate positive issue ID
	if issueID <= 0 {
		return 0, fmt.Errorf("invalid issue ID: must be a positive integer")
	}

	return issueID, nil
}

// formatIssueState formats issue state with color
func formatIssueState(streams *iostreams.IOStreams, state string) string {
	if !streams.ColorEnabled() {
		return state
	}

	switch strings.ToLower(state) {
	case "new":
		return iostreams.Cyan + state + iostreams.Reset
	case "open":
		return iostreams.Green + state + iostreams.Reset
	case "resolved", "closed":
		return iostreams.Magenta + state + iostreams.Reset
	case "on hold":
		return iostreams.Yellow + state + iostreams.Reset
	case "invalid", "duplicate", "wontfix":
		return iostreams.Red + state + iostreams.Reset
	default:
		return state
	}
}

// formatIssuePriority formats issue priority with color
func formatIssuePriority(streams *iostreams.IOStreams, priority string) string {
	if !streams.ColorEnabled() {
		return priority
	}

	switch strings.ToLower(priority) {
	case "blocker", "critical":
		return iostreams.BoldRed + priority + iostreams.Reset
	case "major":
		return iostreams.Red + priority + iostreams.Reset
	case "minor":
		return iostreams.Yellow + priority + iostreams.Reset
	case "trivial":
		return iostreams.Green + priority + iostreams.Reset
	default:
		return priority
	}
}

// formatIssueKind formats issue kind with color
func formatIssueKind(streams *iostreams.IOStreams, kind string) string {
	if !streams.ColorEnabled() {
		return kind
	}

	switch strings.ToLower(kind) {
	case "bug":
		return iostreams.Red + kind + iostreams.Reset
	case "enhancement":
		return iostreams.Blue + kind + iostreams.Reset
	case "proposal":
		return iostreams.Cyan + kind + iostreams.Reset
	case "task":
		return iostreams.Green + kind + iostreams.Reset
	default:
		return kind
	}
}

// timeAgo returns a human-readable relative time string
func timeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 30*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 365*24*time.Hour:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(duration.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// truncateString truncates a string to maxLen characters with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// getUserDisplayName returns the best available display name for a user
func getUserDisplayName(user *api.User) string {
	if user == nil {
		return "-"
	}
	if user.DisplayName != "" {
		return user.DisplayName
	}
	if user.Username != "" {
		return user.Username
	}
	return "unknown"
}

// confirmPrompt prompts the user with a yes/no question and returns true if they confirm
func confirmPrompt(reader io.Reader) bool {
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return input == "y" || input == "yes"
	}
	return false
}

// promptForTitle prompts the user to enter a title
func promptForTitle(streams *iostreams.IOStreams) (string, error) {
	fmt.Fprint(streams.Out, "Title: ")

	reader := bufio.NewReader(os.Stdin)
	title, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(title), nil
}

// resolveUserUUID resolves a username to a UUID
func resolveUserUUID(ctx context.Context, client *api.Client, workspace, username string) (string, error) {
	// First try to get user directly by username
	path := fmt.Sprintf("/users/%s", username)
	resp, err := client.Get(ctx, path, nil)
	if err == nil {
		var user struct {
			UUID string `json:"uuid"`
		}
		if parseErr := json.Unmarshal(resp.Body, &user); parseErr == nil && user.UUID != "" {
			return user.UUID, nil
		}
	}

	// Fallback: try workspace members
	membersPath := fmt.Sprintf("/workspaces/%s/members", workspace)
	resp, err = client.Get(ctx, membersPath, nil)
	if err != nil {
		return "", fmt.Errorf("could not resolve user %q", username)
	}

	var members struct {
		Values []struct {
			User struct {
				UUID        string `json:"uuid"`
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
			} `json:"user"`
		} `json:"values"`
	}
	if err := json.Unmarshal(resp.Body, &members); err != nil {
		return "", fmt.Errorf("could not parse workspace members: %w", err)
	}

	for _, m := range members.Values {
		if m.User.Username == username || m.User.DisplayName == username {
			return m.User.UUID, nil
		}
	}

	return "", fmt.Errorf("user %q not found in workspace %q", username, workspace)
}
