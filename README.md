# bb - Bitbucket CLI

An unofficial command-line interface for Bitbucket Cloud, inspired by GitHub's `gh` CLI.

## Installation

### Homebrew (macOS and Linux)

```bash
brew install rbansal42/tap/bb
```

### Download Binary

Download the latest release from the [releases page](https://github.com/rbansal42/bb/releases).

### Build from Source

```bash
go install github.com/rbansal42/bb/cmd/bb@latest
```

## Quick Start

1. **Authenticate with Bitbucket:**
   ```bash
   bb auth login
   ```

2. **Start using commands:**
   ```bash
   bb pr list
   bb repo clone workspace/repo
   bb issue create
   ```

## Commands

### Authentication
- `bb auth login` - Authenticate with Bitbucket
- `bb auth logout` - Log out of Bitbucket
- `bb auth status` - View authentication status

### Pull Requests
- `bb pr list` - List pull requests
- `bb pr view <number>` - View a pull request
- `bb pr create` - Create a pull request
- `bb pr merge <number>` - Merge a pull request
- `bb pr checkout <number>` - Checkout a pull request
- `bb pr close <number>` - Close a pull request
- `bb pr approve <number>` - Approve a pull request
- `bb pr diff <number>` - View pull request diff

### Repositories
- `bb repo list` - List repositories
- `bb repo view` - View repository details
- `bb repo clone <repo>` - Clone a repository
- `bb repo create` - Create a repository
- `bb repo fork <repo>` - Fork a repository

### Issues
- `bb issue list` - List issues
- `bb issue view <number>` - View an issue
- `bb issue create` - Create an issue
- `bb issue close <number>` - Close an issue
- `bb issue comment <number>` - Comment on an issue

### Pipelines
- `bb pipeline list` - List pipelines
- `bb pipeline view <uuid>` - View pipeline details
- `bb pipeline run` - Trigger a pipeline
- `bb pipeline logs <uuid>` - View pipeline logs
- `bb pipeline stop <uuid>` - Stop a running pipeline

### Branches
- `bb branch list` - List branches
- `bb branch create <name>` - Create a branch
- `bb branch delete <name>` - Delete a branch

### Workspaces
- `bb workspace list` - List workspaces
- `bb workspace view <slug>` - View workspace details
- `bb workspace members <slug>` - List workspace members

### Projects
- `bb project list` - List projects
- `bb project view <key>` - View project details
- `bb project create` - Create a project

### Snippets
- `bb snippet list` - List snippets
- `bb snippet view <id>` - View a snippet
- `bb snippet create` - Create a snippet
- `bb snippet edit <id>` - Edit a snippet
- `bb snippet delete <id>` - Delete a snippet

### Other
- `bb browse` - Open repository in browser
- `bb api <endpoint>` - Make API requests
- `bb config` - Manage configuration
- `bb completion` - Generate shell completions

## Shell Completion

Generate completions for your shell:

```bash
# Bash
bb completion bash > /etc/bash_completion.d/bb

# Zsh
bb completion zsh > "${fpath[1]}/_bb"

# Fish
bb completion fish > ~/.config/fish/completions/bb.fish

# PowerShell
bb completion powershell | Out-String | Invoke-Expression
```

## Configuration

Configuration is stored in `~/.config/bb/` (or `$XDG_CONFIG_HOME/bb/`).

- `hosts.yml` - Authentication tokens and host configuration
- `config.yml` - General settings

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Disclaimer

This is an unofficial CLI tool and is not affiliated with or endorsed by Atlassian or Bitbucket.
