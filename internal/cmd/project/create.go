package project

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type createOptions struct {
	streams     *iostreams.IOStreams
	workspace   string
	key         string
	name        string
	description string
	private     bool
	jsonOut     bool
}

// NewCmdCreate creates the project create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &createOptions{
		streams: streams,
		private: true, // default to private
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Long: `Create a new project in a Bitbucket workspace.

Projects provide a way to group repositories within a workspace. The project
key must be unique within the workspace and is typically a short uppercase
identifier (e.g., "PROJ", "DEV", "CORE").`,
		Example: `  # Create a private project
  bb project create -w myworkspace -k PROJ -n "My Project"

  # Create a public project with description
  bb project create -w myworkspace -k DEV -n "Development" -d "Development projects"

  # Create a project and output as JSON
  bb project create -w myworkspace -k CORE -n "Core" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.workspace == "" {
				return fmt.Errorf("workspace is required. Use --workspace or -w to specify")
			}
			if opts.key == "" {
				return fmt.Errorf("project key is required. Use --key or -k to specify")
			}
			if opts.name == "" {
				return fmt.Errorf("project name is required. Use --name or -n to specify")
			}

			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().StringVarP(&opts.key, "key", "k", "", "Project key (required)")
	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Project name (required)")
	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "Project description")
	cmd.Flags().BoolVarP(&opts.private, "private", "p", true, "Create a private project (default: true)")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	// Get authenticated client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build create options
	createOpts := &api.ProjectCreateOptions{
		Key:         opts.key,
		Name:        opts.name,
		Description: opts.description,
		IsPrivate:   opts.private,
	}

	// Create the project
	project, err := client.CreateProject(ctx, opts.workspace, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Handle --json flag
	if opts.jsonOut {
		data, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(opts.streams.Out, string(data))
		return nil
	}

	// Success message
	opts.streams.Success("Created project %s in workspace %s", opts.key, opts.workspace)

	return nil
}
