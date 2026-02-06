package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// parsePipelineIdentifier parses a pipeline build number or UUID from args
func parsePipelineIdentifier(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("pipeline build number or UUID is required")
	}

	identifier := args[0]

	// Check if it's a number (build number) or UUID
	if _, err := strconv.Atoi(identifier); err == nil {
		// It's a build number, which needs to be converted to UUID via API
		return identifier, nil
	}

	// Check if it looks like a UUID (contains hyphens or curly braces)
	if strings.Contains(identifier, "-") || strings.HasPrefix(identifier, "{") {
		// Clean up UUID format if needed
		identifier = strings.Trim(identifier, "{}")
		if !strings.HasPrefix(identifier, "{") {
			identifier = "{" + identifier + "}"
		}
		return identifier, nil
	}

	return identifier, nil
}

// formatPipelineState formats the pipeline state with appropriate color
func formatPipelineState(streams *iostreams.IOStreams, state *api.PipelineState) string {
	if state == nil {
		return "UNKNOWN"
	}

	stateName := state.Name
	resultName := ""
	if state.Result != nil {
		resultName = state.Result.Name
	}

	// Determine display text
	displayText := stateName
	if resultName != "" {
		displayText = resultName
	}

	if !streams.ColorEnabled() {
		return displayText
	}

	// Apply color based on state
	switch {
	case resultName == "SUCCESSFUL":
		return iostreams.Green + displayText + iostreams.Reset
	case resultName == "FAILED" || resultName == "ERROR":
		return iostreams.Red + displayText + iostreams.Reset
	case resultName == "STOPPED":
		return iostreams.Yellow + displayText + iostreams.Reset
	case stateName == "IN_PROGRESS":
		return iostreams.Yellow + displayText + iostreams.Reset
	case stateName == "PENDING":
		return iostreams.Cyan + displayText + iostreams.Reset
	default:
		return displayText
	}
}

// formatDuration formats a duration in a human-readable format
func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "-"
	}

	d := time.Duration(seconds) * time.Second

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// getCommitShort returns the first 7 characters of a commit hash
func getCommitShort(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

// getTriggerType returns a human-readable trigger type
func getTriggerType(trigger *api.PipelineTrigger) string {
	if trigger == nil {
		return "unknown"
	}

	switch trigger.Type {
	case "pipeline_trigger_push":
		return "push"
	case "pipeline_trigger_pull_request":
		return "pr"
	case "pipeline_trigger_manual":
		return "manual"
	case "pipeline_trigger_schedule":
		return "schedule"
	default:
		// Extract readable name from type
		t := trigger.Type
		t = strings.TrimPrefix(t, "pipeline_trigger_")
		return t
	}
}

// resolvePipelineUUID resolves a build number or UUID to a UUID
func resolvePipelineUUID(ctx context.Context, client *api.Client, workspace, repoSlug, identifier string) (string, error) {
	// Check if it's a build number
	if buildNum, err := strconv.Atoi(identifier); err == nil {
		// It's a build number, need to find the UUID
		// List recent pipelines to find matching build number
		result, err := client.ListPipelines(ctx, workspace, repoSlug, &api.PipelineListOptions{
			Sort: "-created_on",
		})
		if err != nil {
			return "", fmt.Errorf("failed to list pipelines: %w", err)
		}

		for _, p := range result.Values {
			if p.BuildNumber == buildNum {
				return p.UUID, nil
			}
		}
		return "", fmt.Errorf("pipeline #%d not found", buildNum)
	}

	// It's already a UUID, clean it up
	uuid := identifier
	// Ensure UUID has curly braces
	if len(uuid) > 0 && uuid[0] != '{' {
		uuid = "{" + uuid + "}"
	}

	return uuid, nil
}
