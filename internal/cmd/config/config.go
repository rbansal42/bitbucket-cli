package config

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdConfig creates the config command
func NewCmdConfig(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <command>",
		Short: "Manage configuration for bb",
		Long: `Display or change configuration settings for bb.

Configuration is stored in ~/.config/bb/config.yml or the directory
specified by the BB_CONFIG_DIR environment variable.

Available settings:
  git_protocol   The protocol to use for git operations (ssh, https)
  editor         The editor to use for composing text
  prompt         Whether to enable interactive prompts (enabled, disabled)
  pager          The pager to use for output
  browser        The browser to use for opening URLs
  http_timeout   HTTP request timeout in seconds`,
	}

	cmd.AddCommand(NewCmdConfigGet(streams))
	cmd.AddCommand(NewCmdConfigSet(streams))
	cmd.AddCommand(NewCmdConfigList(streams))

	return cmd
}
