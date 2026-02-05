package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

type stopOptions struct {
	streams     *iostreams.IOStreams
	pipelineArg string
	yes         bool
	repo        string
}

// NewCmdStop creates the stop command
func NewCmdStop(streams *iostreams.IOStreams) *cobra.Command {
	opts := &stopOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "stop <pipeline-number-or-uuid>",
		Short: "Stop a running pipeline",
		Long: `Stop a running pipeline by its build number or UUID.

You will be prompted to confirm the stop action unless the --yes flag is provided.`,
		Example: `  # Stop a pipeline by build number
  bb pipeline stop 42

  # Stop a pipeline by UUID
  bb pipeline stop {12345678-1234-1234-1234-123456789012}

  # Stop without confirmation prompt
  bb pipeline stop 42 --yes

  # Stop a pipeline in a different repository
  bb pipeline stop 42 --repo myworkspace/myrepo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.pipelineArg = args[0]
			return runPipelineStop(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	return cmd
}

func runPipelineStop(opts *stopOptions) error {
	// Resolve repository
	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	// Parse the pipeline argument - could be a build number or UUID
	pipelineUUID, buildNumber, err := parsePipelineStopArg(opts.pipelineArg)
	if err != nil {
		return err
	}

	// Get authenticated client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// If we have a build number, we need to get the UUID
	// Bitbucket API accepts build number directly as the identifier
	pipelineID := pipelineUUID
	if pipelineUUID == "" && buildNumber > 0 {
		// Use the build number directly - the Bitbucket API accepts it
		pipelineID = fmt.Sprintf("%d", buildNumber)
	}

	// Confirmation prompt
	if !opts.yes {
		// Require TTY for interactive confirmation
		if !opts.streams.IsStdinTTY() {
			return fmt.Errorf("cannot confirm stop: stdin is not a terminal\nUse --yes flag to skip confirmation in non-interactive mode")
		}

		displayID := opts.pipelineArg
		if buildNumber > 0 {
			displayID = fmt.Sprintf("#%d", buildNumber)
		}

		fmt.Fprintf(opts.streams.Out, "Are you sure you want to stop pipeline %s? [y/N] ", displayID)

		if !confirmStop(opts.streams.In) {
			return fmt.Errorf("stop cancelled")
		}
	}

	// Stop the pipeline
	if err := client.StopPipeline(ctx, workspace, repoSlug, pipelineID); err != nil {
		return fmt.Errorf("failed to stop pipeline: %w", err)
	}

	// Print success
	if buildNumber > 0 {
		opts.streams.Success("Stopped pipeline #%d", buildNumber)
	} else {
		opts.streams.Success("Stopped pipeline %s", pipelineUUID)
	}

	return nil
}

// parsePipelineStopArg parses the pipeline argument, returning either a UUID or build number
func parsePipelineStopArg(arg string) (uuid string, buildNumber int, err error) {
	// Check if it looks like a UUID (contains curly braces or dashes in UUID format)
	if strings.HasPrefix(arg, "{") || strings.Contains(arg, "-") {
		// Treat as UUID
		// Normalize: ensure curly braces
		if !strings.HasPrefix(arg, "{") {
			arg = "{" + arg
		}
		if !strings.HasSuffix(arg, "}") {
			arg = arg + "}"
		}
		return arg, 0, nil
	}

	// Try to parse as build number
	num, err := strconv.Atoi(arg)
	if err != nil {
		return "", 0, fmt.Errorf("invalid pipeline identifier: %s (expected build number or UUID)", arg)
	}
	if num <= 0 {
		return "", 0, fmt.Errorf("invalid pipeline build number: must be a positive integer")
	}

	return "", num, nil
}

// confirmStop prompts the user to confirm stop operation
func confirmStop(in interface{}) bool {
	var reader *bufio.Reader

	// Handle different input types
	switch r := in.(type) {
	case *bufio.Reader:
		reader = r
	case io.Reader:
		reader = bufio.NewReader(r)
	default:
		return false
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
