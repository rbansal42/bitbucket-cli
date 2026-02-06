package snippet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// CreateOptions holds the options for the create command
type CreateOptions struct {
	Workspace string
	Title     string
	Private   bool
	Files     []string // File paths to include
	Streams   *iostreams.IOStreams
	JSON      bool
}

// NewCmdCreate creates the snippet create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &CreateOptions{Streams: streams}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new snippet",
		Long: `Create a new snippet in a Bitbucket workspace.

Specify files to include with --file/-f flag (can be used multiple times).
If no files are specified, reads from stdin.`,
		Example: `  # Create a snippet with one file
  bb snippet create --title "My Snippet" --file script.py --workspace myworkspace

  # Create a private snippet with multiple files
  bb snippet create --title "Config files" --file config.json --file setup.py --private --workspace myworkspace

  # Create from stdin
  echo "print('hello')" | bb snippet create --title "Hello" --workspace myworkspace`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Snippet title (required)")
	cmd.Flags().BoolVarP(&opts.Private, "private", "p", false, "Make snippet private")
	cmd.Flags().StringArrayVarP(&opts.Files, "file", "f", nil, "File to include (can be repeated)")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	cmd.MarkFlagRequired("title")

	return cmd
}

func runCreate(ctx context.Context, opts *CreateOptions) error {
	// Fall back to default workspace if not specified
	if opts.Workspace == "" {
		defaultWs, err := config.GetDefaultWorkspace()
		if err == nil && defaultWs != "" {
			opts.Workspace = defaultWs
		}
	}
	if opts.Workspace == "" {
		return fmt.Errorf("workspace is required. Use --workspace or -w to specify, or set a default with 'bb workspace set-default'")
	}

	// Validate workspace
	if _, err := cmdutil.ParseWorkspace(opts.Workspace); err != nil {
		return err
	}

	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Collect file contents
	files := make(map[string]string)

	if len(opts.Files) > 0 {
		// Read from specified files
		for _, filePath := range opts.Files {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			filename := filepath.Base(filePath)
			files[filename] = string(content)
		}
	} else {
		// Read from stdin
		if !opts.Streams.IsStdinTTY() {
			content, err := io.ReadAll(opts.Streams.In)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			if len(content) == 0 {
				return fmt.Errorf("no content provided. Use --file to specify files or pipe content to stdin")
			}
			files["snippet.txt"] = string(content)
		} else {
			return fmt.Errorf("no files specified. Use --file to specify files or pipe content to stdin")
		}
	}

	// Create snippet
	snippet, err := client.CreateSnippet(ctx, opts.Workspace, opts.Title, opts.Private, files)
	if err != nil {
		return fmt.Errorf("failed to create snippet: %w", err)
	}

	// Output result
	if opts.JSON {
		return outputCreateJSON(opts.Streams, snippet)
	}

	opts.Streams.Success("Created snippet %d in workspace %s", snippet.ID, opts.Workspace)
	if snippet.Links.HTML.Href != "" {
		fmt.Fprintf(opts.Streams.Out, "URL: %s\n", snippet.Links.HTML.Href)
	}

	return nil
}

func outputCreateJSON(streams *iostreams.IOStreams, snippet *api.Snippet) error {
	output := map[string]interface{}{
		"id":         snippet.ID,
		"title":      snippet.Title,
		"is_private": snippet.IsPrivate,
		"created_on": snippet.CreatedOn,
	}

	if snippet.Links.HTML.Href != "" {
		output["url"] = snippet.Links.HTML.Href
	}

	// Include file names
	if len(snippet.Files) > 0 {
		fileNames := make([]string, 0, len(snippet.Files))
		for name := range snippet.Files {
			fileNames = append(fileNames, name)
		}
		output["files"] = fileNames
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}
