package config

import (
	"fmt"

	"github.com/spf13/cobra"

	coreconfig "github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdConfigList creates the config list command
func NewCmdConfigList(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Print a list of configuration keys and values",
		Long: `Print a list of configuration keys and values.

Shows the current configuration settings from the config file.`,
		Example: `  # List all configuration settings
  bb config list`,
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			cfg, err := coreconfig.LoadConfig()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			// Print configuration values
			printConfig(streams, cfg)

			return nil
		},
	}

	return cmd
}

// printConfig prints all configuration values
func printConfig(streams *iostreams.IOStreams, cfg *coreconfig.Config) {
	// Define the order and format of output
	settings := []struct {
		key   string
		value interface{}
	}{
		{"git_protocol", cfg.GitProtocol},
		{"editor", cfg.Editor},
		{"prompt", cfg.Prompt},
		{"pager", cfg.Pager},
		{"browser", cfg.Browser},
		{"http_timeout", cfg.HTTPTimeout},
	}

	for _, s := range settings {
		value := formatValue(s.value)
		if value != "" {
			fmt.Fprintf(streams.Out, "%s=%s\n", s.key, value)
		}
	}
}

// formatValue formats a config value for display
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		if val == 0 {
			return ""
		}
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
