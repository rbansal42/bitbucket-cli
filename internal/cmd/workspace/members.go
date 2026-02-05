package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

// MembersOptions holds the options for the members command
type MembersOptions struct {
	WorkspaceSlug string
	Limit         int
	JSON          bool
	Streams       *iostreams.IOStreams
}

// NewCmdMembers creates the workspace members command
func NewCmdMembers(streams *iostreams.IOStreams) *cobra.Command {
	opts := &MembersOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "members <workspace>",
		Short: "List workspace members",
		Long: `List all members of a Bitbucket workspace.

Shows the username, display name, and role of each member.`,
		Example: `  # List members of a workspace
  bb workspace members myworkspace

  # List with a specific limit
  bb workspace members myworkspace --limit 50

  # Output as JSON
  bb workspace members myworkspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.WorkspaceSlug = args[0]
			return runMembers(cmd.Context(), opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of members to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	return cmd
}

func runMembers(ctx context.Context, opts *MembersOptions) error {
	// Get API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build list options
	listOpts := &api.WorkspaceMemberListOptions{
		Limit: opts.Limit,
	}

	// Fetch members
	result, err := client.ListWorkspaceMembers(ctx, opts.WorkspaceSlug, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list workspace members: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No members found in workspace %s", opts.WorkspaceSlug)
		return nil
	}

	// Output results
	if opts.JSON {
		return outputMembersJSON(opts.Streams, result.Values)
	}

	return outputMembersTable(opts.Streams, result.Values)
}

func outputMembersJSON(streams *iostreams.IOStreams, members []api.WorkspaceMember) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(members))
	for i, m := range members {
		output[i] = map[string]interface{}{
			"role": m.Permission,
		}
		if m.User != nil {
			output[i]["username"] = m.User.Username
			output[i]["display_name"] = m.User.DisplayName
			output[i]["uuid"] = m.User.UUID
			output[i]["account_id"] = m.User.AccountID
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputMembersTable(streams *iostreams.IOStreams, members []api.WorkspaceMember) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "USERNAME\tNAME\tROLE"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	// Print rows
	for _, m := range members {
		username := ""
		displayName := ""
		if m.User != nil {
			username = m.User.Username
			displayName = m.User.DisplayName
		}
		role := formatMemberRole(streams, m.Permission)
		fmt.Fprintf(w, "%s\t%s\t%s\n", username, displayName, role)
	}

	return w.Flush()
}

func formatMemberRole(streams *iostreams.IOStreams, role string) string {
	if !streams.ColorEnabled() {
		return role
	}

	switch role {
	case "owner":
		return iostreams.Yellow + role + iostreams.Reset
	case "collaborator":
		return iostreams.Cyan + role + iostreams.Reset
	default:
		return role
	}
}
