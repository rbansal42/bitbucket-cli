# Configuration Guide

This guide covers all configuration options for `bb`, including file locations, settings, and environment variables.

## Configuration File Locations

`bb` stores its configuration in `~/.config/bb/`:

```
~/.config/bb/
├── config.yml    # General settings
└── hosts.yml     # Authentication credentials per host
```

On Windows, the configuration directory is `%APPDATA%\bb\`.

## config.yml Structure

The main configuration file controls `bb` behavior:

```yaml
# Default Bitbucket host (for users with Bitbucket Data Center)
default_host: bitbucket.org

# Git protocol for cloning and remotes
git_protocol: https  # or: ssh

# Default editor for writing PR descriptions, comments, etc.
editor: vim

# Default workspace (optional - saves typing for single-workspace users)
default_workspace: mycompany

# Preferred pager for long output
pager: less

# HTTP settings
http:
  timeout: 30s
  max_retries: 3

# Output preferences
output:
  format: table  # or: json, yaml
  color: auto    # or: always, never

# Pull request defaults
pr:
  default_branch: main
  draft_by_default: false

# Aliases for common commands
aliases:
  co: pr checkout
  ls: pr list
  rv: pr review
```

## hosts.yml Structure

Authentication credentials are stored separately in `hosts.yml`:

```yaml
bitbucket.org:
  user: your-username
  oauth_token: your-access token-or-token
  git_protocol: ssh  # Override per-host

# For Bitbucket Data Center / Server installations
bitbucket.mycompany.com:
  user: jdoe
  oauth_token: xxxxxxxxxxxxxx
  git_protocol: https
```

> **Security Note:** `hosts.yml` contains sensitive credentials. Ensure it has restricted permissions (`chmod 600 ~/.config/bb/hosts.yml`).

## Using `bb config` Commands

### View Configuration

```bash
# Get a specific value
bb config get git_protocol
# Output: https

# Get a nested value
bb config get output.format
# Output: table

# List all configuration
bb config list
```

### Set Configuration

```bash
# Set a value
bb config set git_protocol ssh

# Set a nested value
bb config set output.format json

# Set for a specific host
bb config set -h bitbucket.mycompany.com git_protocol https
```

### Unset Configuration

```bash
# Remove a configuration value (reverts to default)
bb config unset editor
```

## Git Protocol Preference

`bb` supports both HTTPS and SSH for Git operations:

```bash
# Use SSH (requires SSH key setup)
bb config set git_protocol ssh

# Use HTTPS (requires access token)
bb config set git_protocol https
```

**When to use SSH:**
- You have SSH keys configured with Bitbucket
- You prefer not entering credentials for Git operations
- Your organization requires SSH

**When to use HTTPS:**
- Simpler setup with access tokens
- Working behind corporate firewalls that block SSH
- Using Bitbucket access tokens for authentication

The protocol affects:
- `bb repo clone` - URL used for cloning
- `bb pr checkout` - Remote URL for fetching PR branches
- `bb repo fork` - Remote URL added for your fork

## Editor Configuration

`bb` uses an editor for writing PR descriptions, comments, and other multi-line input:

```bash
# Set your preferred editor
bb config set editor "code --wait"   # VS Code
bb config set editor "vim"           # Vim
bb config set editor "nano"          # Nano
bb config set editor "subl -w"       # Sublime Text
```

Editor resolution order:
1. `BB_EDITOR` environment variable
2. `editor` in config.yml
3. `VISUAL` environment variable
4. `EDITOR` environment variable
5. Default: `nano` (macOS/Linux) or `notepad` (Windows)

## Environment Variables

Environment variables override configuration file settings:

| Variable | Description | Example |
|----------|-------------|---------|
| `BB_TOKEN` | Authentication token | `export BB_TOKEN=xxxx` |
| `BB_HOST` | Default Bitbucket host | `export BB_HOST=bitbucket.mycompany.com` |
| `BB_EDITOR` | Editor for composing text | `export BB_EDITOR="code --wait"` |
| `BB_PAGER` | Pager for long output | `export BB_PAGER=less` |
| `BB_WORKSPACE` | Default workspace | `export BB_WORKSPACE=myteam` |
| `BB_REPO` | Default repository | `export BB_REPO=myteam/myrepo` |
| `BB_NO_COLOR` | Disable colored output | `export BB_NO_COLOR=1` |
| `BB_DEBUG` | Enable debug logging | `export BB_DEBUG=1` |
| `BB_CONFIG_DIR` | Custom config directory | `export BB_CONFIG_DIR=/path/to/config` |

### CI/CD Usage

For CI/CD pipelines, use environment variables for authentication:

```yaml
# Bitbucket Pipelines example
script:
  - export BB_TOKEN=$BB_access token
  - bb pr list --state OPEN
```

```yaml
# GitHub Actions example (for cross-platform tools)
env:
  BB_TOKEN: ${{ secrets.BITBUCKET_TOKEN }}
  BB_WORKSPACE: mycompany
```

## Per-Repository Configuration

Create a `.bb.yml` file in your repository root for project-specific settings:

```yaml
# .bb.yml - Repository-specific configuration

# Default reviewers added to every PR
reviewers:
  - alice
  - bob

# Default PR settings
pr:
  default_branch: develop
  title_prefix: "[PROJ]"
  close_source_branch: true

# Custom pipelines triggers
pipelines:
  run_on_pr: true

# Issue tracker integration
issues:
  prefix: PROJ-
  link_pattern: "https://jira.company.com/browse/{issue}"
```

### Supported .bb.yml Settings

| Setting | Description |
|---------|-------------|
| `reviewers` | Default reviewers for PRs |
| `pr.default_branch` | Target branch for new PRs |
| `pr.close_source_branch` | Auto-close branch on merge |
| `pr.title_prefix` | Prefix added to PR titles |
| `pipelines.run_on_pr` | Trigger pipeline on PR creation |
| `issues.prefix` | Issue ID prefix for linking |
| `issues.link_pattern` | URL pattern for issue links |

## Configuration Precedence

`bb` resolves configuration in this order (highest to lowest priority):

1. **Command-line flags** - `--workspace`, `--repo`, etc.
2. **Environment variables** - `BB_TOKEN`, `BB_WORKSPACE`, etc.
3. **Repository config** - `.bb.yml` in current repo
4. **User config** - `~/.config/bb/config.yml`
5. **Host-specific config** - Settings in `hosts.yml`
6. **Built-in defaults**

### Example

```bash
# config.yml has: git_protocol: https
# Environment has: BB_GIT_PROTOCOL=ssh
# Command run: bb repo clone myrepo

# Result: Uses SSH because environment variable takes precedence
```

## Troubleshooting

### View Resolved Configuration

```bash
# Show all resolved settings with their sources
bb config list --show-source
```

Output:
```
git_protocol: ssh (from: environment BB_GIT_PROTOCOL)
editor: vim (from: /Users/you/.config/bb/config.yml)
default_workspace: myteam (from: .bb.yml)
pager: less (from: default)
```

### Reset Configuration

```bash
# Reset all configuration to defaults
rm -rf ~/.config/bb/config.yml

# Reset authentication (re-run login)
rm ~/.config/bb/hosts.yml
bb auth login
```

### Debug Mode

Enable verbose output to troubleshoot configuration issues:

```bash
BB_DEBUG=1 bb pr list
```

This shows:
- Configuration files loaded
- Environment variables detected
- API requests and responses
- Authentication method used
