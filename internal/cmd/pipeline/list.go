package pipeline

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// ListOptions holds the options for the list command
type ListOptions struct {
	Status  string
	Branch  string
	Limit   int
	JSON    bool
	Repo    string
	Streams *iostreams.IOStreams
}

// NewCmdList creates the pipeline list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipelines in a repository",
		Long: `List pipelines in a Bitbucket repository.

By default, this shows the most recent pipelines. Use the --status flag to filter
by pipeline status (PENDING, IN_PROGRESS, COMPLETED, FAILED, etc.).`,
		Example: `  # List recent pipelines
  bb pipeline list

  # List failed pipelines
  bb pipeline list --status FAILED

  # List pipelines for a specific branch
  bb pipeline list --branch main

  # List with a specific limit
  bb pipeline list --limit 10

  # Output as JSON
  bb pipeline list --json

  # List pipelines for a specific repository
  bb pipeline list --repo workspace/repo`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Status, "status", "s", "", "Filter by status: PENDING, IN_PROGRESS, COMPLETED, FAILED, STOPPED, EXPIRED")
	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "Filter by branch name")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of pipelines to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
	// Get API client
	client, err := cmdutil.GetAPIClient()
	if err != nil {
		return err
	}

	// Parse repository
	workspace, repoSlug, err := cmdutil.ParseRepository(opts.Repo)
	if err != nil {
		return err
	}

	// Build list options
	listOpts := &api.PipelineListOptions{
		Sort: "-created_on", // Sort by newest first
	}

	if opts.Status != "" {
		listOpts.Status = opts.Status
	}

	// Fetch pipelines
	result, err := client.ListPipelines(ctx, workspace, repoSlug, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	// Filter by branch if specified (client-side filter since API may not support it directly)
	var pipelines []api.Pipeline
	for _, p := range result.Values {
		if opts.Branch != "" {
			if p.Target == nil || p.Target.RefName != opts.Branch {
				continue
			}
		}
		pipelines = append(pipelines, p)
		if len(pipelines) >= opts.Limit {
			break
		}
	}

	if len(pipelines) == 0 {
		if opts.Status != "" || opts.Branch != "" {
			opts.Streams.Info("No pipelines found matching the specified filters in %s/%s", workspace, repoSlug)
		} else {
			opts.Streams.Info("No pipelines found in %s/%s", workspace, repoSlug)
		}
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, pipelines)
	}

	return outputListTable(opts.Streams, pipelines)
}

func outputListJSON(streams *iostreams.IOStreams, pipelines []api.Pipeline) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(pipelines))
	for i, p := range pipelines {
		state := ""
		result := ""
		if p.State != nil {
			state = p.State.Name
			if p.State.Result != nil {
				result = p.State.Result.Name
			}
		}

		branch := ""
		commit := ""
		if p.Target != nil {
			branch = p.Target.RefName
			if p.Target.Commit != nil {
				commit = p.Target.Commit.Hash
			}
		}

		trigger := ""
		if p.Trigger != nil {
			trigger = getTriggerType(p.Trigger)
		}

		output[i] = map[string]interface{}{
			"build_number": p.BuildNumber,
			"uuid":         p.UUID,
			"state":        state,
			"result":       result,
			"branch":       branch,
			"commit":       commit,
			"trigger":      trigger,
			"created_on":   p.CreatedOn,
			"completed_on": p.CompletedOn,
			"duration":     p.BuildSecondsUsed,
		}
	}

	return cmdutil.PrintJSON(streams, output)
}

func outputListTable(streams *iostreams.IOStreams, pipelines []api.Pipeline) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "#\tSTATUS\tBRANCH\tCOMMIT\tTRIGGER\tDURATION\tSTARTED"
	cmdutil.PrintTableHeader(streams, w, header)

	// Print rows
	for _, p := range pipelines {
		buildNum := fmt.Sprintf("%d", p.BuildNumber)
		status := formatPipelineState(streams, p.State)

		branch := "-"
		commit := "-"
		if p.Target != nil {
			branch = cmdutil.TruncateString(p.Target.RefName, 25)
			if p.Target.Commit != nil {
				commit = getCommitShort(p.Target.Commit.Hash)
			}
		}

		trigger := getTriggerType(p.Trigger)
		duration := formatDuration(p.BuildSecondsUsed)
		started := cmdutil.TimeAgo(p.CreatedOn)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			buildNum, status, branch, commit, trigger, duration, started)
	}

	return w.Flush()
}

// calculateDuration calculates the duration from created to completed time
func calculateDuration(created time.Time, completed *time.Time) string {
	if completed == nil || completed.IsZero() {
		return "-"
	}
	duration := completed.Sub(created)
	return formatDuration(int(duration.Seconds()))
}
