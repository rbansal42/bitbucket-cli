package snippet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

// EditOptions holds the options for the edit command
type EditOptions struct {
	Workspace string
	SnippetID string
	Title     string
	Files     []string // File paths to update
	JSON      bool
	Streams   *iostreams.IOStreams
}

// NewCmdEdit creates the snippet edit command
func NewCmdEdit(streams *iostreams.IOStreams) *cobra.Command {
	opts := &EditOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "edit <snippet-id>",
		Short: "Edit an existing snippet",
		Long: `Edit an existing snippet in a Bitbucket workspace.

You can update the title and/or add/update files.`,
		Example: `  # Update snippet title
  bb snippet edit abc123 --title "New Title" --workspace myworkspace

  # Update snippet files
  bb snippet edit abc123 --file updated.py --workspace myworkspace

  # Update both title and files
  bb snippet edit abc123 --title "New Title" --file updated.py --workspace myworkspace`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SnippetID = args[0]
			return runEdit(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "New snippet title")
	cmd.Flags().StringArrayVarP(&opts.Files, "file", "f", nil, "File to update (can be repeated)")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	cmd.MarkFlagRequired("workspace")

	return cmd
}

func runEdit(ctx context.Context, opts *EditOptions) error {
	// Validate workspace
	if err := parseWorkspace(opts.Workspace); err != nil {
		return err
	}

	// Must have something to update
	if opts.Title == "" && len(opts.Files) == 0 {
		return fmt.Errorf("nothing to update. Specify --title and/or --file")
	}

	// Get API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Collect file contents
	files := make(map[string]string)
	for _, filePath := range opts.Files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		filename := filepath.Base(filePath)
		files[filename] = string(content)
	}

	// Update snippet
	snippet, err := client.UpdateSnippet(ctx, opts.Workspace, opts.SnippetID, opts.Title, files)
	if err != nil {
		return fmt.Errorf("failed to update snippet: %w", err)
	}

	// Output result
	if opts.JSON {
		return outputEditJSON(opts.Streams, snippet)
	}

	opts.Streams.Success("Updated snippet %d", snippet.ID)
	return nil
}

func outputEditJSON(streams *iostreams.IOStreams, snippet *api.Snippet) error {
	data, err := json.MarshalIndent(snippet, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}
