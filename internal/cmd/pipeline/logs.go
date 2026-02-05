package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

// LogsOptions holds the options for the logs command
type LogsOptions struct {
	Streams *iostreams.IOStreams
	Repo    string
	Step    string // Step UUID or step number (1-indexed)
}

// NewCmdLogs creates the logs command
func NewCmdLogs(streams *iostreams.IOStreams) *cobra.Command {
	opts := &LogsOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "logs <pipeline-number-or-uuid>",
		Short: "View pipeline step logs",
		Long: `View the logs for a pipeline step.

By default, if a step failed, shows that step's logs. Otherwise shows the 
last step's logs. Use --step to specify a particular step by number or UUID.

Step numbers can be obtained from 'bb pipeline steps'.`,
		Example: `  # View logs for pipeline #42 (auto-selects relevant step)
  bb pipeline logs 42

  # View logs for a specific step by number
  bb pipeline logs 42 --step 2

  # View logs for a specific step by UUID
  bb pipeline logs 42 --step "{step-uuid}"

  # View logs for a specific repository
  bb pipeline logs 42 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(cmd.Context(), opts, args[0])
		},
	}

	cmd.Flags().StringVarP(&opts.Step, "step", "s", "", "Step UUID or step number (default: first failed step or last step)")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runLogs(ctx context.Context, opts *LogsOptions, pipelineArg string) error {
	// Get API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Parse repository
	workspace, repoSlug, err := parseRepository(opts.Repo)
	if err != nil {
		return err
	}

	// Resolve pipeline UUID from build number or UUID
	pipelineUUID, err := resolvePipelineUUID(ctx, client, workspace, repoSlug, pipelineArg)
	if err != nil {
		return err
	}

	// Set timeout for API calls
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Fetch pipeline steps to determine which step to show logs for
	stepsResult, err := client.ListPipelineSteps(ctx, workspace, repoSlug, pipelineUUID)
	if err != nil {
		return fmt.Errorf("failed to list pipeline steps: %w", err)
	}

	if len(stepsResult.Values) == 0 {
		return fmt.Errorf("no steps found for pipeline %s", pipelineArg)
	}

	// Determine which step to get logs for
	stepUUID, err := resolveStepUUID(stepsResult.Values, opts.Step)
	if err != nil {
		return err
	}

	// Fetch the step logs
	logContent, err := client.GetPipelineStepLog(ctx, workspace, repoSlug, pipelineUUID, stepUUID)
	if err != nil {
		return fmt.Errorf("failed to get step logs: %w", err)
	}

	// Output raw log content
	fmt.Fprint(opts.Streams.Out, logContent)

	return nil
}

// resolveStepUUID resolves a step selector to a step UUID
// If no selector is provided, returns the first failed step or the last step
func resolveStepUUID(steps []api.PipelineStep, selector string) (string, error) {
	if len(steps) == 0 {
		return "", fmt.Errorf("no steps available")
	}

	// If a selector is provided, use it
	if selector != "" {
		return resolveStepSelector(steps, selector)
	}

	// No selector provided - find first failed step or use last step
	for _, step := range steps {
		if step.State != nil && step.State.Result != nil {
			if step.State.Result.Name == "FAILED" || step.State.Result.Name == "ERROR" {
				return step.UUID, nil
			}
		}
	}

	// No failed step found, return last step
	return steps[len(steps)-1].UUID, nil
}

// resolveStepSelector resolves a step selector (number or UUID) to a step UUID
func resolveStepSelector(steps []api.PipelineStep, selector string) (string, error) {
	// Try to parse as step number (1-indexed)
	if stepNum, err := strconv.Atoi(selector); err == nil {
		if stepNum < 1 || stepNum > len(steps) {
			return "", fmt.Errorf("step %d not found (pipeline has %d steps)", stepNum, len(steps))
		}
		return steps[stepNum-1].UUID, nil
	}

	// Try to match as UUID
	for _, step := range steps {
		// Handle various UUID formats
		if step.UUID == selector ||
			step.UUID == "{"+selector+"}" ||
			"{"+step.UUID+"}" == selector {
			return step.UUID, nil
		}
	}

	return "", fmt.Errorf("step %q not found", selector)
}
