# bb - Bitbucket CLI

An unofficial command-line interface for Bitbucket Cloud, inspired by GitHub's `gh` CLI.

[![CI](https://github.com/rbansal42/bitbucket-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/rbansal42/bitbucket-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/rbansal42/bitbucket-cli)](https://github.com/rbansal42/bitbucket-cli/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Pull Requests**: Create, list, view, merge, review, and manage PRs
- **Repositories**: Clone, create, fork, and manage repositories
- **Issues**: Create, list, view, and manage issue tracker issues
- **Pipelines**: Trigger, monitor, and view CI/CD pipeline logs
- **Branches**: Create, list, and delete branches
- **Workspaces & Projects**: Browse and manage Bitbucket workspaces and projects
- **Snippets**: Create and manage code snippets
- **Authentication**: Secure OAuth and access token support
- **Shell Completions**: Tab completion for Bash, Zsh, Fish, and PowerShell

## Installation

### Homebrew (macOS and Linux)

```bash
brew install rbansal42/tap/bb
```

### Download Binary

Download the latest release from the [releases page](https://github.com/rbansal42/bitbucket-cli/releases).

Available for:
- macOS (Intel and Apple Silicon)
- Linux (amd64, arm64)
- Windows (amd64)

### Build from Source

```bash
go install github.com/rbansal42/bitbucket-cli/cmd/bb@latest
```

Or clone and build:

```bash
git clone https://github.com/rbansal42/bitbucket-cli.git
cd bb
go build -o bb ./cmd/bb
```

## Quick Start

### 1. Authenticate with Bitbucket

```bash
bb auth login
```

This will guide you through OAuth authentication or access token setup.

### 2. Clone a Repository

```bash
bb repo clone myworkspace/myrepo
```

### 3. Work with Pull Requests

```bash
# List open PRs in current repo
bb pr list

# Create a new PR
bb pr create --title "Add new feature" --base main

# View PR details
bb pr view 123

# Checkout a PR locally
bb pr checkout 123

# Merge a PR
bb pr merge 123
```

## Common Workflows

### Pull Request Workflow

```bash
# Create a feature branch and make changes
git checkout -b feature/my-feature
# ... make changes ...
git commit -am "Add my feature"
git push -u origin feature/my-feature

# Create a pull request
bb pr create --title "Add my feature" --body "Description of changes"

# After review, merge the PR
bb pr merge 123 --merge-strategy squash --delete-branch
```

### Code Review Workflow

```bash
# List PRs assigned to you for review
bb pr list --reviewer @me

# View PR details and diff
bb pr view 123
bb pr diff 123

# Approve or request changes
bb pr review 123 --approve
bb pr review 123 --request-changes --body "Please fix the tests"

# Add a comment
bb pr comment 123 --body "Looks good, just one suggestion..."
```

### CI/CD Pipeline Workflow

```bash
# Trigger a pipeline on current branch
bb pipeline run

# Trigger a pipeline on a specific branch
bb pipeline run --branch develop

# List recent pipelines
bb pipeline list

# View pipeline details
bb pipeline view <pipeline-uuid>

# Watch pipeline logs
bb pipeline logs <pipeline-uuid>

# Stop a running pipeline
bb pipeline stop <pipeline-uuid>
```

### Issue Tracking Workflow

```bash
# Create an issue
bb issue create --title "Bug: Login fails" --kind bug --priority critical

# List open issues
bb issue list --state open

# View and comment on an issue
bb issue view 42
bb issue comment 42 --body "I can reproduce this on v2.0"

# Close an issue
bb issue close 42
```

### Repository Management

```bash
# List your repositories
bb repo list

# Create a new repository
bb repo create --name my-new-repo --private --description "My project"

# Fork a repository
bb repo fork otherworkspace/cool-project

# View repository details
bb repo view

# Open repository in browser
bb browse
```

## Commands Reference

### Authentication
| Command | Description |
|---------|-------------|
| `bb auth login` | Authenticate with Bitbucket |
| `bb auth logout` | Log out of Bitbucket |
| `bb auth status` | View authentication status |

### Pull Requests
| Command | Description |
|---------|-------------|
| `bb pr list` | List pull requests |
| `bb pr view <number>` | View a pull request |
| `bb pr create` | Create a pull request |
| `bb pr merge <number>` | Merge a pull request |
| `bb pr checkout <number>` | Checkout a PR branch locally |
| `bb pr close <number>` | Decline/close a pull request |
| `bb pr reopen <number>` | Reopen a declined pull request |
| `bb pr edit <number>` | Edit PR title, description, or base |
| `bb pr review <number>` | Add a review (approve/request-changes) |
| `bb pr comment <number>` | Add a comment to a PR |
| `bb pr diff <number>` | View pull request diff |
| `bb pr checks <number>` | View CI/CD status checks |

### Repositories
| Command | Description |
|---------|-------------|
| `bb repo list` | List repositories |
| `bb repo view` | View repository details |
| `bb repo clone <repo>` | Clone a repository |
| `bb repo create` | Create a new repository |
| `bb repo fork <repo>` | Fork a repository |
| `bb repo delete <repo>` | Delete a repository |
| `bb repo sync` | Sync fork with upstream |
| `bb repo set-default` | Set default repository for current directory |

### Issues
| Command | Description |
|---------|-------------|
| `bb issue list` | List issues |
| `bb issue view <id>` | View an issue |
| `bb issue create` | Create an issue |
| `bb issue edit <id>` | Edit an issue |
| `bb issue close <id>` | Close/resolve an issue |
| `bb issue reopen <id>` | Reopen an issue |
| `bb issue comment <id>` | Add a comment to an issue |
| `bb issue delete <id>` | Delete an issue |

### Pipelines
| Command | Description |
|---------|-------------|
| `bb pipeline list` | List pipelines |
| `bb pipeline view <uuid>` | View pipeline details |
| `bb pipeline run` | Trigger a pipeline |
| `bb pipeline logs <uuid>` | View pipeline logs |
| `bb pipeline steps <uuid>` | View pipeline steps |
| `bb pipeline stop <uuid>` | Stop a running pipeline |

### Branches
| Command | Description |
|---------|-------------|
| `bb branch list` | List branches |
| `bb branch create <name>` | Create a branch |
| `bb branch delete <name>` | Delete a branch |

### Workspaces
| Command | Description |
|---------|-------------|
| `bb workspace list` | List workspaces |
| `bb workspace view <slug>` | View workspace details |
| `bb workspace members <slug>` | List workspace members |

### Projects
| Command | Description |
|---------|-------------|
| `bb project list` | List projects |
| `bb project view <key>` | View project details |
| `bb project create` | Create a project |

### Snippets
| Command | Description |
|---------|-------------|
| `bb snippet list` | List snippets |
| `bb snippet view <id>` | View a snippet |
| `bb snippet create` | Create a snippet |
| `bb snippet edit <id>` | Edit a snippet |
| `bb snippet delete <id>` | Delete a snippet |

### Other Commands
| Command | Description |
|---------|-------------|
| `bb browse` | Open repository in browser |
| `bb api <endpoint>` | Make raw API requests |
| `bb config get/set` | Manage configuration |
| `bb completion <shell>` | Generate shell completions |

## Shell Completion

Enable tab completion for your shell:

### Bash

```bash
# Linux
bb completion bash | sudo tee /etc/bash_completion.d/bb > /dev/null

# macOS (with Homebrew bash-completion)
bb completion bash > $(brew --prefix)/etc/bash_completion.d/bb
```

### Zsh

```bash
# If shell completion is not already enabled, add this to ~/.zshrc:
# autoload -Uz compinit && compinit

bb completion zsh > "${fpath[1]}/_bb"
```

### Fish

```bash
bb completion fish > ~/.config/fish/completions/bb.fish
```

### PowerShell

```powershell
bb completion powershell | Out-String | Invoke-Expression

# To load on startup, add to your profile:
# bb completion powershell | Out-String | Invoke-Expression
```

## Configuration

Configuration files are stored in `~/.config/bb/` (or `$XDG_CONFIG_HOME/bb/` on Linux).

### Files

| File | Description |
|------|-------------|
| `hosts.yml` | Authentication tokens per host |
| `config.yml` | General settings |

### Settings

```bash
# Set preferred git protocol (https or ssh)
bb config set git_protocol ssh

# Set default editor
bb config set editor vim

# View current configuration
bb config get git_protocol
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `BB_TOKEN` | Override authentication token |
| `BITBUCKET_TOKEN` | Alternative token variable |
| `BB_REPO` | Override repository (workspace/repo) |
| `NO_COLOR` | Disable colored output |

## Comparison with gh CLI

`bb` is designed to feel familiar to `gh` CLI users:

| gh command | bb equivalent |
|------------|---------------|
| `gh auth login` | `bb auth login` |
| `gh pr list` | `bb pr list` |
| `gh pr create` | `bb pr create` |
| `gh repo clone` | `bb repo clone` |
| `gh issue create` | `bb issue create` |
| `gh run list` | `bb pipeline list` |
| `gh api` | `bb api` |

### Key Differences

- **Workspaces**: Bitbucket uses workspaces instead of organizations
- **Pipelines**: Bitbucket Pipelines vs GitHub Actions
- **Issue Tracker**: Bitbucket's built-in issue tracker (when enabled)
- **Snippets**: Bitbucket snippets vs GitHub Gists

## Documentation

- [Authentication Guide](docs/guide/authentication.md)
- [Configuration Guide](docs/guide/configuration.md)
- [Scripting & Automation](docs/guide/scripting.md)
- [Troubleshooting](docs/guide/troubleshooting.md)
- [Command Reference](docs/commands/)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Disclaimer

This is an unofficial CLI tool and is not affiliated with or endorsed by Atlassian or Bitbucket.
