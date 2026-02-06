package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// StepsOptions holds the options for the steps command
type StepsOptions struct {
	Streams *iostreams.IOStreams
	Repo    string
	JSON    bool
}

// NewCmdSteps creates the steps command
func NewCmdSteps(streams *iostreams.IOStreams) *cobra.Command {
	opts := &StepsOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "steps <pipeline-number-or-uuid>",
		Short: "List pipeline steps",
		Long: `List the steps of a pipeline run.

Displays all steps in a pipeline with their name, status, and duration.
The step number can be used with the 'bb pipeline logs' command to view 
that step's logs.`,
		Example: `  # List steps for pipeline #42
  bb pipeline steps 42

  # List steps for pipeline by UUID
  bb pipeline steps "{12345678-1234-1234-1234-123456789012}"

  # Output as JSON
  bb pipeline steps 42 --json

  # List steps for a specific repository
  bb pipeline steps 42 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSteps(cmd.Context(), opts, args[0])
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runSteps(ctx context.Context, opts *StepsOptions, pipelineArg string) error {
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

	// Resolve pipeline UUID from build number or UUID
	pipelineUUID, err := resolvePipelineUUID(ctx, client, workspace, repoSlug, pipelineArg)
	if err != nil {
		return err
	}

	// Set timeout for API call
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Fetch pipeline steps
	result, err := client.ListPipelineSteps(ctx, workspace, repoSlug, pipelineUUID)
	if err != nil {
		return fmt.Errorf("failed to list pipeline steps: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No steps found for pipeline %s", pipelineArg)
		return nil
	}

	// Output results
	if opts.JSON {
		return outputStepsJSON(opts.Streams, result.Values)
	}

	return outputStepsTable(opts.Streams, result.Values)
}

func outputStepsJSON(streams *iostreams.IOStreams, steps []api.PipelineStep) error {
	output := make([]map[string]interface{}, len(steps))
	for i, step := range steps {
		state := ""
		result := ""
		if step.State != nil {
			state = step.State.Name
			if step.State.Result != nil {
				result = step.State.Result.Name
			}
		}

		duration := calculateStepDuration(step.StartedOn, step.CompletedOn)

		output[i] = map[string]interface{}{
			"number":       i + 1,
			"uuid":         step.UUID,
			"name":         step.Name,
			"state":        state,
			"result":       result,
			"started_on":   step.StartedOn,
			"completed_on": step.CompletedOn,
			"duration":     duration,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputStepsTable(streams *iostreams.IOStreams, steps []api.PipelineStep) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "#\tNAME\tSTATUS\tDURATION"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	// Print rows
	for i, step := range steps {
		stepNum := fmt.Sprintf("%d", i+1)
		name := step.Name
		if name == "" {
			name = "(unnamed)"
		}
		name = cmdutil.TruncateString(name, 40)
		status := formatStepStatus(streams, step.State)
		duration := formatStepDuration(step.StartedOn, step.CompletedOn)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", stepNum, name, status, duration)
	}

	return w.Flush()
}

// formatStepStatus formats step status with color
func formatStepStatus(streams *iostreams.IOStreams, state *api.PipelineStepState) string {
	if state == nil {
		return "UNKNOWN"
	}

	status := state.Name
	if state.Result != nil {
		status = state.Result.Name
	}

	if !streams.ColorEnabled() {
		return status
	}

	switch status {
	case "SUCCESSFUL":
		return iostreams.Green + status + iostreams.Reset
	case "FAILED":
		return iostreams.Red + status + iostreams.Reset
	case "IN_PROGRESS", "RUNNING":
		return iostreams.Cyan + status + iostreams.Reset
	case "PENDING":
		return iostreams.Yellow + status + iostreams.Reset
	case "STOPPED":
		return iostreams.Magenta + status + iostreams.Reset
	default:
		return status
	}
}

// formatStepDuration formats a duration between two times
func formatStepDuration(start, end *time.Time) string {
	if start == nil {
		return "-"
	}

	var duration time.Duration
	if end != nil {
		duration = end.Sub(*start)
	} else {
		duration = time.Since(*start)
	}

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		secs := int(duration.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}

	hours := int(duration.Hours())
	mins := int(duration.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// calculateStepDuration calculates the duration in seconds between two times
func calculateStepDuration(start, end *time.Time) int {
	if start == nil {
		return 0
	}

	var duration time.Duration
	if end != nil {
		duration = end.Sub(*start)
	} else {
		duration = time.Since(*start)
	}

	return int(duration.Seconds())
}
