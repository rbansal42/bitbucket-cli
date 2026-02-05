package workspace

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type setDefaultOptions struct {
	streams   *iostreams.IOStreams
	workspace string
	unset     bool
}

// NewCmdSetDefault creates the set-default command
func NewCmdSetDefault(streams *iostreams.IOStreams) *cobra.Command {
	opts := &setDefaultOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "set-default [workspace]",
		Short: "Set or unset the default workspace",
		Long: `Set a default workspace for bb commands.

When a default workspace is set, you don't need to specify the workspace
for commands that require one. The default workspace is stored in your
bb configuration.`,
		Example: `  # Set default workspace
  $ bb workspace set-default myworkspace

  # View current default workspace
  $ bb workspace set-default

  # Unset default workspace
  $ bb workspace set-default --unset`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.workspace = args[0]
			}
			return runSetDefault(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.unset, "unset", false, "Unset the default workspace")

	return cmd
}

func runSetDefault(opts *setDefaultOptions) error {
	// If --unset flag is provided
	if opts.unset {
		if err := config.SetDefaultWorkspace(""); err != nil {
			return fmt.Errorf("failed to unset default workspace: %w", err)
		}
		opts.streams.Success("Default workspace unset")
		return nil
	}

	// If no workspace provided, show current default
	if opts.workspace == "" {
		workspace, err := config.GetDefaultWorkspace()
		if err != nil {
			return fmt.Errorf("failed to get default workspace: %w", err)
		}
		if workspace == "" {
			opts.streams.Info("No default workspace set")
			opts.streams.Info("Use 'bb workspace set-default <workspace>' to set one")
		} else {
			opts.streams.Info("Default workspace: %s", workspace)
		}
		return nil
	}

	// Validate workspace exists by making an API call
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get the workspace to validate it exists
	_, err = client.GetWorkspace(ctx, opts.workspace)
	if err != nil {
		return fmt.Errorf("workspace '%s' not found or you don't have access: %w", opts.workspace, err)
	}

	// Save the default workspace
	if err := config.SetDefaultWorkspace(opts.workspace); err != nil {
		return fmt.Errorf("failed to set default workspace: %w", err)
	}

	opts.streams.Success("Default workspace set to: %s", opts.workspace)
	return nil
}
