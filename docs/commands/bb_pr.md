# bb pr

Work with Bitbucket pull requests.

## Synopsis

```
bb pr <subcommand> [flags]
```

## Subcommands

| Command | Description |
|---------|-------------|
| [list](#bb-pr-list) | List pull requests |
| [view](#bb-pr-view) | View pull request details |
| [create](#bb-pr-create) | Create a pull request |
| [merge](#bb-pr-merge) | Merge a pull request |
| [checkout](#bb-pr-checkout) | Checkout a pull request locally |
| [close](#bb-pr-close) | Decline/close a pull request |
| [reopen](#bb-pr-reopen) | Reopen a declined pull request |
| [edit](#bb-pr-edit) | Edit a pull request |
| [review](#bb-pr-review) | Review a pull request |
| [comment](#bb-pr-comment) | Add a comment to a pull request |
| [diff](#bb-pr-diff) | View pull request diff |
| [checks](#bb-pr-checks) | View CI/CD status for a pull request |

---

## bb pr list

List pull requests in the current repository.

### Synopsis

```
bb pr list [flags]
```

### Description

Lists pull requests from the current Bitbucket repository. By default, shows open pull requests. Use flags to filter by state, author, or reviewer.

### Flags

| Flag | Description |
|------|-------------|
| `--state <state>` | Filter by state: `open`, `merged`, `declined`, `all` (default: `open`) |
| `--author <username>` | Filter by author username |
| `--reviewer <username>` | Filter by reviewer username |
| `--limit <n>` | Maximum number of results to return |
| `--json` | Output in JSON format |

### Examples

```bash
# List open pull requests
bb pr list

# List all merged pull requests
bb pr list --state merged

# List PRs authored by a specific user
bb pr list --author johndoe

# List PRs where you are a reviewer
bb pr list --reviewer janedoe

# Combine filters
bb pr list --state open --author johndoe --limit 10
```

### See also

- [bb pr view](#bb-pr-view)
- [bb pr create](#bb-pr-create)

---

## bb pr view

View pull request details.

### Synopsis

```
bb pr view <number> [flags]
```

### Description

Displays detailed information about a pull request, including title, description, author, reviewers, approval status, and build status.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--web` | Open the pull request in a web browser |
| `--json` | Output in JSON format |

### Examples

```bash
# View pull request #42
bb pr view 42

# Open PR in browser
bb pr view 42 --web

# Get PR details as JSON
bb pr view 42 --json
```

### See also

- [bb pr list](#bb-pr-list)
- [bb pr diff](#bb-pr-diff)

---

## bb pr create

Create a new pull request.

### Synopsis

```
bb pr create [flags]
```

### Description

Creates a new pull request from the current branch (or specified head branch) to the target base branch. If `--title` is not provided, opens an editor to compose the PR title and description.

### Flags

| Flag | Description |
|------|-------------|
| `--title <string>` | Pull request title |
| `--body <string>` | Pull request description |
| `--base <branch>` | Base branch to merge into (default: repository default branch) |
| `--head <branch>` | Head branch containing changes (default: current branch) |
| `--draft` | Create as a draft pull request |
| `--reviewer <username>` | Add reviewer (can be repeated) |
| `--close-source-branch` | Delete source branch after merge |
| `--web` | Open the created PR in a web browser |

### Examples

```bash
# Create PR with title and body
bb pr create --title "Add new feature" --body "This PR adds..."

# Create PR from feature branch to main
bb pr create --head feature/login --base main --title "Login feature"

# Create draft PR
bb pr create --title "WIP: New feature" --draft

# Create PR with reviewers
bb pr create --title "Bug fix" --reviewer alice --reviewer bob

# Create PR and open in browser
bb pr create --title "Quick fix" --web
```

### See also

- [bb pr edit](#bb-pr-edit)
- [bb pr merge](#bb-pr-merge)

---

## bb pr merge

Merge a pull request.

### Synopsis

```
bb pr merge <number> [flags]
```

### Description

Merges an approved pull request into its target branch. Requires the PR to be approved and all required checks to pass (if configured).

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--merge-strategy <strategy>` | Merge strategy: `merge-commit`, `squash`, `fast-forward` (default: `merge-commit`) |
| `--delete-branch` | Delete the source branch after merging |
| `--message <string>` | Custom merge commit message |

### Examples

```bash
# Merge PR with default settings
bb pr merge 42

# Squash merge and delete branch
bb pr merge 42 --merge-strategy squash --delete-branch

# Fast-forward merge
bb pr merge 42 --merge-strategy fast-forward

# Merge with custom commit message
bb pr merge 42 --message "Merge feature X into main"
```

### See also

- [bb pr view](#bb-pr-view)
- [bb pr checks](#bb-pr-checks)

---

## bb pr checkout

Checkout a pull request locally.

### Synopsis

```
bb pr checkout <number> [flags]
```

### Description

Fetches and checks out a pull request branch locally for testing or review. Creates a local branch tracking the PR's source branch.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--branch <name>` | Local branch name to create (default: `pr-<number>`) |
| `--force` | Overwrite existing local branch |

### Examples

```bash
# Checkout PR #42
bb pr checkout 42

# Checkout with custom branch name
bb pr checkout 42 --branch review-feature-x

# Force overwrite existing branch
bb pr checkout 42 --force
```

### See also

- [bb pr view](#bb-pr-view)
- [bb pr diff](#bb-pr-diff)

---

## bb pr close

Decline/close a pull request.

### Synopsis

```
bb pr close <number> [flags]
```

### Description

Declines (closes) a pull request without merging. The PR can be reopened later using `bb pr reopen`.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--comment <string>` | Add a comment explaining why the PR is being declined |

### Examples

```bash
# Close PR #42
bb pr close 42

# Close with explanation
bb pr close 42 --comment "Superseded by PR #45"
```

### See also

- [bb pr reopen](#bb-pr-reopen)
- [bb pr view](#bb-pr-view)

---

## bb pr reopen

Reopen a declined pull request.

### Synopsis

```
bb pr reopen <number> [flags]
```

### Description

Reopens a previously declined pull request, returning it to an open state.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--comment <string>` | Add a comment when reopening |

### Examples

```bash
# Reopen PR #42
bb pr reopen 42

# Reopen with comment
bb pr reopen 42 --comment "Ready for re-review after addressing feedback"
```

### See also

- [bb pr close](#bb-pr-close)
- [bb pr view](#bb-pr-view)

---

## bb pr edit

Edit a pull request.

### Synopsis

```
bb pr edit <number> [flags]
```

### Description

Modifies an existing pull request's title, description, or target branch. At least one flag must be provided.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--title <string>` | New pull request title |
| `--body <string>` | New pull request description |
| `--base <branch>` | Change the target base branch |
| `--add-reviewer <username>` | Add a reviewer (can be repeated) |
| `--remove-reviewer <username>` | Remove a reviewer (can be repeated) |

### Examples

```bash
# Update PR title
bb pr edit 42 --title "Updated: Add new feature"

# Update description
bb pr edit 42 --body "New description with more details"

# Change target branch
bb pr edit 42 --base develop

# Add reviewers
bb pr edit 42 --add-reviewer alice --add-reviewer bob

# Combined edits
bb pr edit 42 --title "New title" --body "New body" --add-reviewer charlie
```

### See also

- [bb pr create](#bb-pr-create)
- [bb pr view](#bb-pr-view)

---

## bb pr review

Review a pull request.

### Synopsis

```
bb pr review <number> [flags]
```

### Description

Submit a review for a pull request. You can approve the PR or request changes.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--approve` | Approve the pull request |
| `--request-changes` | Request changes to the pull request |
| `--unapprove` | Remove your approval |
| `--comment <string>` | Add a review comment |

### Examples

```bash
# Approve PR
bb pr review 42 --approve

# Request changes with comment
bb pr review 42 --request-changes --comment "Please fix the failing tests"

# Approve with comment
bb pr review 42 --approve --comment "LGTM!"

# Remove approval
bb pr review 42 --unapprove
```

### See also

- [bb pr view](#bb-pr-view)
- [bb pr comment](#bb-pr-comment)

---

## bb pr comment

Add a comment to a pull request.

### Synopsis

```
bb pr comment <number> [flags]
```

### Description

Adds a general comment to a pull request. For inline code comments, use the web interface.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--body <string>` | Comment text (required, or opens editor if not provided) |

### Examples

```bash
# Add a comment
bb pr comment 42 --body "Great work on this feature!"

# Opens editor if --body not provided
bb pr comment 42
```

### See also

- [bb pr view](#bb-pr-view)
- [bb pr review](#bb-pr-review)

---

## bb pr diff

View pull request diff.

### Synopsis

```
bb pr diff <number> [flags]
```

### Description

Displays the diff of changes in a pull request. Shows all file changes between the source and destination branches.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--stat` | Show diffstat instead of full diff |
| `--name-only` | Show only names of changed files |
| `--color` | Force colored output |
| `--no-color` | Disable colored output |

### Examples

```bash
# View full diff
bb pr diff 42

# View diff statistics
bb pr diff 42 --stat

# List changed files only
bb pr diff 42 --name-only
```

### See also

- [bb pr view](#bb-pr-view)
- [bb pr checkout](#bb-pr-checkout)

---

## bb pr checks

View CI/CD status for a pull request.

### Synopsis

```
bb pr checks <number> [flags]
```

### Description

Displays the status of all CI/CD pipelines and checks associated with a pull request. Shows build status, test results, and other configured checks.

### Arguments

| Argument | Description |
|----------|-------------|
| `<number>` | Pull request ID (required) |

### Flags

| Flag | Description |
|------|-------------|
| `--watch` | Watch for status changes (updates every 10 seconds) |
| `--fail-fast` | Exit with error code if any check fails |
| `--json` | Output in JSON format |

### Examples

```bash
# View check status
bb pr checks 42

# Watch checks until completion
bb pr checks 42 --watch

# Check status for CI scripts
bb pr checks 42 --fail-fast

# Get checks as JSON
bb pr checks 42 --json
```

### See also

- [bb pr view](#bb-pr-view)
- [bb pr merge](#bb-pr-merge)

---

## See also

- [bb repo](bb_repo.md) - Work with repositories
- [bb issue](bb_issue.md) - Work with issues
- [bb pipeline](bb_pipeline.md) - Work with pipelines
