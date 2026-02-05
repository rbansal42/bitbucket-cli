package repo

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
)

type createOptions struct {
	streams     *iostreams.IOStreams
	name        string
	description string
	private     bool
	public      bool
	workspace   string
	project     string
	clone       bool
	gitignore   string
}

// NewCmdCreate creates the repo create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &createOptions{
		streams: streams,
		private: true, // default to private
	}

	cmd := &cobra.Command{
		Use:   "create [<name>]",
		Short: "Create a new repository",
		Long: `Create a new repository in a Bitbucket workspace.

The repository name can be provided as an argument or with the --name flag.
If no name is provided, you will be prompted to enter one interactively.

By default, repositories are created as private. Use --public to create
a public repository instead.`,
		Example: `  # Create a private repository interactively
  bb repo create

  # Create a private repository with a specific name
  bb repo create myrepo

  # Create a public repository
  bb repo create myrepo --public

  # Create a repository with description
  bb repo create myrepo -d "My awesome project"

  # Create a repository in a specific workspace
  bb repo create myrepo -w myworkspace

  # Create and clone the repository
  bb repo create myrepo --clone

  # Create a repository in a project
  bb repo create myrepo -p PROJ`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.name = args[0]
			}

			// Handle conflicting flags - only error if both were explicitly set
			privateChanged := cmd.Flags().Changed("private")
			publicChanged := cmd.Flags().Changed("public")
			if privateChanged && publicChanged {
				return fmt.Errorf("cannot specify both --private and --public")
			}

			// If --public is set, private should be false
			if opts.public {
				opts.private = false
			}

			return runCreate(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Repository name")
	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "Repository description")
	cmd.Flags().BoolVar(&opts.private, "private", true, "Create a private repository (default)")
	cmd.Flags().BoolVar(&opts.public, "public", false, "Create a public repository")
	cmd.Flags().StringVarP(&opts.workspace, "workspace", "w", "", "Workspace to create repository in")
	cmd.Flags().StringVarP(&opts.project, "project", "p", "", "Project key to assign repository to")
	cmd.Flags().BoolVarP(&opts.clone, "clone", "c", false, "Clone the repository after creation")
	cmd.Flags().StringVar(&opts.gitignore, "gitignore", "", "Initialize with gitignore template")

	return cmd
}

func runCreate(opts *createOptions) error {
	// Get authenticated client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Determine workspace
	workspace := opts.workspace
	if workspace == "" {
		workspace, err = getDefaultWorkspace(ctx, client, opts.streams)
		if err != nil {
			return fmt.Errorf("could not determine workspace: %w\nUse --workspace to specify", err)
		}
	}

	// Prompt for name if not provided
	if opts.name == "" {
		if !opts.streams.IsStdinTTY() {
			return fmt.Errorf("repository name is required when not running interactively")
		}

		name, err := promptForName(opts.streams)
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("repository name is required")
		}
		opts.name = name
	}

	// Build create options
	createOpts := &api.RepositoryCreateOptions{
		Name:        opts.name,
		Description: opts.description,
		IsPrivate:   opts.private,
	}

	if opts.project != "" {
		createOpts.Project = &api.Project{Key: opts.project}
	}

	opts.streams.Info("Creating repository %s/%s...", workspace, opts.name)

	// Create the repository
	repo, err := client.CreateRepository(ctx, workspace, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	// Success message
	opts.streams.Success("Created repository %s", repo.FullName)
	fmt.Fprintln(opts.streams.Out)

	// Get preferred protocol for clone URL
	protocol := getPreferredProtocol()
	cloneURL := getCloneURL(repo.Links, protocol)
	fmt.Fprintf(opts.streams.Out, "Clone URL: %s\n", cloneURL)

	// Clone if requested
	if opts.clone {
		fmt.Fprintln(opts.streams.Out)
		opts.streams.Info("Cloning repository...")

		if err := git.Clone(cloneURL, opts.name); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		opts.streams.Success("Cloned to %s/", opts.name)
	}

	return nil
}

// getDefaultWorkspace attempts to get the default workspace for the user
func getDefaultWorkspace(ctx context.Context, client *api.Client, streams *iostreams.IOStreams) (string, error) {
	// First, try to get from hosts config (active user)
	hosts, err := config.LoadHostsConfig()
	if err == nil {
		// Get active user's workspace (often same as username)
		user := hosts.GetActiveUser(config.DefaultHost)
		if user != "" {
			// Try to use username as workspace (common pattern)
			return user, nil
		}
	}

	// Try to get current user from API and use their workspace
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		return "", fmt.Errorf("could not get current user: %w", err)
	}

	if user.Username != "" {
		return user.Username, nil
	}

	return "", fmt.Errorf("could not determine default workspace")
}

// promptForName prompts the user to enter a repository name
func promptForName(streams *iostreams.IOStreams) (string, error) {
	fmt.Fprint(streams.Out, "Repository name: ")

	var reader *bufio.Reader
	if r, ok := streams.In.(io.Reader); ok {
		reader = bufio.NewReader(r)
	} else {
		reader = bufio.NewReader(os.Stdin)
	}

	name, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(name), nil
}
