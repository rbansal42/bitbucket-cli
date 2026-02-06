package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	coreconfig "github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdConfigGet creates the config get command
func NewCmdConfigGet(streams *iostreams.IOStreams) *cobra.Command {
	var host string

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print the value of a configuration key",
		Long: `Print the value of a configuration key.

Available keys:
  git_protocol   The protocol to use for git operations
  editor         The editor to use for composing text
  prompt         Whether to enable interactive prompts
  pager          The pager to use for output
  browser        The browser to use for opening URLs
  http_timeout   HTTP request timeout in seconds`,
		Example: `  # Get the git protocol setting
  bb config get git_protocol

  # Get the editor setting
  bb config get editor`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])

			// Load config
			cfg, err := coreconfig.LoadConfig()
			if err != nil {
				return fmt.Errorf("could not load config: %w", err)
			}

			// Get value using reflection
			value, err := getConfigValue(cfg, key)
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, value)
			return nil
		},
	}

	cmd.Flags().StringVarP(&host, "host", "h", "", "Get per-host configuration")

	return cmd
}

// getConfigValue returns the value of a config key
func getConfigValue(cfg *coreconfig.Config, key string) (string, error) {
	// Map config keys to struct fields
	keyMap := map[string]string{
		"git_protocol": "GitProtocol",
		"editor":       "Editor",
		"prompt":       "Prompt",
		"pager":        "Pager",
		"browser":      "Browser",
		"http_timeout": "HTTPTimeout",
	}

	fieldName, ok := keyMap[key]
	if !ok {
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}

	v := reflect.ValueOf(cfg).Elem()
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return "", fmt.Errorf("configuration key not found: %s", key)
	}

	// Handle different types
	switch field.Kind() {
	case reflect.String:
		return field.String(), nil
	case reflect.Int, reflect.Int64:
		return fmt.Sprintf("%d", field.Int()), nil
	case reflect.Bool:
		return fmt.Sprintf("%t", field.Bool()), nil
	default:
		return fmt.Sprintf("%v", field.Interface()), nil
	}
}
