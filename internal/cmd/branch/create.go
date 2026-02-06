package branch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// CreateOptions holds the options for the create command
type CreateOptions struct {
	BranchName string
	Repo       string
	Target     string
	JSON       bool
	Streams    *iostreams.IOStreams
}

// NewCmdCreate creates the branch create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &CreateOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "create <branch-name>",
		Short: "Create a new branch",
		Long: `Create a new branch in a Bitbucket repository.

You must specify the target branch, tag, or commit to branch from using --target.
By default, this command detects the repository from your git remote.`,
		Example: `  # Create a branch from main
  bb branch create feature-branch --target main

  # Create a branch from a specific commit
  bb branch create hotfix-branch --target abc1234

  # Create a branch in a specific repository
  bb branch create feature-branch --target main --repo myworkspace/myrepo

  # Output as JSON
  bb branch create feature-branch --target main --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.BranchName = args[0]
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format (detects from git remote if not specified)")
	cmd.Flags().StringVarP(&opts.Target, "target", "t", "", "Branch, tag, or commit to branch from (required)")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	cmd.MarkFlagRequired("target")

	return cmd
}

func runCreate(ctx context.Context, opts *CreateOptions) error {
	// Parse repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.Repo)
	if err != nil {
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

	// Try to resolve target as a branch first to get the commit hash
	commitHash := opts.Target
	branch, err := client.GetBranch(ctx, workspace, repoSlug, opts.Target)
	if err == nil && branch.Target != nil {
		// Target is a branch, use its commit hash
		commitHash = branch.Target.Hash
	}
	// If GetBranch fails, assume target is already a commit hash or tag
	// The API will validate if it's a valid reference

	// Create branch options
	createOpts := &api.BranchCreateOptions{
		Name: opts.BranchName,
	}
	createOpts.Target.Hash = commitHash

	// Create the branch
	newBranch, err := client.CreateBranch(ctx, workspace, repoSlug, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Output results
	if opts.JSON {
		return outputCreateJSON(opts.Streams, newBranch)
	}

	opts.Streams.Success("Created branch %s in %s/%s", opts.BranchName, workspace, repoSlug)
	return nil
}

func outputCreateJSON(streams *iostreams.IOStreams, branch *api.BranchFull) error {
	output := map[string]interface{}{
		"name": branch.Name,
	}
	if branch.Target != nil {
		output["commit"] = branch.Target.Hash
		output["message"] = branch.Target.Message
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}
