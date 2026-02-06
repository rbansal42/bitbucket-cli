# bb workspace

Manage Bitbucket workspaces.

## Synopsis

```
bb workspace <subcommand> [flags]
```

## Description

Work with Bitbucket workspaces. List available workspaces, view workspace details, and manage workspace members.

Workspaces are the top-level organizational unit in Bitbucket Cloud. Each workspace can contain multiple repositories and projects.

## Subcommands

- [bb workspace list](#bb-workspace-list) - List workspaces
- [bb workspace view](#bb-workspace-view) - View workspace details
- [bb workspace members](#bb-workspace-members) - List workspace members

---

# bb workspace list

List workspaces you have access to.

## Synopsis

```
bb workspace list [flags]
```

## Description

Display a list of all Bitbucket workspaces that the authenticated user has access to. This includes workspaces you own and workspaces you've been invited to.

## Flags

| Flag | Description |
|------|-------------|
| `-r, --role <role>` | Filter by your role (owner, collaborator, member) |
| `-L, --limit <number>` | Maximum number of workspaces to list (default: 30) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

List all workspaces:

```
$ bb workspace list
SLUG          NAME                  ROLE          REPOSITORIES
myteam        My Team               owner         25
acme-corp     ACME Corporation      collaborator  142
open-source   Open Source Projects  member        8
```

Filter by role:

```
$ bb workspace list --role owner
SLUG          NAME                  ROLE    REPOSITORIES
myteam        My Team               owner   25
personal      Personal              owner   12
```

List with JSON output:

```
$ bb workspace list --json
[
  {
    "slug": "myteam",
    "name": "My Team",
    "role": "owner",
    "repository_count": 25
  },
  ...
]
```

## See also

- [bb workspace view](#bb-workspace-view) - View workspace details
- [bb workspace members](#bb-workspace-members) - List workspace members

---

# bb workspace view

View details of a workspace.

## Synopsis

```
bb workspace view [workspace] [flags]
```

## Description

Display detailed information about a specific workspace, including its name, description, creation date, and summary statistics.

If no workspace is specified, the default workspace (if configured) is shown.

## Flags

| Flag | Description |
|------|-------------|
| `-w, --web` | Open the workspace in a browser |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

View workspace details:

```
$ bb workspace view myteam
Name:           My Team
Slug:           myteam
Type:           team
Created:        2024-03-15
Repositories:   25
Projects:       5
Members:        12

Links:
  Website:      https://bitbucket.org/myteam
  Avatar:       https://bitbucket.org/account/myteam/avatar
```

View the default workspace:

```
$ bb workspace view
Name:           My Team
Slug:           myteam
...
```

Open workspace in browser:

```
$ bb workspace view myteam --web
Opening https://bitbucket.org/myteam in your browser
```

## See also

- [bb workspace list](#bb-workspace-list) - List workspaces
- [bb workspace members](#bb-workspace-members) - List workspace members

---

# bb workspace members

List members of a workspace.

## Synopsis

```
bb workspace members [workspace] [flags]
```

## Description

Display a list of all members in a workspace, including their username, display name, and permission level.

If no workspace is specified, the default workspace (if configured) is used.

## Flags

| Flag | Description |
|------|-------------|
| `-r, --role <role>` | Filter by role (owner, collaborator, member) |
| `-L, --limit <number>` | Maximum number of members to list (default: 30) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

List workspace members:

```
$ bb workspace members myteam
USERNAME      NAME              ROLE
johndoe       John Doe          owner
janesmith     Jane Smith        collaborator
bobwilson     Bob Wilson        member
alicebrown    Alice Brown       member
```

Filter by role:

```
$ bb workspace members myteam --role owner
USERNAME      NAME              ROLE
johndoe       John Doe          owner
```

List members of the default workspace:

```
$ bb workspace members
USERNAME      NAME              ROLE
johndoe       John Doe          owner
...
```

Output as JSON:

```
$ bb workspace members myteam --json
[
  {
    "username": "johndoe",
    "display_name": "John Doe",
    "role": "owner",
    "account_id": "5e5d5e5d5e5d5e5d5e5d5e5d"
  },
  ...
]
```

## See also

- [bb workspace list](#bb-workspace-list) - List workspaces
- [bb workspace view](#bb-workspace-view) - View workspace details
