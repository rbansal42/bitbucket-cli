package project

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/browser"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type viewOptions struct {
	streams   *iostreams.IOStreams
	workspace string
	key       string
	web       bool
	jsonOut   bool
}

// NewCmdView creates the project view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &viewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view <project-key>",
		Short: "View a project",
		Long: `Display the details of a Bitbucket project.

The project key is required as an argument. Project keys are typically
short uppercase identifiers like "PROJ" or "DEV".`,
		Example: `  # View a project
  bb project view PROJ --workspace myworkspace

  # Open project in browser
  bb project view PROJ -w myworkspace --web

  # Output as JSON
  bb project view PROJ -w myworkspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.key = args[0]

			if opts.workspace == "" {
				defaultWs, err := config.GetDefaultWorkspace()
				if err == nil && defaultWs != "" {
					opts.workspace = defaultWs
				}
			}
			if opts.workspace == "" {
				return fmt.Errorf("workspace is required. Use --workspace or -w to specify, or set a default with 'bb workspace set-default'")
			}

			return runView(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().BoolVar(&opts.web, "web", false, "Open the project in a web browser")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

func runView(ctx context.Context, opts *viewOptions) error {
	// Get authenticated client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Fetch project details
	project, err := client.GetProject(ctx, opts.workspace, opts.key)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Handle --web flag
	if opts.web {
		if err := browser.Open(project.Links.HTML.Href); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.streams.Success("Opened %s in your browser", project.Links.HTML.Href)
		return nil
	}

	// Handle --json flag
	if opts.jsonOut {
		return outputViewJSON(opts.streams, project)
	}

	// Display formatted output
	return displayProject(opts.streams, project)
}

func outputViewJSON(streams *iostreams.IOStreams, project *api.ProjectFull) error {
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func displayProject(streams *iostreams.IOStreams, project *api.ProjectFull) error {
	// Header - Name (Key)
	fmt.Fprintf(streams.Out, "%s (%s)\n\n", project.Name, project.Key)

	// Description
	if project.Description != "" {
		fmt.Fprintf(streams.Out, "Description: %s\n", project.Description)
	} else {
		fmt.Fprintf(streams.Out, "Description: (no description)\n")
	}

	// Visibility
	visibility := "public"
	if project.IsPrivate {
		visibility = "private"
	}
	fmt.Fprintf(streams.Out, "Visibility:  %s\n", visibility)

	// UUID
	fmt.Fprintf(streams.Out, "UUID:        %s\n", project.UUID)

	// Created
	fmt.Fprintf(streams.Out, "Created:     %s\n", formatTime(project.CreatedOn))

	// Updated
	fmt.Fprintf(streams.Out, "Updated:     %s\n", formatTime(project.UpdatedOn))

	// Browser URL
	fmt.Fprintln(streams.Out)
	fmt.Fprintf(streams.Out, "View in browser: %s\n", project.Links.HTML.Href)

	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("Jan 02, 2006 15:04 MST")
}
