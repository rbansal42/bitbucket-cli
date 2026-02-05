package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/browser"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type viewOptions struct {
	streams   *iostreams.IOStreams
	repoArg   string
	web       bool
	jsonOut   bool
	workspace string
	repoSlug  string
}

// NewCmdView creates the view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &viewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view [<workspace/repo>]",
		Short: "View a repository",
		Long: `Display the details of a repository.

With no arguments, the repository for the current directory is displayed
(detected from git remote).

You can specify a repository using the workspace/repo format.`,
		Example: `  # View the current repository
  bb repo view

  # View a specific repository
  bb repo view myworkspace/myrepo

  # Open repository in browser
  bb repo view --web

  # Output as JSON
  bb repo view --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.repoArg = args[0]
			}

			return runView(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open the repository in a web browser")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "Output in JSON format")

	return cmd
}

func runView(opts *viewOptions) error {
	// Resolve repository
	var err error
	opts.workspace, opts.repoSlug, err = cmdutil.ParseRepository(opts.repoArg)
	if err != nil {
		return err
	}

	// Get authenticated client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Fetch repository details
	repo, err := client.GetRepository(ctx, opts.workspace, opts.repoSlug)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Handle --web flag
	if opts.web {
		if err := browser.Open(repo.Links.HTML.Href); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.streams.Success("Opened %s in your browser", repo.Links.HTML.Href)
		return nil
	}

	// Handle --json flag
	if opts.jsonOut {
		return outputJSON(opts.streams, repo)
	}

	// Display formatted output
	return displayRepo(opts.streams, repo)
}

func outputJSON(streams *iostreams.IOStreams, repo *api.RepositoryFull) error {
	data, err := json.MarshalIndent(repo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func displayRepo(streams *iostreams.IOStreams, repo *api.RepositoryFull) error {
	// Header - workspace/repo
	fmt.Fprintf(streams.Out, "%s\n\n", repo.FullName)

	// Description
	if repo.Description != "" {
		fmt.Fprintf(streams.Out, "Description: %s\n", repo.Description)
	} else {
		fmt.Fprintf(streams.Out, "Description: (no description)\n")
	}

	// Visibility
	visibility := "public"
	if repo.IsPrivate {
		visibility = "private"
	}
	fmt.Fprintf(streams.Out, "Visibility:  %s\n", visibility)

	// Language
	if repo.Language != "" {
		fmt.Fprintf(streams.Out, "Language:    %s\n", repo.Language)
	}

	// Size
	fmt.Fprintf(streams.Out, "Size:        %s\n", formatSize(repo.Size))

	// Default branch
	if repo.MainBranch != nil {
		fmt.Fprintf(streams.Out, "Default:     %s\n", repo.MainBranch.Name)
	}

	// Project
	if repo.Project != nil && repo.Project.Key != "" {
		fmt.Fprintf(streams.Out, "Project:     %s\n", repo.Project.Key)
	}

	// Clone URLs
	fmt.Fprintln(streams.Out)
	fmt.Fprintln(streams.Out, "Clone URLs:")
	for _, clone := range repo.Links.Clone {
		name := strings.ToUpper(clone.Name)
		fmt.Fprintf(streams.Out, "  %-5s  %s\n", name+":", clone.Href)
	}

	// Browser URL
	fmt.Fprintln(streams.Out)
	fmt.Fprintf(streams.Out, "View in browser: %s\n", repo.Links.HTML.Href)

	return nil
}

// formatSize formats bytes into human readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
