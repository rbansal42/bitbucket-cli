package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/browser"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// ViewOptions holds the options for the view command
type ViewOptions struct {
	Identifier string // Pipeline build number or UUID
	Web        bool
	JSON       bool
	Repo       string
	Streams    *iostreams.IOStreams
}

// NewCmdView creates the pipeline view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ViewOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view <pipeline-number-or-uuid>",
		Short: "View a pipeline's details",
		Long: `Display the details of a specific pipeline run.

You can specify a pipeline by its build number or UUID.`,
		Example: `  # View pipeline by build number
  bb pipeline view 123

  # View pipeline by UUID
  bb pipeline view {12345678-1234-1234-1234-123456789abc}

  # Open pipeline in browser
  bb pipeline view 123 --web

  # Output as JSON
  bb pipeline view 123 --json

  # View pipeline for a specific repository
  bb pipeline view 123 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Identifier = args[0]
			return runView(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open the pipeline in a web browser")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runView(ctx context.Context, opts *ViewOptions) error {
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

	// Resolve pipeline UUID
	pipelineUUID, err := resolvePipelineUUID(ctx, client, workspace, repoSlug, opts.Identifier)
	if err != nil {
		return err
	}

	// Fetch pipeline details
	pipeline, err := client.GetPipeline(ctx, workspace, repoSlug, pipelineUUID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	// Handle --web flag
	if opts.Web {
		webURL := getPipelineWebURL(workspace, repoSlug, pipeline.BuildNumber)
		if err := browser.Open(webURL); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.Streams.Success("Opened %s in your browser", webURL)
		return nil
	}

	// Fetch steps for summary
	steps, err := client.ListPipelineSteps(ctx, workspace, repoSlug, pipelineUUID)
	if err != nil {
		// Non-fatal error, just don't show steps
		steps = nil
	}

	// Handle --json flag
	if opts.JSON {
		return outputViewJSON(opts.Streams, pipeline, steps)
	}

	// Display formatted output
	return displayPipeline(opts.Streams, pipeline, steps)
}

func getPipelineWebURL(workspace, repoSlug string, buildNumber int) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/pipelines/results/%d",
		workspace, repoSlug, buildNumber)
}

func outputViewJSON(streams *iostreams.IOStreams, pipeline *api.Pipeline, steps *api.Paginated[api.PipelineStep]) error {
	output := map[string]interface{}{
		"build_number":       pipeline.BuildNumber,
		"uuid":               pipeline.UUID,
		"created_on":         pipeline.CreatedOn,
		"completed_on":       pipeline.CompletedOn,
		"build_seconds_used": pipeline.BuildSecondsUsed,
	}

	if pipeline.State != nil {
		output["state"] = pipeline.State.Name
		if pipeline.State.Result != nil {
			output["result"] = pipeline.State.Result.Name
		}
	}

	if pipeline.Target != nil {
		target := map[string]interface{}{
			"type":     pipeline.Target.Type,
			"ref_type": pipeline.Target.RefType,
			"ref_name": pipeline.Target.RefName,
		}
		if pipeline.Target.Commit != nil {
			target["commit"] = pipeline.Target.Commit.Hash
		}
		output["target"] = target
	}

	if pipeline.Trigger != nil {
		output["trigger"] = getTriggerType(pipeline.Trigger)
	}

	if pipeline.Creator != nil {
		output["creator"] = map[string]interface{}{
			"display_name": pipeline.Creator.DisplayName,
			"username":     pipeline.Creator.Username,
		}
	}

	if steps != nil && len(steps.Values) > 0 {
		stepsOutput := make([]map[string]interface{}, len(steps.Values))
		for i, step := range steps.Values {
			stepData := map[string]interface{}{
				"uuid": step.UUID,
				"name": step.Name,
			}
			if step.State != nil {
				stepData["state"] = step.State.Name
				if step.State.Result != nil {
					stepData["result"] = step.State.Result.Name
				}
			}
			stepsOutput[i] = stepData
		}
		output["steps"] = stepsOutput
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func displayPipeline(streams *iostreams.IOStreams, pipeline *api.Pipeline, steps *api.Paginated[api.PipelineStep]) error {
	// Title: Pipeline #number
	fmt.Fprintf(streams.Out, "Pipeline #%d\n", pipeline.BuildNumber)
	fmt.Fprintln(streams.Out)

	// Status
	fmt.Fprintf(streams.Out, "Status:    %s\n", formatPipelineState(streams, pipeline.State))

	// Branch/Ref
	if pipeline.Target != nil {
		refType := pipeline.Target.RefType
		if refType == "" {
			refType = "ref"
		}
		fmt.Fprintf(streams.Out, "%s:   %s\n", capitalize(refType), pipeline.Target.RefName)

		// Commit
		if pipeline.Target.Commit != nil {
			fmt.Fprintf(streams.Out, "Commit:    %s\n", getCommitShort(pipeline.Target.Commit.Hash))
		}
	}

	// Trigger
	if pipeline.Trigger != nil {
		fmt.Fprintf(streams.Out, "Trigger:   %s\n", getTriggerType(pipeline.Trigger))
	}

	// Creator
	if pipeline.Creator != nil {
		name := pipeline.Creator.DisplayName
		if name == "" {
			name = pipeline.Creator.Username
		}
		fmt.Fprintf(streams.Out, "Creator:   %s\n", name)
	}

	// Duration
	if pipeline.BuildSecondsUsed > 0 {
		fmt.Fprintf(streams.Out, "Duration:  %s\n", formatDuration(pipeline.BuildSecondsUsed))
	}

	// Timestamps
	fmt.Fprintf(streams.Out, "Started:   %s\n", cmdutil.TimeAgo(pipeline.CreatedOn))
	if pipeline.CompletedOn != nil && !pipeline.CompletedOn.IsZero() {
		fmt.Fprintf(streams.Out, "Completed: %s\n", cmdutil.TimeAgo(*pipeline.CompletedOn))
	}

	// Steps summary
	if steps != nil && len(steps.Values) > 0 {
		fmt.Fprintln(streams.Out)
		fmt.Fprintln(streams.Out, "Steps:")
		for _, step := range steps.Values {
			stepStatus := formatStepState(streams, step.State)
			stepName := step.Name
			if stepName == "" {
				stepName = "Step"
			}
			fmt.Fprintf(streams.Out, "  %s %s\n", stepStatus, stepName)
		}
	}

	return nil
}

// formatStepState formats a pipeline step state with color
func formatStepState(streams *iostreams.IOStreams, state *api.PipelineStepState) string {
	if state == nil {
		return "[?]"
	}

	// Determine icon and color based on state
	stateName := state.Name
	resultName := ""
	if state.Result != nil {
		resultName = state.Result.Name
	}

	var icon string
	var color string

	switch {
	case resultName == "SUCCESSFUL":
		icon = "[ok]"
		color = iostreams.Green
	case resultName == "FAILED" || resultName == "ERROR":
		icon = "[x]"
		color = iostreams.Red
	case resultName == "STOPPED":
		icon = "[!]"
		color = iostreams.Yellow
	case stateName == "IN_PROGRESS":
		icon = "[>]"
		color = iostreams.Yellow
	case stateName == "PENDING":
		icon = "[ ]"
		color = iostreams.Cyan
	default:
		icon = "[?]"
		color = ""
	}

	if !streams.ColorEnabled() || color == "" {
		return icon
	}

	return color + icon + iostreams.Reset
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
