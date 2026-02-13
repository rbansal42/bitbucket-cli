# Dynamic Shell Completions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add dynamic shell completions so pressing Tab suggests actual values (workspace names, repo names, branch names, PR numbers, enum values) instead of only completing command/flag names.

**Architecture:** Create a shared `internal/cmdutil/completion.go` helper module with reusable completion functions. Register completions via Cobra's `RegisterFlagCompletionFunc` (for flags) and `ValidArgsFunction` (for positional args). API-backed completions silently return no suggestions on auth/network failure.

**Tech Stack:** Go, Cobra CLI (spf13/cobra) `RegisterFlagCompletionFunc` + `ValidArgsFunction` + `cobra.ShellCompDirective*`

---

## Task 1: Create the Completion Helper Module

**Files:**
- Create: `internal/cmdutil/completion.go`
- Create: `internal/cmdutil/completion_test.go`

This module provides reusable functions that commands will call to register completions.

**Step 1: Write the test file**

```go
// internal/cmdutil/completion_test.go
package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteNoFiles(t *testing.T) {
	// Verify that the noFiles directive is always included
	got := cobra.ShellCompDirectiveNoFileComp
	if got == cobra.ShellCompDirectiveDefault {
		t.Error("expected NoFileComp directive, got Default")
	}
}

func TestCompleteStaticValues(t *testing.T) {
	values := []string{"OPEN", "MERGED", "DECLINED"}
	fn := StaticFlagCompletion(values)
	completions, directive := fn(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %d", directive)
	}

	if len(completions) != 3 {
		t.Errorf("expected 3 completions, got %d", len(completions))
	}

	if completions[0] != "OPEN" || completions[1] != "MERGED" || completions[2] != "DECLINED" {
		t.Errorf("unexpected completions: %v", completions)
	}
}

func TestCompleteStaticValuesFiltering(t *testing.T) {
	values := []string{"OPEN", "MERGED", "DECLINED"}
	fn := StaticFlagCompletion(values)
	completions, _ := fn(nil, nil, "M")

	if len(completions) != 1 {
		t.Errorf("expected 1 completion, got %d: %v", len(completions), completions)
	}
	if len(completions) > 0 && completions[0] != "MERGED" {
		t.Errorf("expected MERGED, got %s", completions[0])
	}
}
```

**Step 2: Run the test to verify it fails**

```bash
go test ./internal/cmdutil/ -run TestCompleteStaticValues -v
```

Expected: FAIL - `StaticFlagCompletion` undefined.

**Step 3: Write the implementation**

```go
// internal/cmdutil/completion.go
package cmdutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/config"
	"github.com/rbansal42/bitbucket-cli/internal/git"
)

// noFileComp is a shorthand for the "don't complete filenames" directive.
var noFileComp = cobra.ShellCompDirectiveNoFileComp

// StaticFlagCompletion returns a completion function for a fixed set of values.
// It filters values by the current prefix typed by the user (toComplete).
func StaticFlagCompletion(values []string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var filtered []string
		for _, v := range values {
			if strings.HasPrefix(strings.ToUpper(v), strings.ToUpper(toComplete)) {
				filtered = append(filtered, v)
			}
		}
		return filtered, noFileComp
	}
}

// completionCtx returns a context with a short timeout suitable for completion.
func completionCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// completionClient returns an API client for use during completion.
// Returns nil if authentication is not configured (user not logged in).
func completionClient() *api.Client {
	client, err := GetAPIClient()
	if err != nil {
		return nil
	}
	return client
}

// completionRepo resolves workspace and repoSlug from the --repo flag or git remote.
// Returns empty strings if resolution fails (not in a git repo, etc.).
func completionRepo(cmd *cobra.Command) (workspace, repoSlug string) {
	repoFlag, _ := cmd.Flags().GetString("repo")
	ws, slug, err := ParseRepository(repoFlag)
	if err != nil {
		return "", ""
	}
	return ws, slug
}

// completionWorkspace resolves the workspace from the --workspace flag, default config, or git remote.
func completionWorkspace(cmd *cobra.Command) string {
	ws, _ := cmd.Flags().GetString("workspace")
	if ws != "" {
		return ws
	}
	defaultWs, err := config.GetDefaultWorkspace()
	if err == nil && defaultWs != "" {
		return defaultWs
	}
	remote, err := git.GetDefaultRemote()
	if err == nil {
		return remote.Workspace
	}
	return ""
}

// CompleteWorkspaceNames returns workspace slugs for the authenticated user.
func CompleteWorkspaceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, noFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListWorkspaces(ctx, &api.WorkspaceListOptions{Limit: 50})
	if err != nil {
		return nil, noFileComp
	}

	var names []string
	for _, ws := range result.Values {
		slug := ws.Workspace.Slug
		if strings.HasPrefix(slug, toComplete) {
			names = append(names, slug)
		}
	}
	return names, noFileComp
}

// CompleteRepoNames returns repository full names (workspace/repo) for the current workspace.
func CompleteRepoNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, noFileComp
	}

	// Determine workspace: from typed prefix, --workspace flag, default config, or git remote
	ws := ""
	if strings.Contains(toComplete, "/") {
		parts := strings.SplitN(toComplete, "/", 2)
		ws = parts[0]
		toComplete = parts[1]
	}
	if ws == "" {
		ws = completionWorkspace(cmd)
	}
	if ws == "" {
		return nil, noFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListRepositories(ctx, ws, &api.RepositoryListOptions{Limit: 50})
	if err != nil {
		return nil, noFileComp
	}

	var names []string
	for _, repo := range result.Values {
		fullName := fmt.Sprintf("%s/%s", ws, repo.Slug)
		if strings.HasPrefix(repo.Slug, toComplete) || strings.HasPrefix(fullName, toComplete) {
			names = append(names, fullName)
		}
	}
	return names, noFileComp
}

// CompleteBranchNames returns branch names for the resolved repository.
func CompleteBranchNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, noFileComp
	}

	ws, slug := completionRepo(cmd)
	if ws == "" || slug == "" {
		return nil, noFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListBranches(ctx, ws, slug, &api.BranchListOptions{Limit: 50})
	if err != nil {
		return nil, noFileComp
	}

	var names []string
	for _, branch := range result.Values {
		if strings.HasPrefix(branch.Name, toComplete) {
			names = append(names, branch.Name)
		}
	}
	return names, noFileComp
}

// CompletePRNumbers returns open PR numbers with titles as descriptions.
func CompletePRNumbers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, noFileComp
	}

	ws, slug := completionRepo(cmd)
	if ws == "" || slug == "" {
		return nil, noFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListPullRequests(ctx, ws, slug, &api.PRListOptions{
		State: api.PRStateOpen,
		Limit: 30,
	})
	if err != nil {
		return nil, noFileComp
	}

	var completions []string
	for _, pr := range result.Values {
		num := fmt.Sprintf("%d", pr.ID)
		if strings.HasPrefix(num, toComplete) {
			// Format: "123\tTitle of the PR" -- cobra uses \t to separate value from description
			completions = append(completions, fmt.Sprintf("%d\t%s", pr.ID, TruncateString(pr.Title, 50)))
		}
	}
	return completions, noFileComp
}

// CompleteIssueIDs returns issue IDs with titles as descriptions.
func CompleteIssueIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, noFileComp
	}

	ws, slug := completionRepo(cmd)
	if ws == "" || slug == "" {
		return nil, noFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListIssues(ctx, ws, slug, &api.IssueListOptions{Limit: 30})
	if err != nil {
		return nil, noFileComp
	}

	var completions []string
	for _, issue := range result.Values {
		num := fmt.Sprintf("%d", issue.ID)
		if strings.HasPrefix(num, toComplete) {
			completions = append(completions, fmt.Sprintf("%d\t%s", issue.ID, TruncateString(issue.Title, 50)))
		}
	}
	return completions, noFileComp
}

// CompleteWorkspaceMembers returns usernames of workspace members.
func CompleteWorkspaceMembers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client := completionClient()
	if client == nil {
		return nil, noFileComp
	}

	ws, _ := completionRepo(cmd)
	if ws == "" {
		ws = completionWorkspace(cmd)
	}
	if ws == "" {
		return nil, noFileComp
	}

	ctx, cancel := completionCtx()
	defer cancel()

	result, err := client.ListWorkspaceMembers(ctx, ws, &api.WorkspaceMemberListOptions{Limit: 50})
	if err != nil {
		return nil, noFileComp
	}

	var names []string
	for _, member := range result.Values {
		name := member.User.Nickname
		if name == "" {
			name = member.User.DisplayName
		}
		if strings.HasPrefix(name, toComplete) {
			names = append(names, name)
		}
	}
	return names, noFileComp
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/cmdutil/ -run TestComplete -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/cmdutil/completion.go internal/cmdutil/completion_test.go
git commit -m "feat: add shared completion helper module for dynamic shell completions"
```

---

## Task 2: Register Static Enum Completions for PR Commands

**Files:**
- Modify: `internal/cmd/pr/list.go:62` (add `--state` completion)
- Modify: `internal/cmd/pr/view.go:83-84` (add positional arg completion)
- Modify: `internal/cmd/pr/merge.go:98` (add positional arg completion)
- Modify: `internal/cmd/pr/close.go` (add positional arg completion)
- Modify: `internal/cmd/pr/edit.go` (add positional arg completion)
- Modify: `internal/cmd/pr/diff.go` (add positional arg completion)
- Modify: `internal/cmd/pr/comment.go` (add positional arg completion)
- Modify: `internal/cmd/pr/checks.go` (add positional arg completion)
- Modify: `internal/cmd/pr/review.go` (add positional arg completion)
- Modify: `internal/cmd/pr/reopen.go` (add positional arg completion)
- Modify: `internal/cmd/pr/checkout.go` (add positional arg completion)

**Step 1: Add `--state` completion to `pr list`**

In `internal/cmd/pr/list.go`, after line 66 (the `--repo` flag), add:

```go
	_ = cmd.RegisterFlagCompletionFunc("state", cmdutil.StaticFlagCompletion([]string{"OPEN", "MERGED", "DECLINED"}))
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 2: Add `ValidArgsFunction` for PR numbers to all PR commands that take `<number>` positional arg**

For each of these commands: `view`, `merge`, `close`, `edit`, `diff`, `comment`, `checks`, `review`, `reopen`, `checkout` -- set `ValidArgsFunction` on the cobra.Command. For example, in `internal/cmd/pr/merge.go` after the cmd is created (around line 92):

```go
	cmd.ValidArgsFunction = cmdutil.CompletePRNumbers
```

And in `internal/cmd/pr/view.go`, `view` accepts a PR number, URL, or branch, so:

```go
	cmd.ValidArgsFunction = cmdutil.CompletePRNumbers
```

**Step 3: Add `--base` and `--head` branch completion to `pr create`**

In `internal/cmd/pr/create.go`, after the flag definitions (after line 84):

```go
	_ = cmd.RegisterFlagCompletionFunc("base", cmdutil.CompleteBranchNames)
	_ = cmd.RegisterFlagCompletionFunc("head", cmdutil.CompleteBranchNames)
	_ = cmd.RegisterFlagCompletionFunc("reviewer", cmdutil.CompleteWorkspaceMembers)
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 4: Register `--repo` completion on all PR commands that have it**

For every PR subcommand file that defines `--repo`, add after the flag definition:

```go
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 5: Build and verify compilation**

```bash
go build ./...
```

Expected: Successful compilation.

**Step 6: Commit**

```bash
git add internal/cmd/pr/
git commit -m "feat: add dynamic completions for PR commands (state, numbers, branches, reviewers)"
```

---

## Task 3: Register Static Enum Completions for Issue Commands

**Files:**
- Modify: `internal/cmd/issue/list.go:69-71` (add `--state`, `--kind`, `--priority` completions)
- Modify: `internal/cmd/issue/create.go:61-62` (add `--kind`, `--priority` completions)
- Modify: `internal/cmd/issue/edit.go:85-86` (add `--kind`, `--priority` completions)
- Modify: `internal/cmd/issue/view.go` (add positional arg completion)
- Modify: `internal/cmd/issue/close.go` (add positional arg completion)
- Modify: `internal/cmd/issue/reopen.go` (add positional arg completion)
- Modify: `internal/cmd/issue/delete.go` (add positional arg completion)
- Modify: `internal/cmd/issue/comment.go` (add positional arg completion)

**Step 1: Add enum completions to `issue list`**

In `internal/cmd/issue/list.go`, after the flag definitions (after line 75):

```go
	_ = cmd.RegisterFlagCompletionFunc("state", cmdutil.StaticFlagCompletion([]string{
		"new", "open", "resolved", "on hold", "invalid", "duplicate", "wontfix", "closed",
	}))
	_ = cmd.RegisterFlagCompletionFunc("kind", cmdutil.StaticFlagCompletion([]string{
		"bug", "enhancement", "proposal", "task",
	}))
	_ = cmd.RegisterFlagCompletionFunc("priority", cmdutil.StaticFlagCompletion([]string{
		"trivial", "minor", "major", "critical", "blocker",
	}))
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 2: Add enum completions to `issue create`**

In `internal/cmd/issue/create.go`, after line 64:

```go
	_ = cmd.RegisterFlagCompletionFunc("kind", cmdutil.StaticFlagCompletion([]string{
		"bug", "enhancement", "proposal", "task",
	}))
	_ = cmd.RegisterFlagCompletionFunc("priority", cmdutil.StaticFlagCompletion([]string{
		"trivial", "minor", "major", "critical", "blocker",
	}))
	_ = cmd.RegisterFlagCompletionFunc("assignee", cmdutil.CompleteWorkspaceMembers)
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 3: Add enum completions to `issue edit`**

In `internal/cmd/issue/edit.go`, after line 88:

```go
	_ = cmd.RegisterFlagCompletionFunc("kind", cmdutil.StaticFlagCompletion([]string{
		"bug", "enhancement", "proposal", "task",
	}))
	_ = cmd.RegisterFlagCompletionFunc("priority", cmdutil.StaticFlagCompletion([]string{
		"trivial", "minor", "major", "critical", "blocker",
	}))
	_ = cmd.RegisterFlagCompletionFunc("assignee", cmdutil.CompleteWorkspaceMembers)
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 4: Add `ValidArgsFunction` to issue commands that take `<issue-id>` positional arg**

For `view`, `close`, `reopen`, `delete`, `comment`, `edit`:

```go
	cmd.ValidArgsFunction = cmdutil.CompleteIssueIDs
```

**Step 5: Build and verify**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add internal/cmd/issue/
git commit -m "feat: add dynamic completions for issue commands (state, kind, priority, IDs, assignees)"
```

---

## Task 4: Register Completions for Workspace-scoped Commands

**Files:**
- Modify: `internal/cmd/repo/list.go` (add `--workspace` completion)
- Modify: `internal/cmd/repo/create.go` (add `--workspace` completion)
- Modify: `internal/cmd/repo/fork.go` (add `--workspace` completion)
- Modify: `internal/cmd/project/list.go` (add `--workspace` completion)
- Modify: `internal/cmd/project/create.go` (add `--workspace` completion)
- Modify: `internal/cmd/project/view.go` (add `--workspace` completion)
- Modify: `internal/cmd/snippet/list.go` (add `--workspace` completion)
- Modify: `internal/cmd/snippet/create.go` (add `--workspace` completion)
- Modify: `internal/cmd/snippet/view.go` (add `--workspace` completion)
- Modify: `internal/cmd/snippet/edit.go` (add `--workspace` completion)
- Modify: `internal/cmd/snippet/delete.go` (add `--workspace` completion)

**Step 1: Add `--workspace` completion to all workspace-scoped commands**

In each file listed above, add after the `--workspace` flag definition:

```go
	_ = cmd.RegisterFlagCompletionFunc("workspace", cmdutil.CompleteWorkspaceNames)
```

**Step 2: Build and verify**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/cmd/repo/ internal/cmd/project/ internal/cmd/snippet/
git commit -m "feat: add workspace name completion for repo, project, and snippet commands"
```

---

## Task 5: Register Completions for Branch and Pipeline Commands

**Files:**
- Modify: `internal/cmd/branch/list.go` (add `--repo` completion)
- Modify: `internal/cmd/branch/create.go` (add `--target` branch completion, `--repo` completion)
- Modify: `internal/cmd/branch/delete.go` (add positional arg branch completion, `--repo` completion)
- Modify: `internal/cmd/pipeline/list.go` (add `--repo` completion)
- Modify: `internal/cmd/pipeline/view.go` (add `--repo` completion)
- Modify: `internal/cmd/pipeline/run.go` (add `--branch` completion, `--repo` completion)
- Modify: `internal/cmd/pipeline/stop.go` (add `--repo` completion)
- Modify: `internal/cmd/pipeline/steps.go` (add `--repo` completion)
- Modify: `internal/cmd/pipeline/logs.go` (add `--repo` completion)

**Step 1: Add completions to branch commands**

In `branch/delete.go`:
```go
	cmd.ValidArgsFunction = cmdutil.CompleteBranchNames
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

In `branch/create.go` (for `--target` flag that specifies the base branch):
```go
	_ = cmd.RegisterFlagCompletionFunc("target", cmdutil.CompleteBranchNames)
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

In `branch/list.go`:
```go
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 2: Add `--repo` completion to all pipeline commands, and `--branch` completion to `pipeline run`**

In each pipeline command file, add:
```go
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

In `pipeline/run.go`, also add:
```go
	_ = cmd.RegisterFlagCompletionFunc("branch", cmdutil.CompleteBranchNames)
```

**Step 3: Build and verify**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add internal/cmd/branch/ internal/cmd/pipeline/
git commit -m "feat: add dynamic completions for branch and pipeline commands"
```

---

## Task 6: Add `--repo` Completion to Browse Command

**Files:**
- Modify: `internal/cmd/browse/browse.go`

**Step 1: Add `--repo` completion**

After the `--repo` flag definition:
```go
	_ = cmd.RegisterFlagCompletionFunc("repo", cmdutil.CompleteRepoNames)
```

**Step 2: Build and verify**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/cmd/browse/
git commit -m "feat: add repo completion to browse command"
```

---

## Task 7: Full Build + Test + Manual Verification

**Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: All tests PASS.

**Step 2: Build the binary**

```bash
go build -o bb ./cmd/bb/
```

**Step 3: Verify static completions work**

Test that enum completions are registered correctly by using Cobra's built-in `__complete` hidden command:

```bash
# Test PR state completion
./bb __complete pr list --state ""
# Expected output includes: OPEN, MERGED, DECLINED

# Test issue kind completion
./bb __complete issue create --kind ""
# Expected output includes: bug, enhancement, proposal, task

# Test issue priority completion
./bb __complete issue list --priority ""
# Expected output includes: trivial, minor, major, critical, blocker
```

**Step 4: Verify dynamic completions fail gracefully without auth**

```bash
# If not logged in, these should return empty (no error, no crash)
./bb __complete repo list --workspace ""
# Expected: returns completion directives but no crash
```

**Step 5: Commit (if any test fixes were needed)**

```bash
git add -A
git commit -m "fix: address any issues found during completion testing"
```

---

## Summary

| Task | Scope | Type | Commands Affected |
|------|-------|------|-------------------|
| 1 | Completion helper module | New file | All (shared) |
| 2 | PR command completions | Modify | pr list/view/merge/close/edit/diff/comment/checks/review/reopen/checkout/create |
| 3 | Issue command completions | Modify | issue list/create/edit/view/close/reopen/delete/comment |
| 4 | Workspace completions | Modify | repo list/create/fork, project list/create/view, snippet list/create/view/edit/delete |
| 5 | Branch + Pipeline completions | Modify | branch list/create/delete, pipeline list/view/run/stop/steps/logs |
| 6 | Browse completion | Modify | browse |
| 7 | Full verification | Testing | All |

**Completion types implemented:**

| Completion | Function | Source |
|------------|----------|--------|
| `--state` (PR) | `StaticFlagCompletion` | Hard-coded: OPEN, MERGED, DECLINED |
| `--state` (issue) | `StaticFlagCompletion` | Hard-coded: new, open, resolved, etc. |
| `--kind` | `StaticFlagCompletion` | Hard-coded: bug, enhancement, proposal, task |
| `--priority` | `StaticFlagCompletion` | Hard-coded: trivial, minor, major, critical, blocker |
| `--workspace` | `CompleteWorkspaceNames` | API: ListWorkspaces |
| `--repo` | `CompleteRepoNames` | API: ListRepositories |
| `--base`/`--head`/`--target`/`--branch` | `CompleteBranchNames` | API: ListBranches |
| `--reviewer`/`--assignee` | `CompleteWorkspaceMembers` | API: ListWorkspaceMembers |
| `<pr-number>` positional | `CompletePRNumbers` | API: ListPullRequests |
| `<issue-id>` positional | `CompleteIssueIDs` | API: ListIssues |
| `<branch-name>` positional | `CompleteBranchNames` | API: ListBranches |
