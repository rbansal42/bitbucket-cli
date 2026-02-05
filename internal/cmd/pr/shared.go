package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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

// parsePRNumber parses a PR number from args or returns an error
func parsePRNumber(args []string) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("pull request number is required")
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, fmt.Errorf("invalid pull request number: %s", args[0])
	}

	// Validate positive PR number
	if prNum <= 0 {
		return 0, fmt.Errorf("invalid pull request number: must be a positive integer")
	}

	return prNum, nil
}

// openEditor opens the user's preferred editor for text input
func openEditor(initialContent string) (string, error) {
	editor := getEditor()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "bb-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	if initialContent != "" {
		if _, err := tmpFile.WriteString(initialContent); err != nil {
			return "", fmt.Errorf("failed to write to temp file: %w", err)
		}
	}
	tmpFile.Close()

	// Open editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read content back
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read temp file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// getEditor returns the user's preferred editor
func getEditor() string {
	// Check BB_EDITOR first
	if editor := os.Getenv("BB_EDITOR"); editor != "" {
		return editor
	}

	// Check config
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Editor != "" {
		return cfg.Editor
	}

	// Check standard environment variables
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Default to vi
	return "vi"
}

// PRUser represents a user in a pull request context
type PRUser struct {
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AccountID   string `json:"account_id"`
	Nickname    string `json:"nickname"`
	Links       struct {
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// PRParticipant represents a participant in a pull request
type PRParticipant struct {
	User     PRUser `json:"user"`
	Role     string `json:"role"`     // PARTICIPANT, REVIEWER
	Approved bool   `json:"approved"`
	State    string `json:"state"`    // approved, changes_requested, etc.
}

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Author      PRUser `json:"author"`
	Source      struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"destination"`
	Reviewers        []PRUser        `json:"reviewers"`
	Participants     []PRParticipant `json:"participants"`
	CommentCount     int             `json:"comment_count"`
	TaskCount        int             `json:"task_count"`
	CloseSourceBranch bool           `json:"close_source_branch"`
	CreatedOn        string          `json:"created_on"`
	UpdatedOn        string          `json:"updated_on"`
	Links            struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Diff struct {
			Href string `json:"href"`
		} `json:"diff"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// PRComment represents a pull request comment
type PRComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// getPullRequest fetches a pull request by number
func getPullRequest(ctx context.Context, client *api.Client, workspace, repoSlug string, prNum int) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", workspace, repoSlug, prNum)
	resp, err := client.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return api.ParseResponse[*PullRequest](resp)
}
