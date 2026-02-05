package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/browser"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type viewOptions struct {
	streams       *iostreams.IOStreams
	workspaceSlug string
	web           bool
	jsonOut       bool
}

// NewCmdView creates the workspace view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &viewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view <workspace>",
		Short: "View workspace details",
		Long: `Display the details of a Bitbucket workspace.

Shows workspace name, slug, UUID, type, privacy setting, creation date,
and the browser URL.`,
		Example: `  # View a workspace
  bb workspace view myworkspace

  # Open workspace in browser
  bb workspace view myworkspace --web

  # Output as JSON
  bb workspace view myworkspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.workspaceSlug = args[0]
			return runView(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open the workspace in a web browser")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

func runView(ctx context.Context, opts *viewOptions) error {
	// Get authenticated client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Fetch workspace details
	ws, err := client.GetWorkspace(ctx, opts.workspaceSlug)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Handle --web flag
	if opts.web {
		url := ws.Links.HTML.Href
		if url == "" {
			url = fmt.Sprintf("https://bitbucket.org/%s", ws.Slug)
		}
		if err := browser.Open(url); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.streams.Success("Opened %s in your browser", url)
		return nil
	}

	// Handle --json flag
	if opts.jsonOut {
		return outputViewJSON(opts.streams, ws)
	}

	// Display formatted output
	return displayWorkspace(opts.streams, ws)
}

func outputViewJSON(streams *iostreams.IOStreams, ws *api.WorkspaceFull) error {
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func displayWorkspace(streams *iostreams.IOStreams, ws *api.WorkspaceFull) error {
	// Header - workspace name
	fmt.Fprintf(streams.Out, "%s\n\n", ws.Name)

	// Details
	fmt.Fprintf(streams.Out, "Name:     %s\n", ws.Name)
	fmt.Fprintf(streams.Out, "Slug:     %s\n", ws.Slug)
	fmt.Fprintf(streams.Out, "UUID:     %s\n", ws.UUID)
	fmt.Fprintf(streams.Out, "Type:     %s\n", ws.Type)

	// Privacy
	privacy := "public"
	if ws.IsPrivate {
		privacy = "private"
	}
	fmt.Fprintf(streams.Out, "Privacy:  %s\n", privacy)

	// Created date
	if !ws.CreatedOn.IsZero() {
		fmt.Fprintf(streams.Out, "Created:  %s\n", ws.CreatedOn.Format("Jan 02, 2006"))
	}

	// Browser URL
	url := ws.Links.HTML.Href
	if url == "" {
		url = fmt.Sprintf("https://bitbucket.org/%s", ws.Slug)
	}
	fmt.Fprintln(streams.Out)
	fmt.Fprintf(streams.Out, "View in browser: %s\n", url)

	return nil
}
