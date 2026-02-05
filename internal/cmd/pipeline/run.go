package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/git"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

type runOptions struct {
	streams *iostreams.IOStreams
	branch  string
	commit  string
	custom  string
	repo    string
}

// NewCmdRun creates the run command
func NewCmdRun(streams *iostreams.IOStreams) *cobra.Command {
	opts := &runOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a pipeline run",
		Long: `Trigger a new pipeline run for the repository.

By default, the pipeline runs on the current branch. You can specify a different
branch with --branch, a specific commit with --commit, or trigger a custom 
pipeline defined in bitbucket-pipelines.yml with --custom.`,
		Example: `  # Run pipeline on current branch
  bb pipeline run

  # Run pipeline on a specific branch
  bb pipeline run --branch develop

  # Run pipeline on a specific commit
  bb pipeline run --commit abc1234

  # Run a custom pipeline
  bb pipeline run --custom my-custom-pipeline

  # Run pipeline for a different repository
  bb pipeline run --repo myworkspace/myrepo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelineRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "Branch to run pipeline on (default: current branch or main)")
	cmd.Flags().StringVar(&opts.commit, "commit", "", "Specific commit hash to run pipeline on")
	cmd.Flags().StringVar(&opts.custom, "custom", "", "Custom pipeline name (for custom pipelines in bitbucket-pipelines.yml)")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runPipelineRun(opts *runOptions) error {
	// Resolve repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.repo)
	if err != nil {
		return err
	}

	// Determine the branch to use
	branch := opts.branch
	if branch == "" {
		// Try to get current branch from git
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			// Fall back to main if we can't detect the current branch
			branch = "main"
		} else {
			branch = currentBranch
		}
	}

	// Build pipeline run options
	pipelineOpts := buildPipelineRunOptions(branch, opts.commit, opts.custom)

	// Get authenticated client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Display what we're about to do
	if opts.custom != "" {
		opts.streams.Info("Triggering custom pipeline '%s' on branch %s in %s/%s...", opts.custom, branch, workspace, repoSlug)
	} else if opts.commit != "" {
		opts.streams.Info("Triggering pipeline for commit %s in %s/%s...", opts.commit, workspace, repoSlug)
	} else {
		opts.streams.Info("Triggering pipeline on branch %s in %s/%s...", branch, workspace, repoSlug)
	}

	// Trigger the pipeline
	pipeline, err := client.RunPipeline(ctx, workspace, repoSlug, pipelineOpts)
	if err != nil {
		return fmt.Errorf("failed to trigger pipeline: %w", err)
	}

	// Print success output
	opts.streams.Success("Pipeline #%d triggered", pipeline.BuildNumber)

	// Print pipeline URL
	pipelineURL := fmt.Sprintf("https://bitbucket.org/%s/%s/pipelines/results/%d",
		workspace, repoSlug, pipeline.BuildNumber)
	fmt.Fprintf(opts.streams.Out, "  %s\n", pipelineURL)

	return nil
}

// buildPipelineRunOptions constructs the API options for running a pipeline
func buildPipelineRunOptions(branch, commit, custom string) *api.PipelineRunOptions {
	target := &api.PipelineTarget{
		Type:    "pipeline_ref_target",
		RefType: "branch",
		RefName: branch,
	}

	// If a specific commit is provided, include it
	if commit != "" {
		target.Commit = &api.PipelineCommit{
			Type: "commit",
			Hash: commit,
		}
	}

	// If custom pipeline is specified, add the selector
	if custom != "" {
		target.Selector = &api.PipelineSelector{
			Type:    "custom",
			Pattern: custom,
		}
	}

	return &api.PipelineRunOptions{
		Target: target,
	}
}
