package cmdutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/git"
	"github.com/spf13/cobra"
)

// completionTimeout is the maximum time allowed for completion API calls.
const completionTimeout = 5 * time.Second

// StaticFlagCompletion returns a completion function compatible with
// cobra.RegisterFlagCompletionFunc. It filters values by the toComplete
// prefix (case-insensitive) and always returns ShellCompDirectiveNoFileComp.
func StaticFlagCompletion(values []string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return filterPrefix(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// completionCtx returns a context with the completion timeout.
func completionCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), completionTimeout)
}

// completionClient returns an authenticated API client for completions.
// Returns nil on any error (completions must never crash).
func completionClient() *api.Client {
	client, err := GetAPIClient()
	if err != nil {
		return nil
	}
	return client
}

// completionRepo resolves the workspace and repo slug from the --repo flag
// or the current git remote. Returns empty strings on failure.
func completionRepo(cmd *cobra.Command) (workspace, repoSlug string) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	if repoFlag != "" {
		ws, slug, err := ParseRepository(repoFlag)
		if err != nil {
			return "", ""
		}
		return ws, slug
	}

	remote, err := git.GetDefaultRemote()
	if err != nil {
		return "", ""
	}
	return remote.Workspace, remote.RepoSlug
}

// completionWorkspace resolves the workspace from the --workspace flag,
// then default config, then git remote. Returns empty string on failure.
func completionWorkspace(cmd *cobra.Command) string {
	// Try --workspace flag first
	ws, _ := cmd.Flags().GetString("workspace")
	if ws != "" {
		return ws
	}

	// Try default workspace from config
	ws, err := config.GetDefaultWorkspace()
	if err == nil && ws != "" {
		return ws
	}

	// Try git remote
	remote, err := git.GetDefaultRemote()
	if err != nil {
		return ""
	}
	return remote.Workspace
}

// CompleteWorkspaceNames provides completion for workspace names.
func CompleteWorkspaceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListWorkspaces(ctx, &api.WorkspaceListOptions{Limit: 50})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, m := range result.Values {
		if m.Workspace != nil {
			names = append(names, m.Workspace.Slug)
		}
	}

	return filterPrefix(names, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// CompleteRepoNames provides completion for repository names in workspace/repo format.
func CompleteRepoNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ws := completionWorkspace(cmd)
	if ws == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListRepositories(ctx, ws, &api.RepositoryListOptions{Limit: 50})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, repo := range result.Values {
		names = append(names, fmt.Sprintf("%s/%s", ws, repo.Slug))
	}

	return filterPrefix(names, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// CompleteBranchNames provides completion for branch names.
func CompleteBranchNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ws, slug := completionRepo(cmd)
	if ws == "" || slug == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListBranches(ctx, ws, slug, &api.BranchListOptions{Limit: 50})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, b := range result.Values {
		names = append(names, b.Name)
	}

	return filterPrefix(names, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// CompletePRNumbers provides completion for pull request numbers with title descriptions.
func CompletePRNumbers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ws, slug := completionRepo(cmd)
	if ws == "" || slug == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListPullRequests(ctx, ws, slug, &api.PRListOptions{State: api.PRStateOpen, Limit: 30})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, pr := range result.Values {
		completions = append(completions, fmt.Sprintf("%d\t%s", pr.ID, pr.Title))
	}

	return filterPrefix(completions, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// CompleteIssueIDs provides completion for issue IDs with title descriptions.
func CompleteIssueIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ws, slug := completionRepo(cmd)
	if ws == "" || slug == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListIssues(ctx, ws, slug, &api.IssueListOptions{Limit: 30})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, issue := range result.Values {
		completions = append(completions, fmt.Sprintf("%d\t%s", issue.ID, issue.Title))
	}

	return filterPrefix(completions, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// CompleteWorkspaceMembers provides completion for workspace member nicknames.
func CompleteWorkspaceMembers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ws := completionWorkspace(cmd)
	if ws == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListWorkspaceMembers(ctx, ws, &api.WorkspaceMemberListOptions{Limit: 50})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, m := range result.Values {
		if m.User != nil {
			name := m.User.Nickname
			if name == "" {
				name = m.User.DisplayName
			}
			if name != "" {
				names = append(names, name)
			}
		}
	}

	return filterPrefix(names, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// filterPrefix filters values by the toComplete prefix (case-insensitive).
// For tab-separated values ("id\tdescription"), only the part before the tab is matched.
func filterPrefix(values []string, toComplete string) []string {
	if toComplete == "" {
		return values
	}

	prefix := strings.ToLower(toComplete)
	var filtered []string
	for _, v := range values {
		// For tab-separated values, match only the key part
		matchPart := v
		if idx := strings.Index(v, "\t"); idx >= 0 {
			matchPart = v[:idx]
		}
		if strings.HasPrefix(strings.ToLower(matchPart), prefix) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
