# bb repo

Manage Bitbucket repositories.

## Synopsis

```
bb repo <subcommand> [flags]
```

## Subcommands

- [list](#bb-repo-list) - List repositories
- [view](#bb-repo-view) - View repository details
- [clone](#bb-repo-clone) - Clone a repository
- [create](#bb-repo-create) - Create a new repository
- [fork](#bb-repo-fork) - Fork a repository
- [delete](#bb-repo-delete) - Delete a repository
- [sync](#bb-repo-sync) - Sync fork with upstream
- [set-default](#bb-repo-set-default) - Set default repository for directory

---

## bb repo list

List repositories in a workspace.

### Synopsis

```
bb repo list [flags]
```

### Description

Lists repositories accessible to the authenticated user. By default, lists repositories in the current workspace (determined from git remote or configuration).

### Flags

| Flag | Description |
|------|-------------|
| `--workspace`, `-w` | Workspace slug to list repositories from |
| `--limit`, `-l` | Maximum number of repositories to list (default: 30) |

### Examples

```bash
# List repositories in current workspace
bb repo list

# List repositories in a specific workspace
bb repo list --workspace myteam

# List first 50 repositories
bb repo list --limit 50
```

---

## bb repo view

View repository details.

### Synopsis

```
bb repo view [<repo>] [flags]
```

### Description

Displays detailed information about a repository including description, visibility, default branch, clone URLs, and recent activity. If no repository is specified, uses the repository in the current directory.

### Flags

| Flag | Description |
|------|-------------|
| `--web`, `-w` | Open the repository in the browser |

### Examples

```bash
# View current repository details
bb repo view

# View a specific repository
bb repo view myworkspace/myrepo

# Open repository in browser
bb repo view --web

# Open specific repository in browser
bb repo view myworkspace/myrepo --web
```

---

## bb repo clone

Clone a repository locally.

### Synopsis

```
bb repo clone <workspace/repo> [directory] [flags]
```

### Description

Clones a Bitbucket repository to the local filesystem. The repository must be specified in `workspace/repo` format. Optionally specify a target directory name.

### Flags

| Flag | Description |
|------|-------------|
| `--depth`, `-d` | Create a shallow clone with specified commit depth |
| `--branch`, `-b` | Clone a specific branch |

### Examples

```bash
# Clone a repository
bb repo clone myworkspace/myrepo

# Clone into a specific directory
bb repo clone myworkspace/myrepo my-local-dir

# Shallow clone with depth of 1
bb repo clone myworkspace/myrepo --depth 1

# Clone a specific branch
bb repo clone myworkspace/myrepo --branch develop

# Combine flags
bb repo clone myworkspace/myrepo --branch feature --depth 10
```

---

## bb repo create

Create a new repository.

### Synopsis

```
bb repo create [flags]
```

### Description

Creates a new repository in the specified workspace. If run interactively, prompts for required information. The repository name is derived from the `--name` flag or prompted interactively.

### Flags

| Flag | Description |
|------|-------------|
| `--name`, `-n` | Name of the repository |
| `--private`, `-p` | Make the repository private (default: true) |
| `--description`, `-d` | Description of the repository |
| `--project` | Project key to assign the repository to |

### Examples

```bash
# Create a repository interactively
bb repo create

# Create a private repository with name
bb repo create --name my-new-repo

# Create a public repository with description
bb repo create --name my-new-repo --private=false --description "My awesome project"

# Create repository in a specific project
bb repo create --name my-new-repo --project PROJ
```

---

## bb repo fork

Fork a repository.

### Synopsis

```
bb repo fork <workspace/repo> [flags]
```

### Description

Creates a fork of the specified repository in your personal workspace or a workspace you have access to. The fork maintains a link to the upstream repository.

### Examples

```bash
# Fork a repository to your personal workspace
bb repo fork upstream-workspace/repo-name

# Fork and clone in one flow (coming soon)
bb repo fork upstream-workspace/repo-name --clone
```

---

## bb repo delete

Delete a repository.

### Synopsis

```
bb repo delete <workspace/repo> [flags]
```

### Description

Permanently deletes a repository. This action cannot be undone. By default, prompts for confirmation before deletion.

### Flags

| Flag | Description |
|------|-------------|
| `--yes`, `-y` | Skip confirmation prompt |

### Examples

```bash
# Delete a repository (with confirmation)
bb repo delete myworkspace/myrepo

# Delete without confirmation
bb repo delete myworkspace/myrepo --yes
```

---

## bb repo sync

Sync fork with upstream repository.

### Synopsis

```
bb repo sync [flags]
```

### Description

Synchronizes a forked repository with its upstream parent. This fetches changes from the upstream repository and merges them into the fork's default branch. Must be run from within a forked repository.

### Examples

```bash
# Sync current fork with upstream
bb repo sync
```

---

## bb repo set-default

Set the default repository for the current directory.

### Synopsis

```
bb repo set-default [<workspace/repo>] [flags]
```

### Description

Sets or manages the default repository associated with the current directory. This is useful when working in directories that aren't git repositories or when you want to override the detected repository.

### Flags

| Flag | Description |
|------|-------------|
| `--view`, `-v` | View the currently set default repository |
| `--unset`, `-u` | Remove the default repository setting |

### Examples

```bash
# Set default repository for current directory
bb repo set-default myworkspace/myrepo

# View current default repository
bb repo set-default --view

# Remove default repository setting
bb repo set-default --unset
```

---

## See Also

- [bb pr](bb_pr.md) - Manage pull requests
- [bb issue](bb_issue.md) - Manage issues
- [bb workspace](bb_workspace.md) - Manage workspaces
