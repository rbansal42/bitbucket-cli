package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type logoutOptions struct {
	streams  *iostreams.IOStreams
	hostname string
	user     string
}

// NewCmdLogout creates the logout command
func NewCmdLogout(streams *iostreams.IOStreams) *cobra.Command {
	opts := &logoutOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of Bitbucket",
		Long: `Remove authentication for Bitbucket.

This will remove the stored access token from your system keychain
and update your configuration file.`,
		Example: `  # Log out of the default host
  $ bb auth logout

  # Log out a specific user
  $ bb auth logout --user myusername`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(opts)
		},
	}

	cmd.Flags().StringVar(&opts.hostname, "hostname", config.DefaultHost, "Bitbucket hostname")
	cmd.Flags().StringVarP(&opts.user, "user", "u", "", "User to log out (defaults to active user)")

	return cmd
}

func runLogout(opts *logoutOptions) error {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return fmt.Errorf("failed to load hosts config: %w", err)
	}

	// Determine user to log out
	user := opts.user
	if user == "" {
		user = hosts.GetActiveUser(opts.hostname)
		if user == "" {
			return fmt.Errorf("not logged in to %s", opts.hostname)
		}
	}

	// Delete token from keyring
	if err := config.DeleteToken(opts.hostname, user); err != nil {
		opts.streams.Warning("Could not remove token from keyring: %v", err)
	}

	// Update hosts config
	hostConfig, ok := hosts[opts.hostname]
	if ok {
		delete(hostConfig.Users, user)

		// Clear active user if it was the one being logged out
		if hostConfig.User == user {
			hostConfig.User = ""

			// Set another user as active if available
			for u := range hostConfig.Users {
				hostConfig.User = u
				break
			}
		}

		// Remove host if no users left
		if len(hostConfig.Users) == 0 {
			delete(hosts, opts.hostname)
		}

		if err := config.SaveHostsConfig(hosts); err != nil {
			return fmt.Errorf("failed to save hosts config: %w", err)
		}
	}

	opts.streams.Success("Logged out of %s as %s", opts.hostname, user)
	return nil
}
