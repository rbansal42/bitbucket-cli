package snippet

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
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

// parseWorkspace validates a workspace string
func parseWorkspace(workspace string) error {
	if workspace == "" {
		return fmt.Errorf("workspace is required. Use --workspace/-w to specify")
	}
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return fmt.Errorf("workspace cannot be empty")
	}
	return nil
}

// truncateString truncates a string to maxLen characters and replaces newlines
func truncateString(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
