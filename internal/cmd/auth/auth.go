package auth

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdAuth creates the auth command
func NewCmdAuth(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Authenticate bb and git with Bitbucket",
		Long: `Authenticate with Bitbucket to enable API access.

The default authentication mode is interactive and uses OAuth 2.0.
After completing the authentication flow, your access token is stored
securely in your system keychain.

Alternatively, you can use workspace or repository access tokens by
setting the BB_TOKEN environment variable or using --with-token.`,
	}

	cmd.AddCommand(NewCmdLogin(streams))
	cmd.AddCommand(NewCmdLogout(streams))
	cmd.AddCommand(NewCmdStatus(streams))
	cmd.AddCommand(NewCmdToken(streams))

	return cmd
}
