# bb issue

Manage Bitbucket issues.

## Synopsis

```
bb issue <subcommand> [flags]
```

## Description

Create, view, and manage issues in a Bitbucket repository.

> **Note:** The Bitbucket issue tracker must be enabled on the repository to use these commands. Issue tracking is optional per-repository in Bitbucket Cloud. You can enable it in your repository settings under **Repository settings > Features > Issue tracker**.

## Subcommands

- [bb issue list](#bb-issue-list) - List issues
- [bb issue view](#bb-issue-view) - View issue details
- [bb issue create](#bb-issue-create) - Create a new issue
- [bb issue edit](#bb-issue-edit) - Edit an issue
- [bb issue close](#bb-issue-close) - Close an issue
- [bb issue reopen](#bb-issue-reopen) - Reopen an issue
- [bb issue comment](#bb-issue-comment) - Add a comment to an issue
- [bb issue delete](#bb-issue-delete) - Delete an issue

---

# bb issue list

List issues in a repository.

## Synopsis

```
bb issue list [flags]
```

## Description

List issues in the current repository or a specified repository. By default, displays open issues sorted by most recently updated.

Results can be filtered by state, kind, priority, and assignee. The output includes the issue ID, title, state, kind, and priority.

## Flags

| Flag | Description |
|------|-------------|
| `-s, --state <state>` | Filter by state: `new`, `open`, `resolved`, `on hold`, `invalid`, `duplicate`, `wontfix`, `closed` |
| `-k, --kind <kind>` | Filter by kind: `bug`, `enhancement`, `proposal`, `task` |
| `-p, --priority <priority>` | Filter by priority: `trivial`, `minor`, `major`, `critical`, `blocker` |
| `-a, --assignee <username>` | Filter by assignee username |
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `-L, --limit <number>` | Maximum number of issues to list (default 30) |
| `--json` | Output in JSON format |
| `-w, --web` | Open the issue list in browser |
| `-h, --help` | Show help for command |

## Examples

List open issues in the current repository:

```
$ bb issue list
ID    TITLE                          STATE   KIND          PRIORITY
#12   Fix login redirect             open    bug           major
#10   Add dark mode support          new     enhancement   minor
#8    Update documentation           open    task          trivial
```

List all bugs:

```
$ bb issue list --kind bug
```

List critical issues assigned to a specific user:

```
$ bb issue list --priority critical --assignee johndoe
```

List resolved issues:

```
$ bb issue list --state resolved
```

List issues in a specific repository:

```
$ bb issue list -R myworkspace/myrepo
```

Output issues as JSON:

```
$ bb issue list --json
```

## See also

- [bb issue view](#bb-issue-view) - View issue details
- [bb issue create](#bb-issue-create) - Create a new issue

---

# bb issue view

View issue details.

## Synopsis

```
bb issue view <id> [flags]
```

## Description

Display the details of a specific issue, including its title, state, kind, priority, description, reporter, assignee, and recent comments.

The issue ID is the numeric identifier shown in the issue list (e.g., `12` or `#12`).

## Flags

| Flag | Description |
|------|-------------|
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `-c, --comments` | Show issue comments |
| `--json` | Output in JSON format |
| `-w, --web` | Open the issue in browser |
| `-h, --help` | Show help for command |

## Examples

View issue #12:

```
$ bb issue view 12
Fix login redirect
open  bug  major

Opened by johndoe on Feb 5, 2026
Assigned to janedoe

  When users log in, they are redirected to the wrong page.
  
  Steps to reproduce:
  1. Log out
  2. Navigate to /dashboard
  3. Log in
  4. Observe redirect goes to / instead of /dashboard

View this issue on Bitbucket: https://bitbucket.org/myworkspace/myrepo/issues/12
```

View issue with comments:

```
$ bb issue view 12 --comments
```

Open issue in browser:

```
$ bb issue view 12 --web
```

Output as JSON:

```
$ bb issue view 12 --json
```

## See also

- [bb issue list](#bb-issue-list) - List issues
- [bb issue edit](#bb-issue-edit) - Edit an issue
- [bb issue comment](#bb-issue-comment) - Add a comment to an issue

---

# bb issue create

Create a new issue.

## Synopsis

```
bb issue create [flags]
```

## Description

Create a new issue in the repository. If no flags are provided, an interactive editor will open to compose the issue title and description.

The issue kind, priority, and assignee can be specified via flags. If not specified, the issue will be created with default values (kind: bug, priority: major).

## Flags

| Flag | Description |
|------|-------------|
| `-t, --title <title>` | Issue title |
| `-b, --body <body>` | Issue description/body |
| `-k, --kind <kind>` | Issue kind: `bug`, `enhancement`, `proposal`, `task` (default: bug) |
| `-p, --priority <priority>` | Issue priority: `trivial`, `minor`, `major`, `critical`, `blocker` (default: major) |
| `-a, --assignee <username>` | Assign issue to a user |
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `--json` | Output created issue in JSON format |
| `-w, --web` | Open the created issue in browser |
| `-h, --help` | Show help for command |

## Examples

Create an issue interactively:

```
$ bb issue create
```

Create a bug report with title and description:

```
$ bb issue create --title "Login button unresponsive" --body "The login button does not respond to clicks on mobile devices."
```

Create an enhancement request:

```
$ bb issue create --title "Add export to CSV" --kind enhancement --priority minor
```

Create and assign an issue:

```
$ bb issue create -t "Database migration failing" -k bug -p critical -a janedoe
```

Create an issue in a specific repository:

```
$ bb issue create -R myworkspace/myrepo --title "Update README"
```

## See also

- [bb issue list](#bb-issue-list) - List issues
- [bb issue view](#bb-issue-view) - View issue details
- [bb issue edit](#bb-issue-edit) - Edit an issue

---

# bb issue edit

Edit an existing issue.

## Synopsis

```
bb issue edit <id> [flags]
```

## Description

Edit an existing issue's title, description, kind, priority, or assignee. If no flags are provided, an interactive editor will open with the current issue content.

Only the fields specified via flags will be updated; other fields remain unchanged.

## Flags

| Flag | Description |
|------|-------------|
| `-t, --title <title>` | New issue title |
| `-b, --body <body>` | New issue description/body |
| `-k, --kind <kind>` | New issue kind: `bug`, `enhancement`, `proposal`, `task` |
| `-p, --priority <priority>` | New issue priority: `trivial`, `minor`, `major`, `critical`, `blocker` |
| `-a, --assignee <username>` | Reassign issue to a user |
| `--unassign` | Remove assignee from issue |
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `--json` | Output updated issue in JSON format |
| `-h, --help` | Show help for command |

## Examples

Edit issue interactively:

```
$ bb issue edit 12
```

Update issue title:

```
$ bb issue edit 12 --title "Login button unresponsive on mobile"
```

Change issue priority:

```
$ bb issue edit 12 --priority critical
```

Reassign an issue:

```
$ bb issue edit 12 --assignee bobsmith
```

Remove assignee from issue:

```
$ bb issue edit 12 --unassign
```

Update multiple fields:

```
$ bb issue edit 12 --title "Updated title" --priority major --kind enhancement
```

## See also

- [bb issue view](#bb-issue-view) - View issue details
- [bb issue close](#bb-issue-close) - Close an issue

---

# bb issue close

Close an issue.

## Synopsis

```
bb issue close <id> [flags]
```

## Description

Close an issue by setting its state to `resolved`. Optionally add a closing comment explaining the resolution.

Closed issues can be reopened using `bb issue reopen`.

## Flags

| Flag | Description |
|------|-------------|
| `-c, --comment <text>` | Add a comment when closing |
| `-r, --reason <state>` | Close reason/state: `resolved`, `invalid`, `duplicate`, `wontfix` (default: resolved) |
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `-h, --help` | Show help for command |

## Examples

Close an issue:

```
$ bb issue close 12
Issue #12 closed
```

Close with a comment:

```
$ bb issue close 12 --comment "Fixed in commit abc123"
```

Close as duplicate:

```
$ bb issue close 12 --reason duplicate --comment "Duplicate of #8"
```

Close as won't fix:

```
$ bb issue close 12 --reason wontfix --comment "Out of scope for this release"
```

## See also

- [bb issue reopen](#bb-issue-reopen) - Reopen an issue
- [bb issue view](#bb-issue-view) - View issue details

---

# bb issue reopen

Reopen a closed issue.

## Synopsis

```
bb issue reopen <id> [flags]
```

## Description

Reopen a previously closed issue by setting its state back to `open`. Optionally add a comment explaining why the issue is being reopened.

## Flags

| Flag | Description |
|------|-------------|
| `-c, --comment <text>` | Add a comment when reopening |
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `-h, --help` | Show help for command |

## Examples

Reopen an issue:

```
$ bb issue reopen 12
Issue #12 reopened
```

Reopen with a comment:

```
$ bb issue reopen 12 --comment "Bug still occurs in edge case"
```

## See also

- [bb issue close](#bb-issue-close) - Close an issue
- [bb issue view](#bb-issue-view) - View issue details

---

# bb issue comment

Add a comment to an issue.

## Synopsis

```
bb issue comment <id> [flags]
```

## Description

Add a comment to an existing issue. If the `--body` flag is not provided, an interactive editor will open to compose the comment.

Comments are displayed when viewing an issue with `bb issue view --comments`.

## Flags

| Flag | Description |
|------|-------------|
| `-b, --body <text>` | Comment text |
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `--json` | Output created comment in JSON format |
| `-w, --web` | Open the issue in browser after commenting |
| `-h, --help` | Show help for command |

## Examples

Add a comment interactively:

```
$ bb issue comment 12
```

Add a comment directly:

```
$ bb issue comment 12 --body "I can reproduce this on version 2.1.0"
```

Add a comment from a file:

```
$ bb issue comment 12 --body "$(cat comment.txt)"
```

Add a comment and open in browser:

```
$ bb issue comment 12 --body "See attached screenshot" --web
```

## See also

- [bb issue view](#bb-issue-view) - View issue details
- [bb issue edit](#bb-issue-edit) - Edit an issue

---

# bb issue delete

Delete an issue.

## Synopsis

```
bb issue delete <id> [flags]
```

## Description

Permanently delete an issue from the repository. This action cannot be undone.

By default, you will be prompted to confirm the deletion. Use `--yes` to skip the confirmation prompt.

## Flags

| Flag | Description |
|------|-------------|
| `-y, --yes` | Skip confirmation prompt |
| `-R, --repo <repo>` | Select repository as `workspace/repo` |
| `-h, --help` | Show help for command |

## Examples

Delete an issue (with confirmation):

```
$ bb issue delete 12
? Are you sure you want to delete issue #12 "Fix login redirect"? Yes
Issue #12 deleted
```

Delete without confirmation:

```
$ bb issue delete 12 --yes
Issue #12 deleted
```

Delete an issue in a specific repository:

```
$ bb issue delete 12 -R myworkspace/myrepo --yes
```

## See also

- [bb issue close](#bb-issue-close) - Close an issue
- [bb issue list](#bb-issue-list) - List issues
