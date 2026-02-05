package auth

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type tokenOptions struct {
	streams  *iostreams.IOStreams
	hostname string
}

// NewCmdToken creates the token command
func NewCmdToken(streams *iostreams.IOStreams) *cobra.Command {
	opts := &tokenOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the authentication token",
		Long: `Print the authentication token for Bitbucket.

This command outputs the access token that bb uses for API authentication.
This is useful for using the token with other tools or scripts.`,
		Example: `  # Print the token
  $ bb auth token

  # Use the token with curl
  $ curl -H "Authorization: Bearer $(bb auth token)" https://api.bitbucket.org/2.0/user`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runToken(opts)
		},
	}

	cmd.Flags().StringVar(&opts.hostname, "hostname", config.DefaultHost, "Bitbucket hostname")

	return cmd
}

func runToken(opts *tokenOptions) error {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return fmt.Errorf("failed to load hosts config: %w", err)
	}

	user := hosts.GetActiveUser(opts.hostname)
	if user == "" {
		return fmt.Errorf("not logged in to %s. Run 'bb auth login' to authenticate", opts.hostname)
	}

	// Get token
	tokenData, _, err := config.GetTokenFromEnvOrKeyring(opts.hostname, user)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Try to parse as JSON (OAuth token) or use as plain token
	var tokenResp oauthTokenResponse
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		fmt.Println(tokenResp.AccessToken)
	} else {
		fmt.Println(tokenData)
	}

	return nil
}
