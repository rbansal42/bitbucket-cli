package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	coreconfig "github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdConfigSet creates the config set command
func NewCmdConfigSet(streams *iostreams.IOStreams) *cobra.Command {
	var host string

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Update configuration with a value for the given key",
		Long: `Update configuration with a value for the given key.

Available keys:
  git_protocol   The protocol to use for git operations (ssh, https)
  editor         The editor to use for composing text
  prompt         Whether to enable interactive prompts (enabled, disabled)
  pager          The pager to use for output
  browser        The browser to use for opening URLs
  http_timeout   HTTP request timeout in seconds`,
		Example: `  # Set the git protocol to HTTPS
  bb config set git_protocol https

  # Set the editor to vim
  bb config set editor vim

  # Disable interactive prompts
  bb config set prompt disabled

  # Set HTTP timeout to 60 seconds
  bb config set http_timeout 60`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])
			value := args[1]

			// Load config
			cfg, err := coreconfig.LoadConfig()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			// Validate and set value
			if err := setConfigValue(cfg, key, value); err != nil {
				return err
			}

			// Save config
			if err := coreconfig.SaveConfig(cfg); err != nil {
				return fmt.Errorf("could not save config: %w", err)
			}

			streams.Success("Set %s to %s", key, value)
			return nil
		},
	}

	cmd.Flags().StringVarP(&host, "host", "h", "", "Set per-host configuration")

	return cmd
}

// setConfigValue sets a config value with validation
func setConfigValue(cfg *coreconfig.Config, key, value string) error {
	switch key {
	case "git_protocol":
		if value != "ssh" && value != "https" {
			return fmt.Errorf("invalid git_protocol: %s (must be 'ssh' or 'https')", value)
		}
		cfg.GitProtocol = value

	case "editor":
		cfg.Editor = value

	case "prompt":
		if value != "enabled" && value != "disabled" {
			return fmt.Errorf("invalid prompt value: %s (must be 'enabled' or 'disabled')", value)
		}
		cfg.Prompt = value

	case "pager":
		cfg.Pager = value

	case "browser":
		cfg.Browser = value

	case "http_timeout":
		timeout, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid http_timeout: %s (must be a number)", value)
		}
		if timeout < 1 {
			return fmt.Errorf("http_timeout must be at least 1 second")
		}
		cfg.HTTPTimeout = timeout

	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}
