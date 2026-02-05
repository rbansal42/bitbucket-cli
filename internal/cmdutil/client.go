// Package cmdutil provides shared utilities for command implementations.
package cmdutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/config"
)

// GetAPIClient creates an authenticated API client.
// This is the canonical implementation used by all commands.
func GetAPIClient() (*api.Client, error) {
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

	// Check if this is Basic Auth credentials (prefixed with "basic:")
	if strings.HasPrefix(tokenData, "basic:") {
		credentials := strings.TrimPrefix(tokenData, "basic:")
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid stored credentials format")
		}
		return api.NewClient(api.WithBasicAuth(parts[0], parts[1])), nil
	}

	// Try to parse as JSON (OAuth token) or use as plain token (Bearer)
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	token := tokenData
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		token = tokenResp.AccessToken
	}

	return api.NewClient(api.WithToken(token)), nil
}
