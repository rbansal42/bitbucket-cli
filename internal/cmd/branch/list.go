package branch

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
	Repo    string
	Limit   int
	JSON    bool
	Streams *iostreams.IOStreams
}

// NewCmdList creates the branch list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &ListOptions{
		Streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branches in a repository",
		Long: `List branches in a Bitbucket repository.

By default, this command detects the repository from your git remote.
Use the --repo flag to specify a different repository.`,
		Example: `  # List branches in the current repository
  bb branch list

  # List branches in a specific repository
  bb branch list --repo myworkspace/myrepo

  # Limit the number of branches shown
  bb branch list --limit 10

  # Output as JSON
  bb branch list --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository in WORKSPACE/REPO format (detects from git remote if not specified)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "l", 30, "Maximum number of branches to list")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output in JSON format")

	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
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

	// Build list options
	listOpts := &api.BranchListOptions{
		Limit: opts.Limit,
	}

	// Fetch branches
	result, err := client.ListBranches(ctx, workspace, repoSlug, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(result.Values) == 0 {
		opts.Streams.Info("No branches found in %s/%s", workspace, repoSlug)
		return nil
	}

	// Output results
	if opts.JSON {
		return outputListJSON(opts.Streams, result.Values)
	}

	return outputTable(opts.Streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, branches []api.BranchFull) error {
	// Create simplified JSON output
	output := make([]map[string]interface{}, len(branches))
	for i, branch := range branches {
		item := map[string]interface{}{
			"name": branch.Name,
		}
		if branch.Target != nil {
			item["commit"] = branch.Target.Hash
			item["message"] = branch.Target.Message
		}
		output[i] = item
	}

	return cmdutil.PrintJSON(streams, output)
}

func outputTable(streams *iostreams.IOStreams, branches []api.BranchFull) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	// Print header
	header := "NAME\tCOMMIT\tMESSAGE"
	cmdutil.PrintTableHeader(streams, w, header)

	// Print rows
	for _, branch := range branches {
		name := branch.Name
		commit := ""
		message := ""

		if branch.Target != nil {
			// First 7 characters of commit hash
			if len(branch.Target.Hash) >= 7 {
				commit = branch.Target.Hash[:7]
			} else {
				commit = branch.Target.Hash
			}
			// Truncate message to 50 chars and replace newlines
			message = cmdutil.TruncateString(branch.Target.Message, 50)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", name, commit, message)
	}

	return w.Flush()
}
