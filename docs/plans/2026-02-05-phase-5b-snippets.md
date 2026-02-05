# Phase 5b: Snippet Commands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add snippet commands to bb CLI (Bitbucket's equivalent to GitHub Gists).

**Architecture:** New API file (snippets.go) with corresponding command package. Follows existing patterns.

**Tech Stack:** Go, Cobra CLI, Bitbucket Cloud API 2.0

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /snippets/{workspace} | List snippets in workspace |
| POST | /snippets/{workspace} | Create snippet |
| GET | /snippets/{workspace}/{encoded_id} | Get snippet |
| PUT | /snippets/{workspace}/{encoded_id} | Update snippet |
| DELETE | /snippets/{workspace}/{encoded_id} | Delete snippet |
| GET | /snippets/{workspace}/{encoded_id}/files/{path} | Get raw file content |

## Commands to Implement

1. `bb snippet list` - List snippets in a workspace
2. `bb snippet view <id>` - View snippet details
3. `bb snippet create` - Create a new snippet
4. `bb snippet edit <id>` - Edit an existing snippet  
5. `bb snippet delete <id>` - Delete a snippet

---

## Task 1: API Layer - Snippets Types and Methods

**Files:** `internal/api/snippets.go`

### Types

```go
// SnippetFile represents a file in a snippet
type SnippetFile struct {
    Links struct {
        Self Link `json:"self"`
        HTML Link `json:"html"`
    } `json:"links"`
}

// Snippet represents a Bitbucket snippet
type Snippet struct {
    Type      string                  `json:"type"`
    ID        int                     `json:"id"`
    Title     string                  `json:"title"`
    Scm       string                  `json:"scm"`
    CreatedOn string                  `json:"created_on"`
    UpdatedOn string                  `json:"updated_on"`
    Owner     *User                   `json:"owner"`
    Creator   *User                   `json:"creator"`
    IsPrivate bool                    `json:"is_private"`
    Files     map[string]SnippetFile  `json:"files"`
    Links     SnippetLinks            `json:"links"`
}

// SnippetLinks contains links for a snippet
type SnippetLinks struct {
    Self     Link `json:"self"`
    HTML     Link `json:"html"`
    Comments Link `json:"comments"`
    Watchers Link `json:"watchers"`
    Commits  Link `json:"commits"`
}

// SnippetListOptions for listing snippets
type SnippetListOptions struct {
    Role  string // owner, contributor, member
    Page  int
    Limit int
}

// SnippetCreateOptions for creating snippets
type SnippetCreateOptions struct {
    Title     string `json:"title"`
    IsPrivate bool   `json:"is_private"`
    Scm       string `json:"scm,omitempty"` // defaults to "git"
}
```

### Methods

```go
func (c *Client) ListSnippets(ctx, workspace string, opts *SnippetListOptions) (*Paginated[Snippet], error)
func (c *Client) GetSnippet(ctx, workspace, encodedID string) (*Snippet, error)
func (c *Client) CreateSnippet(ctx, workspace string, title string, isPrivate bool, files map[string]string) (*Snippet, error)
func (c *Client) UpdateSnippet(ctx, workspace, encodedID string, title string, files map[string]string) (*Snippet, error)
func (c *Client) DeleteSnippet(ctx, workspace, encodedID string) error
func (c *Client) GetSnippetFileContent(ctx, workspace, encodedID, path string) ([]byte, error)
```

---

## Task 2: API Layer - Snippets Tests

**Files:** `internal/api/snippets_test.go`

Test cases:
- TestListSnippets - basic list, with role filter, pagination, workspace not found
- TestGetSnippet - success, not found
- TestCreateSnippet - success, error cases
- TestUpdateSnippet - success, not found
- TestDeleteSnippet - success, not found
- TestGetSnippetFileContent - success, not found

---

## Task 3: Snippet Commands - Parent, Shared, List

**Files:**
- `internal/cmd/snippet/snippet.go` - Parent command
- `internal/cmd/snippet/shared.go` - Shared utilities
- `internal/cmd/snippet/list.go` - List command

### snippet.go
```go
func NewCmdSnippet(streams *iostreams.IOStreams) *cobra.Command
// Use: "snippet <command>", Aliases: []string{"snip"}
// Subcommands: list, view, create, edit, delete
```

### list.go
```go
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command
// Flags: --workspace/-w (required), --role (owner/contributor/member), --limit/-l, --json
// Aliases: []string{"ls"}
// Table: ID, TITLE, VISIBILITY, UPDATED
```

---

## Task 4: Snippet Commands - View

**Files:** `internal/cmd/snippet/view.go`

```go
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command
// Args: snippet ID (required)
// Flags: --workspace/-w (required), --web, --json, --raw (show raw content)
// Display: Title, ID, Visibility, Owner, Created, Updated, Files list, URL
```

---

## Task 5: Snippet Commands - Create

**Files:** `internal/cmd/snippet/create.go`

```go
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command
// Flags: --workspace/-w (required), --title/-t (required), --private/-p, --file/-f (repeatable), --json
// Read from stdin if no --file provided
// On success: "Created snippet {ID} in workspace {workspace}"
```

---

## Task 6: Snippet Commands - Edit and Delete

**Files:**
- `internal/cmd/snippet/edit.go`
- `internal/cmd/snippet/delete.go`

### edit.go
```go
func NewCmdEdit(streams *iostreams.IOStreams) *cobra.Command
// Args: snippet ID (required)
// Flags: --workspace/-w (required), --title/-t, --file/-f (repeatable), --json
```

### delete.go
```go
func NewCmdDelete(streams *iostreams.IOStreams) *cobra.Command
// Args: snippet ID (required)
// Flags: --workspace/-w (required), --force/-f
// Confirmation unless --force
```

---

## Task 7: Root Command Integration

**Files:** `internal/cmd/root.go`

Add import and registration:
```go
import "github.com/rbansal42/bb/internal/cmd/snippet"
rootCmd.AddCommand(snippet.NewCmdSnippet(GetStreams()))
```

---

## Summary

| Task | Files | Description |
|------|-------|-------------|
| 1 | snippets.go | API types and methods |
| 2 | snippets_test.go | API tests |
| 3 | snippet.go, shared.go, list.go | Parent and list commands |
| 4 | view.go | View command |
| 5 | create.go | Create command |
| 6 | edit.go, delete.go | Edit and delete commands |
| 7 | root.go | Integration |
