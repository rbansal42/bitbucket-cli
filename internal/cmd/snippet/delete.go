package snippet

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// DeleteOptions holds the options for the delete command
type DeleteOptions struct {
	Workspace string
	SnippetID string
	Force     bool
	JSON      bool
	Streams   *iostreams.IOStreams
}

// NewCmdDelete creates the snippet delete command
func NewCmdDelete(streams *iostreams.IOStreams) *cobra.Command {
	opts := &DeleteOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "delete <snippet-id>",
		Short: "Delete a snippet",
		Long: `Delete a snippet from a Bitbucket workspace.

By default, you will be prompted to confirm the deletion.
Use --force to skip the confirmation prompt.`,
		Example: `  # Delete with confirmation
  bb snippet delete abc123 --workspace myworkspace

  # Delete without confirmation
  bb snippet delete abc123 --workspace myworkspace --force`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SnippetID = args[0]
			return runDelete(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	cmd.MarkFlagRequired("workspace")

	return cmd
}

func runDelete(ctx context.Context, opts *DeleteOptions) error {
	// Validate workspace
	if err := parseWorkspace(opts.Workspace); err != nil {
		return err
	}

	// If not forced, prompt for confirmation
	if !opts.Force {
		// Require TTY for interactive confirmation
		if !opts.Streams.IsStdinTTY() {
			return fmt.Errorf("cannot confirm deletion in non-interactive mode\nUse --force flag to skip confirmation")
		}

		fmt.Fprintf(opts.Streams.Out, "Delete snippet %s from %s? [y/N]: ", opts.SnippetID, opts.Workspace)

		reader := bufio.NewReader(opts.Streams.In)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			opts.Streams.Info("Deletion cancelled")
			return nil
		}
	}

	// Get API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Delete snippet
	if err := client.DeleteSnippet(ctx, opts.Workspace, opts.SnippetID); err != nil {
		return fmt.Errorf("failed to delete snippet: %w", err)
	}

	// Output result
	if opts.JSON {
		output := map[string]interface{}{
			"deleted":    true,
			"snippet_id": opts.SnippetID,
			"workspace":  opts.Workspace,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(opts.Streams.Out, string(data))
		return nil
	}

	opts.Streams.Success("Deleted snippet %s", opts.SnippetID)
	return nil
}
