# bb - Bitbucket CLI Design Document

**Date:** 2026-02-05  
**Status:** Approved  
**Target:** Bitbucket Cloud API 2.0

## Overview

`bb` is an unofficial command-line interface for Bitbucket Cloud, designed to provide feature parity with GitHub's `gh` CLI. It enables developers to work seamlessly with Bitbucket from the terminal.

## Decisions

| Decision | Choice |
|----------|--------|
| Target Platform | Bitbucket Cloud |
| Scope | Full `gh` CLI parity |
| Language | Go |
| CLI Name | `bb` |
| Authentication | OAuth 2.0 + Access Tokens |
| Configuration | `~/.config/bb/` + system keychain |
| Distribution | Homebrew + go install + GitHub Releases |

## Project Structure

```
bb/
├── cmd/
│   └── bb/
│       └── main.go              # Entry point
├── internal/
│   ├── api/                     # Bitbucket API client
│   │   ├── client.go            # HTTP client, auth handling
│   │   ├── pullrequests.go      # PR endpoints
│   │   ├── repositories.go      # Repo endpoints
│   │   ├── issues.go            # Issue endpoints
│   │   ├── pipelines.go         # Pipeline endpoints
│   │   └── ...
│   ├── cmd/                     # Command implementations
│   │   ├── root.go              # Root command
│   │   ├── auth/                # bb auth
│   │   ├── pr/                  # bb pr
│   │   ├── repo/                # bb repo
│   │   ├── issue/               # bb issue
│   │   ├── pipeline/            # bb pipeline
│   │   └── ...
│   ├── config/                  # Configuration management
│   │   ├── config.go            # Read/write ~/.config/bb/
│   │   └── keyring.go           # System keychain integration
│   ├── browser/                 # Open URLs in browser
│   ├── git/                     # Git operations
│   └── iostreams/               # Terminal I/O, colors, formatting
├── pkg/                         # Public packages (if any)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Key Libraries

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/zalando/go-keyring` - System keychain
- `golang.org/x/oauth2` - OAuth 2.0 flow

## Command Mapping (gh → bb)

### Core Commands

| gh | bb | Bitbucket API |
|----|-----|---------------|
| `gh auth login` | `bb auth login` | OAuth 2.0 flow |
| `gh auth status` | `bb auth status` | User endpoint |
| `gh pr create` | `bb pr create` | `POST /pullrequests` |
| `gh pr list` | `bb pr list` | `GET /pullrequests` |
| `gh pr view` | `bb pr view` | `GET /pullrequests/{id}` |
| `gh pr merge` | `bb pr merge` | `POST /pullrequests/{id}/merge` |
| `gh pr checkout` | `bb pr checkout` | Fetch PR branch + git checkout |
| `gh pr review` | `bb pr review` | `POST /pullrequests/{id}/approve` |
| `gh repo clone` | `bb repo clone` | Get clone URL + git clone |
| `gh repo create` | `bb repo create` | `POST /repositories/{workspace}/{repo}` |
| `gh repo fork` | `bb repo fork` | `POST /repositories/{workspace}/{repo}/forks` |
| `gh issue create` | `bb issue create` | `POST /issues` |
| `gh issue list` | `bb issue list` | `GET /issues` |
| `gh run list` | `bb pipeline list` | `GET /pipelines` |
| `gh run view` | `bb pipeline view` | `GET /pipelines/{uuid}` |
| `gh gist create` | `bb snippet create` | `POST /snippets` |
| `gh browse` | `bb browse` | Generate URL + open browser |

### Bitbucket-Specific Commands

- `bb workspace list` - List workspaces
- `bb project list` - List projects in workspace

## Full Command Tree

```
bb
├── auth
│   ├── login          # Authenticate with Bitbucket
│   ├── logout         # Remove authentication
│   ├── status         # Show authentication status
│   ├── token          # Print access token
│   └── refresh        # Refresh OAuth token
│
├── pr
│   ├── create         # Create a pull request
│   ├── list           # List pull requests
│   ├── view           # View pull request details
│   ├── checkout       # Check out PR branch locally
│   ├── diff           # View PR diff
│   ├── merge          # Merge a pull request
│   ├── close          # Decline a pull request
│   ├── reopen         # Reopen a declined PR
│   ├── review         # Approve / request changes / comment
│   ├── comment        # Add comment to PR
│   ├── edit           # Edit PR title/description
│   └── checks         # View pipeline status for PR
│
├── repo
│   ├── create         # Create new repository
│   ├── clone          # Clone a repository
│   ├── fork           # Fork a repository
│   ├── list           # List repositories
│   ├── view           # View repository details
│   ├── delete         # Delete a repository
│   ├── sync           # Sync fork with upstream
│   └── set-default    # Set default repo for directory
│
├── issue
│   ├── create         # Create new issue
│   ├── list           # List issues
│   ├── view           # View issue details
│   ├── close          # Close an issue
│   ├── reopen         # Reopen an issue
│   ├── comment        # Add comment to issue
│   └── edit           # Edit issue
│
├── pipeline
│   ├── list           # List pipeline runs
│   ├── view           # View pipeline details
│   ├── run            # Trigger a pipeline
│   ├── stop           # Stop a running pipeline
│   └── logs           # View pipeline logs
│
├── snippet
│   ├── create         # Create a snippet (gist)
│   ├── list           # List snippets
│   ├── view           # View snippet
│   ├── edit           # Edit snippet
│   └── delete         # Delete snippet
│
├── workspace
│   ├── list           # List workspaces
│   └── view           # View workspace details
│
├── project
│   ├── list           # List projects
│   └── view           # View project details
│
├── browse             # Open in browser
├── api                # Raw API requests
├── config             # Manage configuration
│   ├── get
│   ├── set
│   └── list
│
└── completion         # Shell completions (bash/zsh/fish)
```

## Authentication

### OAuth 2.0 Flow (Interactive)

```
$ bb auth login

1. User runs `bb auth login`
2. bb starts local HTTP server on random port (e.g., localhost:8372)
3. bb opens browser to Bitbucket OAuth authorization URL:
   https://bitbucket.org/site/oauth2/authorize?
     client_id=<BB_CLIENT_ID>
     &response_type=code
     &redirect_uri=http://localhost:8372/callback
     &scope=repository:write pullrequest:write issue:write pipeline:read ...
4. User authorizes in browser
5. Bitbucket redirects to localhost:8372/callback?code=<AUTH_CODE>
6. bb exchanges code for access_token + refresh_token
7. Tokens stored in system keychain
8. Config updated in ~/.config/bb/hosts.yml
```

### Access Token Flow (Non-interactive)

```
$ bb auth login --with-token < token.txt
# OR
$ export BB_TOKEN=<workspace_or_repo_token>
$ bb pr list  # Uses BB_TOKEN automatically
```

### Token Storage

- OAuth tokens auto-refresh when expired
- Tokens stored in system keychain (macOS Keychain, Linux Secret Service)
- Fallback to encrypted file if keychain unavailable

### Multi-account Support

```yaml
# ~/.config/bb/hosts.yml
bitbucket.org:
  users:
    rbansal42:
      # token in keychain: bb:bitbucket.org:rbansal42
    work-account:
      # token in keychain: bb:bitbucket.org:work-account
  user: rbansal42  # active user
```

## Configuration

### File Locations

- `~/.config/bb/config.yml` - General settings
- `~/.config/bb/hosts.yml` - Host/user configuration

### Configuration Precedence (highest to lowest)

1. Command-line flags (`--repo`, `--workspace`)
2. Environment variables (`BB_TOKEN`, `BB_WORKSPACE`)
3. Repository-local config (`.bb.yml` in repo root)
4. User config (`~/.config/bb/config.yml`)
5. Defaults

### Environment Variables

```
BB_TOKEN          - Access token (skips keychain)
BB_WORKSPACE      - Default workspace
BB_REPO           - Default repository
BB_NO_COLOR       - Disable colors
BB_PAGER          - Pager for long output
BB_BROWSER        - Browser for --web commands
```

## Context Detection

When running commands without specifying a repo:

1. Check for `--repo`/`-R` flag first
2. Look for git remote in current directory:
   - Parse `.git/config` for remotes
   - Match patterns:
     - `git@bitbucket.org:workspace/repo.git`
     - `https://bitbucket.org/workspace/repo.git`
3. If multiple Bitbucket remotes, prefer "origin"

## Output Formatting

```bash
# Default: human-readable tables
$ bb pr list
ID    TITLE                    BRANCH           STATUS
123   Add user authentication  feature/auth     OPEN
124   Fix login bug            bugfix/login     MERGED

# JSON output for scripting
$ bb pr list --json id,title,state
[{"id":123,"title":"Add user authentication","state":"OPEN"},...]

# Quiet mode (IDs only)
$ bb pr list -q
123
124

# Web mode (open in browser)
$ bb pr view 123 --web
```

## Error Handling

```go
// Consistent error types
type APIError struct {
    StatusCode int
    Message    string
    Detail     string
}
```

User-friendly messages with hints:
```
$ bb pr merge 123
Error: Cannot merge PR #123: has unresolved comments
Hint: Use --force to merge anyway, or resolve comments first
```

## Testing Strategy

- Unit tests for API client (mock HTTP responses)
- Integration tests against Bitbucket API (optional, needs credentials)
- E2E tests for CLI commands (capture stdout/stderr)
- Test fixtures from real API responses

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
- Project scaffolding (Go modules, Cobra setup)
- Configuration system (`~/.config/bb/`)
- Keyring integration
- OAuth 2.0 flow (`bb auth login/logout/status`)
- API client base (HTTP client, auth headers, pagination)
- Git remote detection
- `bb api` command for raw API access

### Phase 2: Pull Requests (Week 3-4)
- `bb pr list` - List PRs with filtering
- `bb pr view` - View PR details
- `bb pr create` - Create PR (interactive + flags)
- `bb pr checkout` - Fetch and checkout PR branch
- `bb pr merge` - Merge PR
- `bb pr diff` - View diff
- `bb pr comment/review` - Add feedback

### Phase 3: Repositories & Browse (Week 5)
- `bb repo clone/create/fork/list/view`
- `bb browse` - Open URLs in browser
- `bb repo set-default`

### Phase 4: Issues & Pipelines (Week 6)
- `bb issue create/list/view/close/comment`
- `bb pipeline list/view/run/logs`

### Phase 5: Extras & Polish (Week 7)
- `bb snippet` commands
- `bb workspace/project` commands
- Shell completions
- Homebrew formula
- GitHub Actions for releases
- Documentation

## Distribution

### Homebrew
```bash
brew tap user/bb
brew install bb
```

### Go Install
```bash
go install github.com/user/bb@latest
```

### Binary Releases
- GitHub Releases with pre-built binaries
- Platforms: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
