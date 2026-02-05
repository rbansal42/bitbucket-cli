# AGENTS.md - Guide for LLMs and Autonomous Agents

This document provides guidance for LLMs (Claude, GPT-4, etc.) and autonomous agents (Devin, OpenHands, Claude Code) to effectively use the `bb` CLI for Bitbucket Cloud operations.

## Core Concepts

### What is `bb`?

`bb` is an unofficial command-line interface for Bitbucket Cloud, similar to GitHub's `gh` CLI. It allows you to interact with Bitbucket repositories, pull requests, issues, pipelines, and more from the terminal.

### Resource Hierarchy

Bitbucket resources follow this hierarchy:

```
Workspace (e.g., "hudle")
├── Project (e.g., "PROJ")
│   └── Repository (e.g., "backend-api")
│       ├── Pull Requests
│       ├── Issues
│       ├── Pipelines
│       └── Branches
└── Snippets (workspace-level code snippets)
```

Understanding this hierarchy is essential:
- **Workspace**: Top-level container (like a GitHub organization)
- **Project**: Optional grouping of repositories within a workspace
- **Repository**: A git repository, always belongs to a workspace

### Repository Identification

Repositories are identified in two ways:

1. **Explicit**: `workspace/repo-name` format
   ```bash
   bb pr list --repo hudle/backend-api
   ```

2. **Implicit**: Detected from current git directory's remote
   ```bash
   cd ~/projects/backend-api
   bb pr list  # auto-detects from git remote
   ```

3. **Default workspace**: If a default workspace is configured, you can use just the repo name
   ```bash
   bb workspace set-default hudle
   bb pr list --repo backend-api  # resolves to hudle/backend-api
   ```

### Output Formats

Most commands support two output formats:

- **Human-readable** (default): Formatted tables for terminal display
- **JSON** (`--json` flag): Structured output for programmatic parsing

For programmatic use, always prefer `--json` output:
```bash
bb pr list --json
bb repo list --workspace myworkspace --json
```

---

## Authentication

### Checking Authentication Status

Before performing any operations, verify authentication:

```bash
bb auth status
```

Expected output when authenticated:
```
bitbucket.org
✓ Logged in to bitbucket.org account username (keyring)
  - Active account: true
  - Git operations protocol: ssh
  - Token: ATAT****...****
```

If not authenticated, the output will indicate no active account.

### Authentication Methods

#### Method 1: API Token (Recommended for CI/CD)

Atlassian API tokens are the simplest method, especially for automation:

```bash
bb auth login
# Select "API Token" when prompted
# Enter your Atlassian account email
# Paste your API token from https://id.atlassian.com/manage-profile/security/api-tokens
```

#### Method 2: OAuth (Interactive)

OAuth is browser-based and supports automatic token refresh:

```bash
bb auth login
# Select "OAuth" when prompted
# Complete browser-based authorization
```

### Environment Variables for CI/CD

For automated environments, set credentials via environment variables:

| Variable | Description |
|----------|-------------|
| `BB_TOKEN` | API token or OAuth access token |
| `BB_USERNAME` | Atlassian account email (for API token auth) |

Example CI/CD setup:
```bash
export BB_USERNAME="ci-bot@company.com"
export BB_TOKEN="ATATT3xFfGF0..."
bb pr list --repo myworkspace/myrepo
```

### Authentication Precedence

The CLI checks for credentials in this order:
1. Environment variables (`BB_TOKEN`, `BB_USERNAME`)
2. System keyring (stored by `bb auth login`)
3. Config file credentials

### Re-authentication

If authentication expires or becomes invalid:

```bash
bb auth logout
bb auth login
```

---

## Command Reference by Category

### Workspace Commands

```bash
# List workspaces you have access to
bb workspace list

# View workspace details
bb workspace view <workspace>

# List workspace members
bb workspace members <workspace>

# Set default workspace (eliminates need for --workspace flag)
bb workspace set-default <workspace>

# Show current default workspace
bb workspace set-default
```

### Repository Commands

```bash
# List repositories in a workspace
bb repo list --workspace <workspace>
bb repo list  # uses default workspace

# View repository details
bb repo view <workspace/repo>
bb repo view  # uses current git directory

# Clone a repository
bb repo clone <workspace/repo> [directory]

# Create a new repository
bb repo create --name <name> --workspace <workspace>
bb repo create --name <name>  # uses default workspace

# Delete a repository (requires confirmation)
bb repo delete <workspace/repo>

# Fork a repository
bb repo fork <workspace/repo> --workspace <destination-workspace>

# Sync fork with upstream
bb repo sync
```

### Pull Request Commands

```bash
# List pull requests
bb pr list --repo <workspace/repo>
bb pr list --state open|merged|declined|superseded
bb pr list  # uses current git directory

# View pull request details
bb pr view <number> --repo <workspace/repo>
bb pr view <number>  # uses current git directory

# Create a pull request
bb pr create --title "Title" --source <branch> --destination <branch>
bb pr create  # interactive mode

# Edit a pull request
bb pr edit <number> --title "New Title"

# Merge a pull request
bb pr merge <number>
bb pr merge <number> --strategy squash|merge-commit|fast-forward

# Close/decline a pull request
bb pr close <number>

# Reopen a declined pull request
bb pr reopen <number>

# Checkout PR branch locally
bb pr checkout <number>

# View PR diff
bb pr diff <number>

# Check PR pipeline status
bb pr checks <number>

# Add comment to PR
bb pr comment <number> --body "Comment text"

# Review a PR (approve/request-changes/unapprove)
bb pr review <number> --approve
bb pr review <number> --request-changes --body "Please fix..."
```

### Pipeline Commands

```bash
# List pipelines
bb pipeline list --repo <workspace/repo>

# View pipeline details
bb pipeline view <pipeline-id>

# Run a pipeline
bb pipeline run --branch <branch>
bb pipeline run --branch <branch> --custom <pipeline-name>

# Stop a running pipeline
bb pipeline stop <pipeline-id>

# View pipeline step details
bb pipeline steps <pipeline-id>

# View pipeline logs
bb pipeline logs <pipeline-id>
bb pipeline logs <pipeline-id> --step <step-uuid>
```

### Issue Commands

```bash
# List issues
bb issue list --repo <workspace/repo>
bb issue list --state open|closed|new|on-hold

# View issue details
bb issue view <issue-id>

# Create an issue
bb issue create --title "Title" --kind bug|enhancement|proposal|task
bb issue create --title "Title" --priority trivial|minor|major|critical|blocker

# Edit an issue
bb issue edit <issue-id> --title "New Title"
bb issue edit <issue-id> --assignee <username>

# Close an issue
bb issue close <issue-id>

# Reopen an issue
bb issue reopen <issue-id>

# Delete an issue
bb issue delete <issue-id>

# Comment on an issue
bb issue comment <issue-id> --body "Comment text"
```

### Branch Commands

```bash
# List branches
bb branch list --repo <workspace/repo>

# Create a branch
bb branch create <branch-name> --target <base-branch>

# Delete a branch
bb branch delete <branch-name>
```

### Project Commands

```bash
# List projects in a workspace
bb project list --workspace <workspace>

# View project details
bb project view <project-key> --workspace <workspace>

# Create a project
bb project create --key <KEY> --name "Project Name" --workspace <workspace>
```

### Snippet Commands

```bash
# List snippets
bb snippet list --workspace <workspace>

# View a snippet
bb snippet view <snippet-id> --workspace <workspace>

# Create a snippet
bb snippet create --title "Title" --file <path> --workspace <workspace>

# Edit a snippet
bb snippet edit <snippet-id> --title "New Title" --workspace <workspace>

# Delete a snippet
bb snippet delete <snippet-id> --workspace <workspace>
```

### Utility Commands

```bash
# Open repository in browser
bb browse
bb browse --repo <workspace/repo>

# Raw API requests
bb api <endpoint>
bb api /repositories/workspace/repo
bb api /user

# Configuration
bb config list
bb config get <key>
bb config set <key> <value>
```

---

## Common Errors and Recovery

### Authentication Errors

#### Error: "not logged in"
```
Error: not logged in to any Bitbucket account
```
**Cause:** No authentication credentials found.
**Recovery:**
```bash
bb auth login
```

#### Error: "Token is invalid, expired, or not supported"
```
API error 401: Token is invalid, expired, or not supported for this endpoint.
```
**Cause:** Authentication token has expired or is invalid.
**Recovery:**
```bash
bb auth logout
bb auth login
```

#### Error: "Authentication failed"
```
Error: authentication failed
```
**Cause:** Invalid credentials or token.
**Recovery:**
1. Verify your API token is correct
2. Check if token has been revoked at https://id.atlassian.com/manage-profile/security/api-tokens
3. Re-authenticate with `bb auth login`

### Permission Errors

#### Error: "You do not have access to view this workspace"
```
API error 403: You do not have access to view this workspace.
```
**Cause:** Insufficient permissions for the requested operation. This often occurs when trying to create/delete resources in a workspace where you only have read access.
**Recovery:**
1. Check your role in the workspace:
   ```bash
   bb workspace list
   ```
2. If you're a "member", you may need "contributor" or "admin" role for write operations
3. Contact a workspace admin to request elevated permissions

#### Error: "Repository not found" (but it exists)
```
API error 404: Repository not found
```
**Cause:** You don't have access to this private repository, or the workspace/repo name is incorrect.
**Recovery:**
1. Verify the repository name and workspace:
   ```bash
   bb repo list --workspace <workspace>
   ```
2. Check if you have access to the repository
3. Verify spelling of workspace and repository names

### Repository Resolution Errors

#### Error: "invalid repository format"
```
Error: invalid repository format: myrepo (expected workspace/repo)
```
**Cause:** Repository specified without workspace, and no default workspace is configured.
**Recovery:**
```bash
# Option 1: Use full format
bb pr list --repo workspace/myrepo

# Option 2: Set a default workspace
bb workspace set-default <workspace>
bb pr list --repo myrepo
```

#### Error: "could not detect repository"
```
Error: could not detect repository: not a git repository
```
**Cause:** Command run outside a git repository and no `--repo` flag provided.
**Recovery:**
```bash
# Option 1: Specify repository explicitly
bb pr list --repo workspace/repo

# Option 2: Navigate to a git repository
cd /path/to/repo
bb pr list
```

### Rate Limiting

#### Error: "Rate limit exceeded"
```
API error 429: Rate limit exceeded
```
**Cause:** Too many API requests in a short period.
**Recovery:**
1. Wait a few minutes before retrying
2. Reduce frequency of API calls
3. Use `--json` output and cache results when possible

### Pipeline Errors

#### Error: "Pipeline not found"
```
API error 404: Pipeline not found
```
**Cause:** The pipeline ID doesn't exist or has been deleted.
**Recovery:**
```bash
# List available pipelines
bb pipeline list --repo workspace/repo
```

#### Error: "Cannot run pipeline"
```
Error: cannot run pipeline: no bitbucket-pipelines.yml found
```
**Cause:** Repository doesn't have pipelines configured.
**Recovery:**
1. Ensure `bitbucket-pipelines.yml` exists in the repository root
2. Verify pipelines are enabled in repository settings

---

## Best Practices for Agents

### 1. Always Verify Authentication First

Before performing any operations, check authentication status:
```bash
bb auth status
```

If not authenticated, handle gracefully by informing the user or attempting to authenticate if credentials are available.

### 2. Use Explicit Repository References

While auto-detection from git remotes is convenient, explicit references are more reliable:
```bash
# Prefer explicit
bb pr list --repo workspace/repo

# Over implicit (may fail if not in git directory)
bb pr list
```

### 3. Set Default Workspace for Repeated Operations

When performing multiple operations on the same workspace:
```bash
bb workspace set-default myworkspace
# Subsequent commands don't need --workspace
bb repo list
bb project list
```

### 4. Use JSON Output for Parsing

Always use `--json` flag when you need to process command output:
```bash
bb pr list --json
bb repo view workspace/repo --json
```

### 5. Check Before Destructive Operations

Before delete or merge operations, verify the resource exists:
```bash
# Check PR exists and is open before merging
bb pr view 123 --json

# Then merge
bb pr merge 123
```

### 6. Handle Errors Gracefully

Parse error messages to determine the appropriate recovery action:
- 401 errors: Re-authenticate
- 403 errors: Permission issue - inform user
- 404 errors: Resource not found - verify names
- 429 errors: Rate limited - wait and retry

### 7. Use Appropriate Flags for Non-Interactive Mode

For automated scripts and CI/CD, use flags that avoid interactive prompts:
```bash
# Force flag for destructive operations
bb repo delete workspace/repo --force
bb snippet delete abc123 --workspace ws --force

# Specify all required information via flags
bb pr create --title "Title" --source feature --destination main --body "Description"
```

### 8. Respect API Rate Limits

Bitbucket has API rate limits. When making multiple requests:
- Add small delays between requests if making many calls
- Cache responses when appropriate
- Use list commands with `--limit` to reduce data transfer

### 9. Validate Inputs Before Commands

Before running commands, validate that required information is available:
- Workspace name is valid
- Repository exists
- PR/Issue numbers are numeric
- Branch names are valid

### 10. Provide Clear Feedback

When using `bb` as part of a larger workflow, capture and relay command output to provide clear feedback about what operations were performed and their results.

---

## Configuration Reference

### Config File Location

```
~/.config/bb/config.yml
```

### Available Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `git_protocol` | Protocol for git operations (`ssh` or `https`) | `ssh` |
| `editor` | Editor for interactive input | `$EDITOR` or `vim` |
| `prompt` | Enable/disable prompts | `enabled` |
| `pager` | Pager for long output | `$PAGER` or `less` |
| `browser` | Browser for web commands | system default |
| `http_timeout` | API request timeout (seconds) | `30` |
| `default_workspace` | Default workspace for commands | none |

### Setting Configuration

```bash
bb config set git_protocol https
bb config set default_workspace myworkspace
bb config get default_workspace
bb config list
```

---

## Version and Help

```bash
# Show version
bb --version

# Show help
bb --help
bb <command> --help
bb <command> <subcommand> --help
```
