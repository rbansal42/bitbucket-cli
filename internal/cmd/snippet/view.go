package snippet

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/browser"
	"github.com/rbansal42/bb/internal/iostreams"
)

// ViewOptions holds the options for the view command
type ViewOptions struct {
	Workspace string
	SnippetID string
	Web       bool
	JSON      bool
	Raw       bool // Show raw file content
	Streams   *iostreams.IOStreams
}

// NewCmdView creates the snippet view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ViewOptions{Streams: streams}

	cmd := &cobra.Command{
		Use:   "view <snippet-id>",
		Short: "View a snippet's details",
		Long: `View details of a Bitbucket snippet.

By default, shows snippet metadata. Use --raw to view file contents.`,
		Example: `  # View snippet details
  bb snippet view abc123 --workspace myworkspace

  # View snippet file contents
  bb snippet view abc123 --workspace myworkspace --raw

  # Open snippet in browser
  bb snippet view abc123 --workspace myworkspace --web

  # Output as JSON
  bb snippet view abc123 --workspace myworkspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.SnippetID = args[0]
			return runView(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().BoolVar(&opts.Web, "web", false, "Open snippet in browser")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.Raw, "raw", false, "Show raw file contents")

	cmd.MarkFlagRequired("workspace")

	return cmd
}

func runView(ctx context.Context, opts *ViewOptions) error {
	// Validate workspace
	if err := parseWorkspace(opts.Workspace); err != nil {
		return err
	}

	// Get API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Fetch snippet
	snippet, err := client.GetSnippet(ctx, opts.Workspace, opts.SnippetID)
	if err != nil {
		return fmt.Errorf("failed to get snippet: %w", err)
	}

	// Open in browser
	if opts.Web {
		url := snippet.Links.HTML.Href
		if url == "" {
			return fmt.Errorf("no URL available for this snippet")
		}
		if err := browser.Open(url); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.Streams.Success("Opened %s in your browser", url)
		return nil
	}

	// JSON output
	if opts.JSON {
		return outputViewJSON(opts.Streams, snippet)
	}

	// Raw file contents
	if opts.Raw {
		return outputRawFiles(ctx, client, opts, snippet)
	}

	// Default: show snippet details
	return outputSnippetDetails(opts.Streams, snippet)
}

func outputViewJSON(streams *iostreams.IOStreams, snippet *api.Snippet) error {
	data, err := json.MarshalIndent(snippet, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputSnippetDetails(streams *iostreams.IOStreams, snippet *api.Snippet) error {
	// Title and ID
	if snippet.Title != "" {
		fmt.Fprintf(streams.Out, "%s\n", snippet.Title)
	} else {
		fmt.Fprintf(streams.Out, "(untitled snippet)\n")
	}
	fmt.Fprintf(streams.Out, "ID: %d\n\n", snippet.ID)

	// Visibility
	visibility := "public"
	if snippet.IsPrivate {
		visibility = "private"
	}
	fmt.Fprintf(streams.Out, "Visibility:  %s\n", visibility)

	// Owner
	if snippet.Owner != nil {
		displayName := snippet.Owner.DisplayName
		if displayName == "" {
			displayName = snippet.Owner.Username
		}
		if displayName != "" {
			fmt.Fprintf(streams.Out, "Owner:       %s\n", displayName)
		}
	}

	// Timestamps
	if snippet.CreatedOn != "" {
		created := formatTimestamp(snippet.CreatedOn)
		fmt.Fprintf(streams.Out, "Created:     %s\n", created)
	}
	if snippet.UpdatedOn != "" {
		updated := formatTimestamp(snippet.UpdatedOn)
		fmt.Fprintf(streams.Out, "Updated:     %s\n", updated)
	}

	// Files (sorted for deterministic output)
	if len(snippet.Files) > 0 {
		fmt.Fprintln(streams.Out)
		fmt.Fprintln(streams.Out, "Files:")
		filenames := make([]string, 0, len(snippet.Files))
		for filename := range snippet.Files {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)
		for _, filename := range filenames {
			fmt.Fprintf(streams.Out, "  %s\n", filename)
		}
	}

	// URL
	if snippet.Links.HTML.Href != "" {
		fmt.Fprintln(streams.Out)
		fmt.Fprintf(streams.Out, "View in browser: %s\n", snippet.Links.HTML.Href)
	}

	return nil
}

func outputRawFiles(ctx context.Context, client *api.Client, opts *ViewOptions, snippet *api.Snippet) error {
	if len(snippet.Files) == 0 {
		opts.Streams.Info("No files in this snippet")
		return nil
	}

	// Get sorted list of filenames for deterministic output
	filenames := make([]string, 0, len(snippet.Files))
	for filename := range snippet.Files {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)

	// Fetch and display each file's content
	isFirst := true
	for _, filename := range filenames {
		if !isFirst {
			fmt.Fprintln(opts.Streams.Out) // Blank line between files
		}
		isFirst = false

		// Print file header
		fmt.Fprintf(opts.Streams.Out, "==> %s <==\n", filename)

		content, err := client.GetSnippetFileContent(ctx, opts.Workspace, opts.SnippetID, filename)
		if err != nil {
			fmt.Fprintf(opts.Streams.ErrOut, "Error fetching %s: %v\n", filename, err)
			continue
		}

		// Print file content
		contentStr := string(content)
		// Ensure content ends with newline for clean output
		if !strings.HasSuffix(contentStr, "\n") {
			contentStr += "\n"
		}
		fmt.Fprint(opts.Streams.Out, contentStr)
	}

	return nil
}

// formatTimestamp formats an ISO 8601 timestamp into a human-readable format
func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Try alternate format
		t, err = time.Parse("2006-01-02T15:04:05.999999-07:00", ts)
		if err != nil {
			return ts // Return as-is if parsing fails
		}
	}
	return t.Format("Jan 02, 2006 15:04 MST")
}
