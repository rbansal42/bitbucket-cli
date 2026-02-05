package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type statusOptions struct {
	streams  *iostreams.IOStreams
	hostname string
}

// NewCmdStatus creates the status command
func NewCmdStatus(streams *iostreams.IOStreams) *cobra.Command {
	opts := &statusOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "View authentication status",
		Long: `View authentication status for Bitbucket.

This command displays information about your current authentication state,
including the logged-in user and token status.`,
		Example: `  # Check authentication status
  $ bb auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(opts)
		},
	}

	cmd.Flags().StringVar(&opts.hostname, "hostname", config.DefaultHost, "Bitbucket hostname")

	return cmd
}

func runStatus(opts *statusOptions) error {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return fmt.Errorf("failed to load hosts config: %w", err)
	}

	user := hosts.GetActiveUser(opts.hostname)
	if user == "" {
		opts.streams.Info("%s", opts.hostname)
		opts.streams.Error("Not logged in to %s", opts.hostname)
		opts.streams.Info("  Run 'bb auth login' to authenticate")
		return nil
	}

	// Get token
	tokenData, source, err := config.GetTokenFromEnvOrKeyring(opts.hostname, user)
	if err != nil {
		opts.streams.Info("%s", opts.hostname)
		opts.streams.Error("Token not found for %s", user)
		return nil
	}

	// Create API client based on token type
	var client *api.Client
	var displayToken string

	if strings.HasPrefix(tokenData, "basic:") {
		// Basic Auth credentials (email:api_token)
		credentials := strings.TrimPrefix(tokenData, "basic:")
		parts := strings.SplitN(credentials, ":", 2)
		if len(parts) != 2 {
			opts.streams.Info("%s", opts.hostname)
			opts.streams.Error("Invalid stored credentials format for %s", user)
			return nil
		}
		client = api.NewClient(api.WithBasicAuth(parts[0], parts[1]))
		displayToken = parts[1] // Show API token portion
	} else {
		// Try to parse as JSON (OAuth token) or use as plain token
		var tokenResp oauthTokenResponse
		if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
			displayToken = tokenResp.AccessToken
		} else {
			displayToken = tokenData
		}
		client = api.NewClient(api.WithToken(displayToken))
	}

	// Validate token by making an API request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiUser, err := client.GetCurrentUser(ctx)
	if err != nil {
		opts.streams.Info("%s", opts.hostname)
		opts.streams.Error("Token is invalid or expired for %s", user)
		opts.streams.Info("  Run 'bb auth login' to re-authenticate")
		return nil
	}

	// Print status
	opts.streams.Info("%s", opts.hostname)
	opts.streams.Success("Logged in to %s account %s (%s)", opts.hostname, apiUser.Username, source)
	opts.streams.Info("  - Active account: true")
	opts.streams.Info("  - Git operations protocol: %s", hosts.GetGitProtocol(opts.hostname))

	// Mask token for display
	maskedToken := maskToken(displayToken)
	opts.streams.Info("  - Token: %s", maskedToken)

	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
