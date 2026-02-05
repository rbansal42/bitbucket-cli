# bb branch

Manage repository branches.

## Synopsis

```
bb branch <subcommand> [flags]
```

## Description

Create, list, and manage branches in a Bitbucket repository. These commands allow you to work with branches directly from the command line.

## Subcommands

- [bb branch list](#bb-branch-list) - List branches
- [bb branch create](#bb-branch-create) - Create a new branch
- [bb branch delete](#bb-branch-delete) - Delete a branch

---

# bb branch list

List branches in a repository.

## Synopsis

```
bb branch list [flags]
```

## Description

Display a list of branches in the current or specified repository. By default, branches are sorted by most recent commit.

The default branch is indicated with an asterisk (*) and highlighted when viewing in a terminal.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-s, --sort <field>` | Sort by field: name, date (default: date) |
| `-f, --filter <pattern>` | Filter branches by name pattern |
| `-L, --limit <number>` | Maximum number of branches to list (default: 30) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

List branches:

```
$ bb branch list
* main          abc1234  Fix auth bug                2026-02-05
  feature/api   def5678  Add new endpoint            2026-02-04
  develop       ghi9012  Merge feature branch        2026-02-03
  hotfix/login  jkl3456  Emergency login fix         2026-02-01
```

Filter branches by pattern:

```
$ bb branch list --filter "feature/*"
  feature/api       def5678  Add new endpoint       2026-02-04
  feature/ui        mno7890  Update dashboard       2026-02-02
  feature/reports   pqr1234  Add export feature     2026-01-30
```

Sort by name:

```
$ bb branch list --sort name
```

List branches for a specific repository:

```
$ bb branch list -R myworkspace/myrepo
```

## See also

- [bb branch create](#bb-branch-create) - Create a new branch
- [bb branch delete](#bb-branch-delete) - Delete a branch

---

# bb branch create

Create a new branch.

## Synopsis

```
bb branch create <name> [flags]
```

## Description

Create a new branch in the repository. By default, the branch is created from the current HEAD commit. Use the `--target` flag to specify a different starting point.

The new branch is created remotely on Bitbucket. Use `git fetch` to retrieve it locally.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-t, --target <ref>` | Create branch from this ref (branch name, tag, or commit SHA) |
| `--checkout` | Checkout the new branch locally after creation |
| `-h, --help` | Show help for command |

## Examples

Create a branch from current HEAD:

```
$ bb branch create feature/new-feature
Created branch 'feature/new-feature' from main (abc1234)
```

Create a branch from a specific commit:

```
$ bb branch create hotfix/urgent --target abc1234
Created branch 'hotfix/urgent' from abc1234
```

Create a branch from another branch:

```
$ bb branch create release/v2.0 --target develop
Created branch 'release/v2.0' from develop (def5678)
```

Create and checkout locally:

```
$ bb branch create feature/api --checkout
Created branch 'feature/api' from main (abc1234)
Switched to branch 'feature/api'
```

## See also

- [bb branch list](#bb-branch-list) - List branches
- [bb branch delete](#bb-branch-delete) - Delete a branch

---

# bb branch delete

Delete a branch.

## Synopsis

```
bb branch delete <name> [flags]
```

## Description

Delete a branch from the remote repository on Bitbucket.

By default, you will be prompted to confirm deletion. The default branch cannot be deleted.

If the branch has unmerged changes, you will be warned unless `--force` is specified.

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <owner/repo>` | Select a repository (default: current repository) |
| `-y, --yes` | Skip confirmation prompt |
| `-f, --force` | Force deletion even if branch has unmerged commits |
| `-h, --help` | Show help for command |

## Examples

Delete a branch:

```
$ bb branch delete feature/old-feature
? Are you sure you want to delete branch 'feature/old-feature'? Yes
Deleted branch 'feature/old-feature'
```

Delete without confirmation:

```
$ bb branch delete feature/old-feature --yes
Deleted branch 'feature/old-feature'
```

Force delete a branch with unmerged changes:

```
$ bb branch delete feature/abandoned --force
Warning: Branch 'feature/abandoned' has 3 unmerged commits
Deleted branch 'feature/abandoned'
```

## See also

- [bb branch list](#bb-branch-list) - List branches
- [bb branch create](#bb-branch-create) - Create a new branch
