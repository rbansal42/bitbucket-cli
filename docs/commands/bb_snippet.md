# bb snippet

Manage Bitbucket snippets.

## Synopsis

```
bb snippet <subcommand> [flags]
```

## Description

Create, view, and manage code snippets in Bitbucket. Snippets are small pieces of code or text that can be shared with others, similar to GitHub Gists.

Snippets can be public or private, and support multiple files with syntax highlighting.

## Subcommands

- [bb snippet list](#bb-snippet-list) - List snippets
- [bb snippet view](#bb-snippet-view) - View a snippet
- [bb snippet create](#bb-snippet-create) - Create a new snippet
- [bb snippet edit](#bb-snippet-edit) - Edit an existing snippet
- [bb snippet delete](#bb-snippet-delete) - Delete a snippet

---

# bb snippet list

List snippets.

## Synopsis

```
bb snippet list [flags]
```

## Description

Display a list of snippets owned by the authenticated user or a specified workspace. Shows the snippet title, visibility, and last modification date.

## Flags

| Flag | Description |
|------|-------------|
| `-w, --workspace <slug>` | List snippets from a specific workspace |
| `-r, --role <role>` | Filter by role (owner, contributor, member) |
| `-L, --limit <number>` | Maximum number of snippets to list (default: 30) |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

List your snippets:

```
$ bb snippet list
ID        TITLE                    VISIBILITY  UPDATED
abc123    Docker Compose Template  public      2026-02-05
def456    Bash Utilities           private     2026-02-03
ghi789    Python Helpers           public      2026-01-28
```

List snippets from a workspace:

```
$ bb snippet list --workspace myteam
ID        TITLE                    VISIBILITY  UPDATED
jkl012    Team Code Standards      public      2026-02-04
mno345    Deployment Scripts       private     2026-02-01
```

Output as JSON:

```
$ bb snippet list --json
[
  {
    "id": "abc123",
    "title": "Docker Compose Template",
    "is_private": false,
    "updated_on": "2026-02-05T10:30:00Z"
  },
  ...
]
```

## See also

- [bb snippet view](#bb-snippet-view) - View a snippet
- [bb snippet create](#bb-snippet-create) - Create a new snippet

---

# bb snippet view

View a snippet.

## Synopsis

```
bb snippet view <snippet-id> [flags]
```

## Description

Display the contents of a specific snippet. By default, shows all files in the snippet with syntax highlighting.

## Flags

| Flag | Description |
|------|-------------|
| `-f, --file <filename>` | Show only a specific file from the snippet |
| `-r, --raw` | Output raw content without formatting |
| `-w, --web` | Open the snippet in a browser |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

View a snippet:

```
$ bb snippet view abc123
Snippet: Docker Compose Template
Owner:   johndoe
Created: 2026-01-15
Updated: 2026-02-05
URL:     https://bitbucket.org/snippets/johndoe/abc123

--- docker-compose.yml ---
version: '3.8'
services:
  web:
    build: .
    ports:
      - "8080:8080"
  db:
    image: postgres:14
    environment:
      POSTGRES_PASSWORD: secret
```

View a specific file:

```
$ bb snippet view abc123 --file docker-compose.yml
version: '3.8'
services:
  web:
    build: .
...
```

Get raw content (useful for piping):

```
$ bb snippet view abc123 --raw > docker-compose.yml
```

Open in browser:

```
$ bb snippet view abc123 --web
Opening https://bitbucket.org/snippets/johndoe/abc123
```

## See also

- [bb snippet list](#bb-snippet-list) - List snippets
- [bb snippet edit](#bb-snippet-edit) - Edit an existing snippet

---

# bb snippet create

Create a new snippet.

## Synopsis

```
bb snippet create [flags]
```

## Description

Create a new snippet from files or standard input. Snippets can contain one or multiple files and can be public or private.

## Flags

| Flag | Description |
|------|-------------|
| `-t, --title <title>` | Title for the snippet |
| `-f, --filename <name>` | Filename when reading from stdin |
| `-p, --private` | Make the snippet private (default: public) |
| `-w, --workspace <slug>` | Create snippet in a workspace |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

Create a snippet from a file:

```
$ bb snippet create script.sh
Created snippet 'script.sh'
https://bitbucket.org/snippets/johndoe/xyz789
```

Create a snippet with a title:

```
$ bb snippet create config.yml --title "My Config Template"
Created snippet 'My Config Template'
https://bitbucket.org/snippets/johndoe/xyz789
```

Create from multiple files:

```
$ bb snippet create docker-compose.yml Dockerfile .env.example
Created snippet with 3 files
https://bitbucket.org/snippets/johndoe/xyz789
```

Create a private snippet:

```
$ bb snippet create secrets.sh --private
Created private snippet 'secrets.sh'
https://bitbucket.org/snippets/johndoe/xyz789
```

Create from stdin:

```
$ echo 'echo "Hello World"' | bb snippet create --filename hello.sh
Created snippet 'hello.sh'
https://bitbucket.org/snippets/johndoe/xyz789
```

Create in a workspace:

```
$ bb snippet create team-script.sh --workspace myteam
Created snippet 'team-script.sh' in workspace myteam
https://bitbucket.org/snippets/myteam/xyz789
```

## See also

- [bb snippet list](#bb-snippet-list) - List snippets
- [bb snippet view](#bb-snippet-view) - View a snippet
- [bb snippet edit](#bb-snippet-edit) - Edit an existing snippet

---

# bb snippet edit

Edit an existing snippet.

## Synopsis

```
bb snippet edit <snippet-id> [flags]
```

## Description

Edit an existing snippet by updating its title, adding files, removing files, or replacing file contents.

You can only edit snippets that you own or have write access to.

## Flags

| Flag | Description |
|------|-------------|
| `-t, --title <title>` | Update the snippet title |
| `-a, --add <file>` | Add a file to the snippet |
| `-r, --remove <filename>` | Remove a file from the snippet |
| `-u, --update <file>` | Update an existing file in the snippet |
| `-e, --editor` | Open snippet in your default editor |
| `--json` | Output in JSON format |
| `-h, --help` | Show help for command |

## Examples

Update snippet title:

```
$ bb snippet edit abc123 --title "Updated Docker Template"
Updated snippet 'Updated Docker Template'
```

Add a file to the snippet:

```
$ bb snippet edit abc123 --add nginx.conf
Added nginx.conf to snippet abc123
```

Remove a file:

```
$ bb snippet edit abc123 --remove old-config.yml
Removed old-config.yml from snippet abc123
```

Update an existing file:

```
$ bb snippet edit abc123 --update docker-compose.yml
Updated docker-compose.yml in snippet abc123
```

Open in editor:

```
$ bb snippet edit abc123 --editor
Opening snippet in editor...
Updated snippet abc123
```

## See also

- [bb snippet view](#bb-snippet-view) - View a snippet
- [bb snippet create](#bb-snippet-create) - Create a new snippet
- [bb snippet delete](#bb-snippet-delete) - Delete a snippet

---

# bb snippet delete

Delete a snippet.

## Synopsis

```
bb snippet delete <snippet-id> [flags]
```

## Description

Permanently delete a snippet. This action cannot be undone.

You can only delete snippets that you own or have admin access to.

## Flags

| Flag | Description |
|------|-------------|
| `-y, --yes` | Skip confirmation prompt |
| `-h, --help` | Show help for command |

## Examples

Delete a snippet:

```
$ bb snippet delete abc123
? Are you sure you want to delete snippet 'Docker Compose Template'? Yes
Deleted snippet abc123
```

Delete without confirmation:

```
$ bb snippet delete abc123 --yes
Deleted snippet abc123
```

## See also

- [bb snippet list](#bb-snippet-list) - List snippets
- [bb snippet create](#bb-snippet-create) - Create a new snippet
