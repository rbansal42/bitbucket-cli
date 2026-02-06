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

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

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
