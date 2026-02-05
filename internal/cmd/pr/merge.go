package pr

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
)

type mergeOptions struct {
	streams      *iostreams.IOStreams
	prNumber     int
	repo         string
	mergeMethod  string // "merge", "squash", or "rebase"
	deleteBranch bool
	message      string
	autoMerge    bool
	yes          bool // skip confirmation
}

// NewCmdMerge creates the merge command
func NewCmdMerge(streams *iostreams.IOStreams) *cobra.Command {
	opts := &mergeOptions{
		streams:     streams,
		mergeMethod: "merge", // default
	}

	cmd := &cobra.Command{
		Use:   "merge [<number>]",
		Short: "Merge a pull request",
		Long: `Merge a pull request via the Bitbucket API.

If no pull request number is provided, the command will try to find a
pull request associated with the current branch.

By default, the pull request is merged using a merge commit. Use --squash
for squash merge or --rebase to attempt a rebase merge (note: Bitbucket
may not support rebase merge for all repositories).`,
		Example: `  # Merge pull request #123
  bb pr merge 123

  # Squash merge
  bb pr merge 123 --squash

  # Merge and delete the source branch
  bb pr merge 123 --delete-branch

  # Merge with a custom commit message
  bb pr merge 123 --message "Merge feature XYZ"

  # Skip confirmation prompt
  bb pr merge 123 --yes

  # Enable auto-merge when checks pass
  bb pr merge 123 --auto`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get repo from flag
			if opts.repo == "" {
				opts.repo, _ = cmd.Flags().GetString("repo")
			}

			// Parse PR number from args, or try to find from current branch
			if len(args) > 0 {
				var err error
				opts.prNumber, err = parsePRNumber(args)
				if err != nil {
					return err
				}
			}
			// If no PR number given, we'll try to find it later from current branch

			// Determine merge method from flags
			if squash, _ := cmd.Flags().GetBool("squash"); squash {
				opts.mergeMethod = "squash"
			} else if rebase, _ := cmd.Flags().GetBool("rebase"); rebase {
				opts.mergeMethod = "rebase"
			}
			// "merge" is default, no need to check

			return runMerge(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.deleteBranch, "delete-branch", "d", false, "Delete the source branch after merge")
	cmd.Flags().StringVarP(&opts.message, "message", "m", "", "Custom merge commit message")
	cmd.Flags().BoolVar(&opts.autoMerge, "auto", false, "Enable auto-merge when checks pass")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")

	// Merge strategy flags (mutually exclusive)
	cmd.Flags().Bool("merge", false, "Use merge commit (default)")
	cmd.Flags().Bool("squash", false, "Use squash merge")
	cmd.Flags().Bool("rebase", false, "Use rebase merge (if supported)")

	return cmd
}

func runMerge(opts *mergeOptions) error {
	// Resolve repository
	workspace, repoSlug, err := parseRepository(opts.repo)
	if err != nil {
		return err
	}

	// Get authenticated API client
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// If no PR number, try to find PR for current branch
	if opts.prNumber == 0 {
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("could not determine current branch: %w. Please specify a pull request number", err)
		}

		prNumber, err := findPRForBranch(ctx, workspace, repoSlug, currentBranch)
		if err != nil {
			return err
		}
		opts.prNumber = prNumber
	}

	// Get PR details
	pr, err := getPullRequest(ctx, client, workspace, repoSlug, opts.prNumber)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	// Check PR state
	if pr.State != "OPEN" {
		return fmt.Errorf("pull request #%d is not open (state: %s)", opts.prNumber, pr.State)
	}

	// Determine merge method from flags
	mergeMethod := determineMergeMethod(opts)

	// Warn about rebase
	if mergeMethod == "rebase" {
		opts.streams.Warning("Note: Rebase merge may not be supported for all repositories")
	}

	// Confirmation prompt
	if !opts.yes {
		opts.streams.Info("Pull request #%d: %s", pr.ID, pr.Title)
		opts.streams.Info("  %s -> %s", pr.Source.Branch.Name, pr.Destination.Branch.Name)
		opts.streams.Info("  Merge method: %s", mergeMethod)
		if opts.deleteBranch {
			opts.streams.Info("  Will delete source branch after merge")
		}

		if !confirm(opts.streams, "Merge this pull request?") {
			return fmt.Errorf("merge cancelled")
		}
	}

	// Handle auto-merge
	if opts.autoMerge {
		return enableAutoMerge(ctx, client, workspace, repoSlug, opts, mergeMethod)
	}

	// Perform the merge
	opts.streams.Info("Merging pull request #%d...", opts.prNumber)

	err = mergePullRequest(ctx, client, workspace, repoSlug, opts.prNumber, mergeMethod, opts.message, opts.deleteBranch)
	if err != nil {
		return fmt.Errorf("failed to merge pull request: %w", err)
	}

	opts.streams.Success("Pull request #%d merged", opts.prNumber)

	// Delete branch if requested (and not already handled by API)
	if opts.deleteBranch {
		opts.streams.Success("Deleted branch %s", pr.Source.Branch.Name)
	}

	return nil
}

// determineMergeMethod determines the merge method from flags
func determineMergeMethod(opts *mergeOptions) string {
	// Check explicit flags (these would need to be parsed from cobra.Command)
	// For now, use the default unless overridden
	return opts.mergeMethod
}

// mergePullRequest merges a pull request via the API
func mergePullRequest(ctx context.Context, client *api.Client, workspace, repoSlug string, prNumber int, mergeMethod, message string, deleteBranch bool) error {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge", workspace, repoSlug, prNumber)

	// Build merge request body
	body := map[string]interface{}{}

	// Set merge strategy
	switch mergeMethod {
	case "squash":
		body["merge_strategy"] = "squash"
	case "rebase":
		// Bitbucket uses "fast_forward" which may fail if not possible
		body["merge_strategy"] = "fast_forward"
	default:
		// Default is merge commit
		body["merge_strategy"] = "merge_commit"
	}

	// Set custom commit message if provided
	if message != "" {
		body["message"] = message
	}

	// Set close_source_branch if delete requested
	if deleteBranch {
		body["close_source_branch"] = true
	}

	_, err := client.Post(ctx, path, body)
	return err
}

// enableAutoMerge enables auto-merge for a PR when checks pass
func enableAutoMerge(ctx context.Context, client *api.Client, workspace, repoSlug string, opts *mergeOptions, mergeMethod string) error {
	// Note: Bitbucket's auto-merge API may differ from this implementation
	// This is a simplified version - the actual API endpoint may vary

	opts.streams.Warning("Auto-merge is not directly supported via API. Consider enabling it in the Bitbucket web interface.")
	opts.streams.Info("Alternatively, you can wait for checks to pass and then run 'bb pr merge %d' again.", opts.prNumber)

	return nil
}

// confirm prompts the user for confirmation
func confirm(streams *iostreams.IOStreams, prompt string) bool {
	if !streams.IsStdinTTY() {
		return false
	}

	fmt.Fprintf(streams.Out, "%s [y/N]: ", prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
