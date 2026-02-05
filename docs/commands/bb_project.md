# bb project

Manage Bitbucket projects.

## Synopsis

```
bb project <subcommand> [flags]
```

## Description

Work with Bitbucket projects within a workspace. Projects are used to organize and group related repositories.

Projects provide a way to manage permissions and settings across multiple repositories at once.

## Subcommands

- [bb project list](#bb-project-list) - List projects
- [bb project view](#bb-project-view) - View project details
- [bb project create](#bb-project-create) - Create a new project

---

# bb project list

List projects in a workspace.

## Synopsis

```
bb project list [flags]
```

## Description

Display a list of projects in a workspace. Projects help organize related repositories and can have their own permissions and settings.

## Flags

| Flag | Description |
|------|-------------|
| `-w, --workspace <slug>` | Workspace to list projects from (default: configured workspace) |
| `-L, --limit <number>` | Maximum number of projects to list (default: 30) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

List projects in the default workspace:

```
$ bb project list
KEY       NAME                  REPOSITORIES  DESCRIPTION
CORE      Core Platform         12            Core platform services
WEB       Web Applications      8             Frontend applications
MOBILE    Mobile Apps           5             iOS and Android apps
INFRA     Infrastructure        15            DevOps and infrastructure
```

List projects in a specific workspace:

```
$ bb project list --workspace acme-corp
KEY       NAME                  REPOSITORIES  DESCRIPTION
PROD      Product               20            Main product repos
INTERNAL  Internal Tools        7             Internal tooling
```

Output as JSON:

```
$ bb project list --json
[
  {
    "key": "CORE",
    "name": "Core Platform",
    "repository_count": 12,
    "description": "Core platform services"
  },
  ...
]
```

## See also

- [bb project view](#bb-project-view) - View project details
- [bb project create](#bb-project-create) - Create a new project

---

# bb project view

View details of a project.

## Synopsis

```
bb project view <project-key> [flags]
```

## Description

Display detailed information about a specific project, including its name, description, and associated repositories.

## Flags

| Flag | Description |
|------|-------------|
| `-w, --workspace <slug>` | Workspace containing the project (default: configured workspace) |
| `--web` | Open the project in a browser |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

View project details:

```
$ bb project view CORE
Key:            CORE
Name:           Core Platform
Workspace:      myteam
Description:    Core platform services and shared libraries
Created:        2024-01-15
Updated:        2026-02-01
Repositories:   12
Private:        Yes

Links:
  URL:          https://bitbucket.org/myteam/workspace/projects/CORE
  Avatar:       https://bitbucket.org/account/myteam/projects/CORE/avatar
```

View project in a specific workspace:

```
$ bb project view PROD --workspace acme-corp
```

Open project in browser:

```
$ bb project view CORE --web
Opening https://bitbucket.org/myteam/workspace/projects/CORE
```

## See also

- [bb project list](#bb-project-list) - List projects
- [bb project create](#bb-project-create) - Create a new project

---

# bb project create

Create a new project.

## Synopsis

```
bb project create <project-key> [flags]
```

## Description

Create a new project in a workspace. Projects are used to group and organize related repositories.

The project key must be unique within the workspace and typically uses uppercase letters (e.g., CORE, WEB, API).

## Flags

| Flag | Description |
|------|-------------|
| `-w, --workspace <slug>` | Workspace to create the project in (default: configured workspace) |
| `-n, --name <name>` | Display name for the project (required) |
| `-d, --description <text>` | Project description |
| `--private` | Make the project private (default: follows workspace settings) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

Create a project:

```
$ bb project create API --name "API Services"
Created project 'API' in workspace myteam
https://bitbucket.org/myteam/workspace/projects/API
```

Create a project with description:

```
$ bb project create DOCS --name "Documentation" --description "All documentation repositories"
Created project 'DOCS' in workspace myteam
https://bitbucket.org/myteam/workspace/projects/DOCS
```

Create a private project:

```
$ bb project create INTERNAL --name "Internal Tools" --private
Created project 'INTERNAL' in workspace myteam (private)
```

Create in a specific workspace:

```
$ bb project create QA --name "Quality Assurance" --workspace acme-corp
Created project 'QA' in workspace acme-corp
```

## See also

- [bb project list](#bb-project-list) - List projects
- [bb project view](#bb-project-view) - View project details
