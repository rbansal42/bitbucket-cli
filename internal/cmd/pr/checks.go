package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

// ChecksOptions holds the options for the checks command
type ChecksOptions struct {
	Repo    string
	PRID    int64
	JSON    bool
	Streams *iostreams.IOStreams
}

// NewCmdChecks creates the pr checks command
func NewCmdChecks(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ChecksOptions{Streams: streams}

	cmd := &cobra.Command{
		Use:   "checks <number>",
		Short: "View status checks for a pull request",
		Long: `View the status of CI/CD checks for a pull request.

Shows build statuses, pipeline results, and other commit statuses
associated with the pull request.`,
		Example: `  # View checks for PR #123
  bb pr checks 123

  # View checks with JSON output
  bb pr checks 123 --json

  # View checks for a specific repository
  bb pr checks 123 --repo workspace/repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pull request number: %s", args[0])
			}
			if id <= 0 {
				return fmt.Errorf("invalid pull request number: must be a positive integer")
			}
			opts.PRID = id
			return runChecks(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	return cmd
}

func runChecks(ctx context.Context, opts *ChecksOptions) error {
	// Parse repository
	workspace, repoSlug, err := parseRepository(opts.Repo)
	if err != nil {
		return err
	}

	// Get API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get statuses
	result, err := client.GetPullRequestStatuses(ctx, workspace, repoSlug, opts.PRID)
	if err != nil {
		return fmt.Errorf("failed to get status checks: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No status checks found for PR #%d", opts.PRID)
		return nil
	}

	// Output
	if opts.JSON {
		return outputChecksJSON(opts.Streams, result.Values)
	}

	return outputChecksTable(opts.Streams, result.Values)
}

func outputChecksJSON(streams *iostreams.IOStreams, statuses []api.CommitStatus) error {
	output := make([]map[string]interface{}, len(statuses))
	for i, s := range statuses {
		output[i] = map[string]interface{}{
			"name":        s.Name,
			"key":         s.Key,
			"state":       s.State,
			"description": s.Description,
			"url":         s.URL,
			"created_on":  s.CreatedOn,
			"updated_on":  s.UpdatedOn,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputChecksTable(streams *iostreams.IOStreams, statuses []api.CommitStatus) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Header
	header := "STATUS\tNAME\tDESCRIPTION"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	// Rows
	for _, s := range statuses {
		status := formatCheckStatus(s.State, streams.ColorEnabled())
		name := s.Name
		if name == "" {
			name = s.Key
		}
		desc := truncateString(s.Description, 50)

		fmt.Fprintf(w, "%s\t%s\t%s\n", status, name, desc)
	}

	return w.Flush()
}

// formatCheckStatus formats the check status with optional color
func formatCheckStatus(state string, color bool) string {
	// States: SUCCESSFUL, FAILED, INPROGRESS, STOPPED
	switch state {
	case "SUCCESSFUL":
		if color {
			return iostreams.Green + "✓ pass" + iostreams.Reset
		}
		return "✓ pass"
	case "FAILED":
		if color {
			return iostreams.Red + "✗ fail" + iostreams.Reset
		}
		return "✗ fail"
	case "INPROGRESS":
		if color {
			return iostreams.Yellow + "○ running" + iostreams.Reset
		}
		return "○ running"
	case "STOPPED":
		if color {
			return iostreams.White + "◌ stopped" + iostreams.Reset
		}
		return "◌ stopped"
	default:
		return state
	}
}
