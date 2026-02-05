# Phase 5a: Workspace, Project & Branch Commands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add workspace, project, and branch management commands to bb CLI.

**Architecture:** Three new API files (workspaces.go, projects.go, branches.go) with corresponding command packages. Each follows existing patterns from repositories.go and repo/ commands.

**Tech Stack:** Go, Cobra CLI, Bitbucket Cloud API 2.0

---

## Task 1: API Layer - Workspaces

**Files:**
- Create: `internal/api/workspaces.go`
- Test: `internal/api/workspaces_test.go`

**Step 1: Create workspaces.go with types and methods**

```go
package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// WorkspaceFull represents a Bitbucket workspace with full details
type WorkspaceFull struct {
	UUID      string         `json:"uuid"`
	Slug      string         `json:"slug"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	IsPrivate bool           `json:"is_private"`
	CreatedOn time.Time      `json:"created_on"`
	Links     WorkspaceLinks `json:"links"`
}

// WorkspaceLinks contains links related to a workspace
type WorkspaceLinks struct {
	Self         Link `json:"self"`
	HTML         Link `json:"html"`
	Avatar       Link `json:"avatar"`
	Members      Link `json:"members"`
	Projects     Link `json:"projects"`
	Repositories Link `json:"repositories"`
}

// WorkspaceMembership represents a user's membership in a workspace
type WorkspaceMembership struct {
	User       *User     `json:"user"`
	Workspace  Workspace `json:"workspace"`
	Permission string    `json:"permission"` // owner, collaborator, member
	AddedOn    time.Time `json:"added_on"`
	Links      struct {
		Self Link `json:"self"`
	} `json:"links"`
}

// WorkspaceListOptions are options for listing workspaces
type WorkspaceListOptions struct {
	Role  string // Filter by role: owner, collaborator, member
	Sort  string // Sort field
	Query string // Filter query
	Page  int
	Limit int
}

// WorkspaceMemberListOptions are options for listing workspace members
type WorkspaceMemberListOptions struct {
	Page  int
	Limit int
}

// ListWorkspaces lists workspaces the authenticated user has access to
func (c *Client) ListWorkspaces(ctx context.Context, opts *WorkspaceListOptions) (*Paginated[WorkspaceMembership], error) {
	path := "/user/permissions/workspaces"

	query := url.Values{}
	if opts != nil {
		if opts.Role != "" {
			query.Set("q", fmt.Sprintf("permission=\"%s\"", opts.Role))
		}
		if opts.Sort != "" {
			query.Set("sort", opts.Sort)
		}
		if opts.Query != "" {
			if query.Get("q") != "" {
				query.Set("q", query.Get("q")+" AND "+opts.Query)
			} else {
				query.Set("q", opts.Query)
			}
		}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Limit > 0 {
			query.Set("pagelen", strconv.Itoa(opts.Limit))
		}
	}

	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[WorkspaceMembership]](resp)
}

// GetWorkspace retrieves a single workspace by slug
func (c *Client) GetWorkspace(ctx context.Context, workspaceSlug string) (*WorkspaceFull, error) {
	path := fmt.Sprintf("/workspaces/%s", workspaceSlug)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*WorkspaceFull](resp)
}

// ListWorkspaceMembers lists members of a workspace
func (c *Client) ListWorkspaceMembers(ctx context.Context, workspaceSlug string, opts *WorkspaceMemberListOptions) (*Paginated[WorkspaceMembership], error) {
	path := fmt.Sprintf("/workspaces/%s/permissions", workspaceSlug)

	query := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Limit > 0 {
			query.Set("pagelen", strconv.Itoa(opts.Limit))
		}
	}

	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[WorkspaceMembership]](resp)
}
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/api/workspaces.go
git commit -m "feat(api): add workspace types and methods"
```

---

## Task 2: API Layer - Workspaces Tests

**Files:**
- Create: `internal/api/workspaces_test.go`

**Step 1: Create workspaces_test.go**

```go
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListWorkspaces(t *testing.T) {
	tests := []struct {
		name          string
		opts          *WorkspaceListOptions
		expectedQuery map[string]string
		response      string
		statusCode    int
		wantErr       bool
		wantCount     int
	}{
		{
			name: "basic list without options",
			opts: nil,
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"workspace": {"uuid": "{ws-1}", "slug": "workspace1", "name": "Workspace 1"}, "permission": "owner"},
					{"workspace": {"uuid": "{ws-2}", "slug": "workspace2", "name": "Workspace 2"}, "permission": "member"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:          "list with role filter",
			opts:          &WorkspaceListOptions{Role: "owner"},
			expectedQuery: map[string]string{"q": `permission="owner"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"workspace": {"uuid": "{ws-1}", "slug": "myworkspace"}, "permission": "owner"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:          "list with pagination",
			opts:          &WorkspaceListOptions{Page: 2, Limit: 5},
			expectedQuery: map[string]string{"page": "2", "pagelen": "5"},
			response: `{
				"size": 10,
				"page": 2,
				"pagelen": 5,
				"values": []
			}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "handles 401 unauthorized",
			opts:       nil,
			response:   `{"error": {"message": "Unauthorized"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.ListWorkspaces(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify query parameters
			for key, expected := range tt.expectedQuery {
				actual := receivedReq.URL.Query().Get(key)
				if actual != expected {
					t.Errorf("expected query param %s=%q, got %q", key, expected, actual)
				}
			}

			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d workspaces, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetWorkspace(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		response   string
		statusCode int
		wantErr    bool
		wantSlug   string
	}{
		{
			name: "successfully get workspace",
			slug: "myworkspace",
			response: `{
				"uuid": "{ws-uuid}",
				"slug": "myworkspace",
				"name": "My Workspace",
				"type": "workspace",
				"is_private": false,
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace"},
					"html": {"href": "https://bitbucket.org/myworkspace"}
				}
			}`,
			statusCode: http.StatusOK,
			wantSlug:   "myworkspace",
		},
		{
			name:       "workspace not found",
			slug:       "nonexistent",
			response:   `{"error": {"message": "Workspace not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.GetWorkspace(context.Background(), tt.slug)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedPath := "/workspaces/" + tt.slug
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path to end with %q, got %q", expectedPath, receivedReq.URL.Path)
			}

			if result.Slug != tt.wantSlug {
				t.Errorf("expected slug %q, got %q", tt.wantSlug, result.Slug)
			}
		})
	}
}

func TestListWorkspaceMembers(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		opts       *WorkspaceMemberListOptions
		response   string
		statusCode int
		wantErr    bool
		wantCount  int
	}{
		{
			name: "list members successfully",
			slug: "myworkspace",
			opts: nil,
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"user": {"display_name": "User 1", "uuid": "{user-1}"}, "permission": "owner"},
					{"user": {"display_name": "User 2", "uuid": "{user-2}"}, "permission": "member"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "workspace not found",
			slug:       "nonexistent",
			opts:       nil,
			response:   `{"error": {"message": "Workspace not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.ListWorkspaceMembers(context.Background(), tt.slug, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d members, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/api/... -run TestWorkspace -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/api/workspaces_test.go
git commit -m "test(api): add workspace API tests"
```

---

## Task 3: API Layer - Projects

**Files:**
- Create: `internal/api/projects.go`
- Test: `internal/api/projects_test.go`

**Step 1: Create projects.go with types and methods**

```go
package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ProjectFull represents a Bitbucket project with full details
type ProjectFull struct {
	UUID        string       `json:"uuid"`
	Key         string       `json:"key"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsPrivate   bool         `json:"is_private"`
	CreatedOn   time.Time    `json:"created_on"`
	UpdatedOn   time.Time    `json:"updated_on"`
	Owner       *User        `json:"owner"`
	Workspace   *Workspace   `json:"workspace"`
	Links       ProjectLinks `json:"links"`
}

// ProjectLinks contains links related to a project
type ProjectLinks struct {
	Self   Link `json:"self"`
	HTML   Link `json:"html"`
	Avatar Link `json:"avatar"`
}

// ProjectListOptions are options for listing projects
type ProjectListOptions struct {
	Sort  string // Sort field
	Query string // Filter query
	Page  int
	Limit int
}

// ProjectCreateOptions are options for creating a project
type ProjectCreateOptions struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
}

// ListProjects lists projects in a workspace
func (c *Client) ListProjects(ctx context.Context, workspaceSlug string, opts *ProjectListOptions) (*Paginated[ProjectFull], error) {
	path := fmt.Sprintf("/workspaces/%s/projects", workspaceSlug)

	query := url.Values{}
	if opts != nil {
		if opts.Sort != "" {
			query.Set("sort", opts.Sort)
		}
		if opts.Query != "" {
			query.Set("q", opts.Query)
		}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Limit > 0 {
			query.Set("pagelen", strconv.Itoa(opts.Limit))
		}
	}

	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[ProjectFull]](resp)
}

// GetProject retrieves a single project by key
func (c *Client) GetProject(ctx context.Context, workspaceSlug, projectKey string) (*ProjectFull, error) {
	path := fmt.Sprintf("/workspaces/%s/projects/%s", workspaceSlug, projectKey)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*ProjectFull](resp)
}

// CreateProject creates a new project in a workspace
func (c *Client) CreateProject(ctx context.Context, workspaceSlug string, opts *ProjectCreateOptions) (*ProjectFull, error) {
	path := fmt.Sprintf("/workspaces/%s/projects", workspaceSlug)

	resp, err := c.Post(ctx, path, opts)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*ProjectFull](resp)
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(ctx context.Context, workspaceSlug, projectKey string, opts *ProjectCreateOptions) (*ProjectFull, error) {
	path := fmt.Sprintf("/workspaces/%s/projects/%s", workspaceSlug, projectKey)

	resp, err := c.Put(ctx, path, opts)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*ProjectFull](resp)
}

// DeleteProject deletes a project
func (c *Client) DeleteProject(ctx context.Context, workspaceSlug, projectKey string) error {
	path := fmt.Sprintf("/workspaces/%s/projects/%s", workspaceSlug, projectKey)

	_, err := c.Delete(ctx, path)
	return err
}
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/api/projects.go
git commit -m "feat(api): add project types and methods"
```

---

## Task 4: API Layer - Projects Tests

**Files:**
- Create: `internal/api/projects_test.go`

**Step 1: Create projects_test.go**

```go
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListProjects(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		opts       *ProjectListOptions
		response   string
		statusCode int
		wantErr    bool
		wantCount  int
	}{
		{
			name:      "basic list without options",
			workspace: "myworkspace",
			opts:      nil,
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"uuid": "{proj-1}", "key": "PROJ1", "name": "Project 1"},
					{"uuid": "{proj-2}", "key": "PROJ2", "name": "Project 2"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:      "list with pagination",
			workspace: "myworkspace",
			opts:      &ProjectListOptions{Page: 2, Limit: 5},
			response: `{
				"size": 10,
				"page": 2,
				"pagelen": 5,
				"values": []
			}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "workspace not found",
			workspace:  "nonexistent",
			opts:       nil,
			response:   `{"error": {"message": "Workspace not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.ListProjects(context.Background(), tt.workspace, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d projects, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetProject(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		projectKey string
		response   string
		statusCode int
		wantErr    bool
		wantKey    string
	}{
		{
			name:       "successfully get project",
			workspace:  "myworkspace",
			projectKey: "PROJ",
			response: `{
				"uuid": "{proj-uuid}",
				"key": "PROJ",
				"name": "My Project",
				"description": "A test project",
				"is_private": true,
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects/PROJ"},
					"html": {"href": "https://bitbucket.org/myworkspace/workspace/projects/PROJ"}
				}
			}`,
			statusCode: http.StatusOK,
			wantKey:    "PROJ",
		},
		{
			name:       "project not found",
			workspace:  "myworkspace",
			projectKey: "NONEXISTENT",
			response:   `{"error": {"message": "Project not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.GetProject(context.Background(), tt.workspace, tt.projectKey)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedPath := "/workspaces/" + tt.workspace + "/projects/" + tt.projectKey
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path to end with %q, got %q", expectedPath, receivedReq.URL.Path)
			}

			if result.Key != tt.wantKey {
				t.Errorf("expected key %q, got %q", tt.wantKey, result.Key)
			}
		})
	}
}

func TestCreateProject(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		opts       *ProjectCreateOptions
		response   string
		statusCode int
		wantErr    bool
		wantKey    string
	}{
		{
			name:      "create project successfully",
			workspace: "myworkspace",
			opts: &ProjectCreateOptions{
				Key:         "NEWPROJ",
				Name:        "New Project",
				Description: "A new project",
				IsPrivate:   true,
			},
			response: `{
				"uuid": "{new-proj-uuid}",
				"key": "NEWPROJ",
				"name": "New Project",
				"description": "A new project",
				"is_private": true
			}`,
			statusCode: http.StatusCreated,
			wantKey:    "NEWPROJ",
		},
		{
			name:      "project already exists",
			workspace: "myworkspace",
			opts: &ProjectCreateOptions{
				Key:  "EXISTING",
				Name: "Existing Project",
			},
			response:   `{"error": {"message": "Project with this key already exists"}}`,
			statusCode: http.StatusConflict,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedBody []byte
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				receivedBody, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.CreateProject(context.Background(), tt.workspace, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(receivedBody, &body); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			if body["key"] != tt.opts.Key {
				t.Errorf("expected key %q in body, got %v", tt.opts.Key, body["key"])
			}

			if result.Key != tt.wantKey {
				t.Errorf("expected key %q, got %q", tt.wantKey, result.Key)
			}
		})
	}
}

func TestDeleteProject(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		projectKey string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful deletion",
			workspace:  "myworkspace",
			projectKey: "PROJ",
			statusCode: http.StatusNoContent,
			response:   "",
			wantErr:    false,
		},
		{
			name:       "project not found",
			workspace:  "myworkspace",
			projectKey: "NONEXISTENT",
			statusCode: http.StatusNotFound,
			response:   `{"error": {"message": "Project not found"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.response != "" {
					w.Write([]byte(tt.response))
				}
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			err := client.DeleteProject(context.Background(), tt.workspace, tt.projectKey)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedReq.Method != http.MethodDelete {
				t.Errorf("expected DELETE method, got %s", receivedReq.Method)
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/api/... -run TestProject -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/api/projects_test.go
git commit -m "test(api): add project API tests"
```

---

## Task 5: API Layer - Branches

**Files:**
- Create: `internal/api/branches.go`
- Test: `internal/api/branches_test.go`

**Step 1: Create branches.go with types and methods**

```go
package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// BranchFull represents a Bitbucket branch with full details
type BranchFull struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Target *BranchHead `json:"target"`
	Links  BranchLinks `json:"links"`
}

// BranchHead represents the head commit of a branch
type BranchHead struct {
	Hash    string `json:"hash"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Author  *struct {
		Raw  string `json:"raw"`
		User *User  `json:"user"`
	} `json:"author"`
	Date  string `json:"date"`
	Links struct {
		Self Link `json:"self"`
		HTML Link `json:"html"`
	} `json:"links"`
}

// BranchLinks contains links related to a branch
type BranchLinks struct {
	Self    Link `json:"self"`
	Commits Link `json:"commits"`
	HTML    Link `json:"html"`
}

// BranchListOptions are options for listing branches
type BranchListOptions struct {
	Sort  string // Sort field: name, -name
	Query string // Filter query
	Page  int
	Limit int
}

// BranchCreateOptions are options for creating a branch
type BranchCreateOptions struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

// ListBranches lists branches in a repository
func (c *Client) ListBranches(ctx context.Context, workspace, repoSlug string, opts *BranchListOptions) (*Paginated[BranchFull], error) {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches", workspace, repoSlug)

	query := url.Values{}
	if opts != nil {
		if opts.Sort != "" {
			query.Set("sort", opts.Sort)
		}
		if opts.Query != "" {
			query.Set("q", opts.Query)
		}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Limit > 0 {
			query.Set("pagelen", strconv.Itoa(opts.Limit))
		}
	}

	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[BranchFull]](resp)
}

// GetBranch retrieves a single branch by name
func (c *Client) GetBranch(ctx context.Context, workspace, repoSlug, branchName string) (*BranchFull, error) {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", workspace, repoSlug, url.PathEscape(branchName))

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*BranchFull](resp)
}

// CreateBranch creates a new branch in a repository
func (c *Client) CreateBranch(ctx context.Context, workspace, repoSlug string, opts *BranchCreateOptions) (*BranchFull, error) {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches", workspace, repoSlug)

	resp, err := c.Post(ctx, path, opts)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*BranchFull](resp)
}

// DeleteBranch deletes a branch from a repository
func (c *Client) DeleteBranch(ctx context.Context, workspace, repoSlug, branchName string) error {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", workspace, repoSlug, url.PathEscape(branchName))

	_, err := c.Delete(ctx, path)
	return err
}
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/api/branches.go
git commit -m "feat(api): add branch types and methods"
```

---

## Task 6: API Layer - Branches Tests

**Files:**
- Create: `internal/api/branches_test.go`

**Step 1: Create branches_test.go**

```go
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListBranches(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		opts       *BranchListOptions
		response   string
		statusCode int
		wantErr    bool
		wantCount  int
	}{
		{
			name:      "basic list without options",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts:      nil,
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"name": "main", "type": "branch", "target": {"hash": "abc123"}},
					{"name": "develop", "type": "branch", "target": {"hash": "def456"}}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:      "list with pagination",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts:      &BranchListOptions{Page: 2, Limit: 5},
			response: `{
				"size": 10,
				"page": 2,
				"pagelen": 5,
				"values": []
			}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "repository not found",
			workspace:  "myworkspace",
			repoSlug:   "nonexistent",
			opts:       nil,
			response:   `{"error": {"message": "Repository not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.ListBranches(context.Background(), tt.workspace, tt.repoSlug, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d branches, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetBranch(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		branchName string
		response   string
		statusCode int
		wantErr    bool
		wantName   string
	}{
		{
			name:       "successfully get branch",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			branchName: "main",
			response: `{
				"name": "main",
				"type": "branch",
				"target": {
					"hash": "abc123def456",
					"message": "Latest commit",
					"date": "2024-01-15T10:00:00Z"
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches/main"},
					"html": {"href": "https://bitbucket.org/myworkspace/myrepo/branch/main"}
				}
			}`,
			statusCode: http.StatusOK,
			wantName:   "main",
		},
		{
			name:       "branch with slash in name",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			branchName: "feature/new-feature",
			response: `{
				"name": "feature/new-feature",
				"type": "branch",
				"target": {"hash": "xyz789"}
			}`,
			statusCode: http.StatusOK,
			wantName:   "feature/new-feature",
		},
		{
			name:       "branch not found",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			branchName: "nonexistent",
			response:   `{"error": {"message": "Branch not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.GetBranch(context.Background(), tt.workspace, tt.repoSlug, tt.branchName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
			}
		})
	}
}

func TestCreateBranch(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		opts       *BranchCreateOptions
		response   string
		statusCode int
		wantErr    bool
		wantName   string
	}{
		{
			name:      "create branch successfully",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &BranchCreateOptions{
				Name:   "feature/new-branch",
				Target: struct{ Hash string `json:"hash"` }{Hash: "abc123"},
			},
			response: `{
				"name": "feature/new-branch",
				"type": "branch",
				"target": {"hash": "abc123"}
			}`,
			statusCode: http.StatusCreated,
			wantName:   "feature/new-branch",
		},
		{
			name:      "branch already exists",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &BranchCreateOptions{
				Name:   "existing-branch",
				Target: struct{ Hash string `json:"hash"` }{Hash: "abc123"},
			},
			response:   `{"error": {"message": "Branch already exists"}}`,
			statusCode: http.StatusConflict,
			wantErr:    true,
		},
		{
			name:      "invalid target commit",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &BranchCreateOptions{
				Name:   "new-branch",
				Target: struct{ Hash string `json:"hash"` }{Hash: "invalid"},
			},
			response:   `{"error": {"message": "Target commit not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedBody []byte
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				receivedBody, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.CreateBranch(context.Background(), tt.workspace, tt.repoSlug, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(receivedBody, &body); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			if body["name"] != tt.opts.Name {
				t.Errorf("expected name %q in body, got %v", tt.opts.Name, body["name"])
			}

			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
			}
		})
	}
}

func TestDeleteBranch(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		branchName string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful deletion",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			branchName: "feature/old-branch",
			statusCode: http.StatusNoContent,
			response:   "",
			wantErr:    false,
		},
		{
			name:       "branch not found",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			branchName: "nonexistent",
			statusCode: http.StatusNotFound,
			response:   `{"error": {"message": "Branch not found"}}`,
			wantErr:    true,
		},
		{
			name:       "cannot delete default branch",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			branchName: "main",
			statusCode: http.StatusForbidden,
			response:   `{"error": {"message": "Cannot delete the main branch"}}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				if tt.response != "" {
					w.Write([]byte(tt.response))
				}
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			err := client.DeleteBranch(context.Background(), tt.workspace, tt.repoSlug, tt.branchName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedReq.Method != http.MethodDelete {
				t.Errorf("expected DELETE method, got %s", receivedReq.Method)
			}

			// Verify branch name is in URL (URL encoded for slashes)
			if !strings.Contains(receivedReq.URL.Path, "refs/branches") {
				t.Errorf("expected URL to contain refs/branches, got %s", receivedReq.URL.Path)
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/api/... -run TestBranch -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/api/branches_test.go
git commit -m "test(api): add branch API tests"
```

---

## Task 7: Workspace Commands - Parent and List

**Files:**
- Create: `internal/cmd/workspace/workspace.go`
- Create: `internal/cmd/workspace/list.go`
- Create: `internal/cmd/workspace/shared.go`

**Step 1: Create workspace parent command**

```go
// internal/cmd/workspace/workspace.go
package workspace

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdWorkspace creates the workspace command and its subcommands
func NewCmdWorkspace(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace <command>",
		Short: "Work with Bitbucket workspaces",
		Long: `View and manage Bitbucket workspaces.

A workspace is a shared space where teams can collaborate on repositories.
You can belong to multiple workspaces with different roles.`,
		Example: `  # List all workspaces you have access to
  bb workspace list

  # View details of a specific workspace
  bb workspace view myworkspace

  # List members of a workspace
  bb workspace members myworkspace`,
		Aliases: []string{"ws"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdMembers(streams))

	return cmd
}
```

**Step 2: Create shared utilities**

```go
// internal/cmd/workspace/shared.go
package workspace

import (
	"encoding/json"
	"fmt"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
)

// getAPIClient creates an authenticated API client
func getAPIClient() (*api.Client, error) {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load hosts config: %w", err)
	}

	user := hosts.GetActiveUser(config.DefaultHost)
	if user == "" {
		return nil, fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
	}

	tokenData, _, err := config.GetTokenFromEnvOrKeyring(config.DefaultHost, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	token := tokenData
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		token = tokenResp.AccessToken
	}

	return api.NewClient(api.WithToken(token)), nil
}
```

**Step 3: Create list command**

```go
// internal/cmd/workspace/list.go
package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type listOptions struct {
	streams *iostreams.IOStreams
	role    string
	limit   int
	json    bool
}

// NewCmdList creates the workspace list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &listOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces you have access to",
		Long: `List all Bitbucket workspaces you have access to.

You can filter by your role in the workspace using the --role flag.`,
		Example: `  # List all workspaces
  bb workspace list

  # List only workspaces you own
  bb workspace list --role owner

  # Output as JSON
  bb workspace list --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.role, "role", "r", "", "Filter by role (owner, collaborator, member)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 30, "Maximum number of workspaces to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	listOpts := &api.WorkspaceListOptions{
		Role:  opts.role,
		Limit: opts.limit,
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := client.ListWorkspaces(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(result.Values) == 0 {
		opts.streams.Info("No workspaces found")
		return nil
	}

	if opts.json {
		return outputListJSON(opts.streams, result.Values)
	}

	return outputListTable(opts.streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, memberships []api.WorkspaceMembership) error {
	output := make([]map[string]interface{}, len(memberships))
	for i, m := range memberships {
		output[i] = map[string]interface{}{
			"slug":       m.Workspace.Slug,
			"name":       m.Workspace.Name,
			"uuid":       m.Workspace.UUID,
			"permission": m.Permission,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputListTable(streams *iostreams.IOStreams, memberships []api.WorkspaceMembership) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	header := "SLUG\tNAME\tROLE"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	for _, m := range memberships {
		role := formatRole(streams, m.Permission)
		fmt.Fprintf(w, "%s\t%s\t%s\n", m.Workspace.Slug, m.Workspace.Name, role)
	}

	return w.Flush()
}

func formatRole(streams *iostreams.IOStreams, role string) string {
	if !streams.ColorEnabled() {
		return role
	}

	switch role {
	case "owner":
		return iostreams.Yellow + role + iostreams.Reset
	case "collaborator":
		return iostreams.Cyan + role + iostreams.Reset
	default:
		return role
	}
}
```

**Step 4: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/cmd/workspace/
git commit -m "feat(cmd): add workspace list command"
```

---

## Task 8: Workspace Commands - View and Members

**Files:**
- Create: `internal/cmd/workspace/view.go`
- Create: `internal/cmd/workspace/members.go`

**Step 1: Create view command**

```go
// internal/cmd/workspace/view.go
package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/browser"
	"github.com/rbansal42/bb/internal/iostreams"
)

type viewOptions struct {
	streams *iostreams.IOStreams
	slug    string
	web     bool
	json    bool
}

// NewCmdView creates the workspace view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &viewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view <workspace>",
		Short: "View a workspace",
		Long: `Display details of a Bitbucket workspace.

Shows workspace information including name, UUID, and links.`,
		Example: `  # View a workspace
  bb workspace view myworkspace

  # Open workspace in browser
  bb workspace view myworkspace --web

  # Output as JSON
  bb workspace view myworkspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.slug = args[0]
			return runView(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.web, "web", "w", false, "Open the workspace in a web browser")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runView(ctx context.Context, opts *viewOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	workspace, err := client.GetWorkspace(ctx, opts.slug)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if opts.web {
		if err := browser.Open(workspace.Links.HTML.Href); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.streams.Success("Opened %s in your browser", workspace.Links.HTML.Href)
		return nil
	}

	if opts.json {
		data, err := json.MarshalIndent(workspace, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(opts.streams.Out, string(data))
		return nil
	}

	return displayWorkspace(opts.streams, workspace)
}

func displayWorkspace(streams *iostreams.IOStreams, ws *api.WorkspaceFull) error {
	fmt.Fprintf(streams.Out, "%s\n\n", ws.Name)
	fmt.Fprintf(streams.Out, "Slug:    %s\n", ws.Slug)
	fmt.Fprintf(streams.Out, "UUID:    %s\n", ws.UUID)
	fmt.Fprintf(streams.Out, "Type:    %s\n", ws.Type)

	visibility := "public"
	if ws.IsPrivate {
		visibility = "private"
	}
	fmt.Fprintf(streams.Out, "Privacy: %s\n", visibility)

	if !ws.CreatedOn.IsZero() {
		fmt.Fprintf(streams.Out, "Created: %s\n", ws.CreatedOn.Format("Jan 2, 2006"))
	}

	fmt.Fprintln(streams.Out)
	fmt.Fprintf(streams.Out, "View in browser: %s\n", ws.Links.HTML.Href)

	return nil
}
```

**Step 2: Create members command**

```go
// internal/cmd/workspace/members.go
package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type membersOptions struct {
	streams *iostreams.IOStreams
	slug    string
	limit   int
	json    bool
}

// NewCmdMembers creates the workspace members command
func NewCmdMembers(streams *iostreams.IOStreams) *cobra.Command {
	opts := &membersOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "members <workspace>",
		Short: "List workspace members",
		Long: `List all members of a Bitbucket workspace.

Shows each member's name and their permission level in the workspace.`,
		Example: `  # List members of a workspace
  bb workspace members myworkspace

  # Output as JSON
  bb workspace members myworkspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.slug = args[0]
			return runMembers(cmd.Context(), opts)
		},
	}

	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 30, "Maximum number of members to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runMembers(ctx context.Context, opts *membersOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	listOpts := &api.WorkspaceMemberListOptions{
		Limit: opts.limit,
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := client.ListWorkspaceMembers(ctx, opts.slug, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list workspace members: %w", err)
	}

	if len(result.Values) == 0 {
		opts.streams.Info("No members found in workspace %s", opts.slug)
		return nil
	}

	if opts.json {
		return outputMembersJSON(opts.streams, result.Values)
	}

	return outputMembersTable(opts.streams, result.Values)
}

func outputMembersJSON(streams *iostreams.IOStreams, members []api.WorkspaceMembership) error {
	output := make([]map[string]interface{}, len(members))
	for i, m := range members {
		entry := map[string]interface{}{
			"permission": m.Permission,
		}
		if m.User != nil {
			entry["username"] = m.User.Username
			entry["display_name"] = m.User.DisplayName
			entry["uuid"] = m.User.UUID
			entry["account_id"] = m.User.AccountID
		}
		output[i] = entry
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputMembersTable(streams *iostreams.IOStreams, members []api.WorkspaceMembership) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	header := "USERNAME\tNAME\tROLE"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	for _, m := range members {
		username := ""
		displayName := ""
		if m.User != nil {
			username = m.User.Username
			displayName = m.User.DisplayName
		}
		role := formatRole(streams, m.Permission)
		fmt.Fprintf(w, "%s\t%s\t%s\n", username, displayName, role)
	}

	return w.Flush()
}
```

**Step 3: Add missing import to view.go**

Note: view.go needs the api import. Update the import section:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/browser"
	"github.com/rbansal42/bb/internal/iostreams"
)
```

**Step 4: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/cmd/workspace/
git commit -m "feat(cmd): add workspace view and members commands"
```

---

## Task 9: Project Commands

**Files:**
- Create: `internal/cmd/project/project.go`
- Create: `internal/cmd/project/shared.go`
- Create: `internal/cmd/project/list.go`
- Create: `internal/cmd/project/view.go`
- Create: `internal/cmd/project/create.go`

**Step 1: Create project parent command**

```go
// internal/cmd/project/project.go
package project

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdProject creates the project command and its subcommands
func NewCmdProject(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project <command>",
		Short: "Work with Bitbucket projects",
		Long: `View and manage Bitbucket projects.

Projects are containers for organizing related repositories within a workspace.`,
		Example: `  # List projects in a workspace
  bb project list --workspace myworkspace

  # View a specific project
  bb project view PROJ --workspace myworkspace

  # Create a new project
  bb project create --key NEWPROJ --name "New Project" --workspace myworkspace`,
		Aliases: []string{"proj"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdCreate(streams))

	return cmd
}
```

**Step 2: Create shared utilities**

```go
// internal/cmd/project/shared.go
package project

import (
	"encoding/json"
	"fmt"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
)

// getAPIClient creates an authenticated API client
func getAPIClient() (*api.Client, error) {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load hosts config: %w", err)
	}

	user := hosts.GetActiveUser(config.DefaultHost)
	if user == "" {
		return nil, fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
	}

	tokenData, _, err := config.GetTokenFromEnvOrKeyring(config.DefaultHost, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	token := tokenData
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		token = tokenResp.AccessToken
	}

	return api.NewClient(api.WithToken(token)), nil
}
```

**Step 3: Create list command**

```go
// internal/cmd/project/list.go
package project

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type listOptions struct {
	streams   *iostreams.IOStreams
	workspace string
	limit     int
	json      bool
}

// NewCmdList creates the project list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &listOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects in a workspace",
		Long: `List all projects in a Bitbucket workspace.

Projects help organize related repositories within a workspace.`,
		Example: `  # List projects in a workspace
  bb project list --workspace myworkspace

  # Output as JSON
  bb project list --workspace myworkspace --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.workspace == "" {
				return fmt.Errorf("workspace is required. Use --workspace or -w to specify")
			}
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 30, "Maximum number of projects to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	listOpts := &api.ProjectListOptions{
		Limit: opts.limit,
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := client.ListProjects(ctx, opts.workspace, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(result.Values) == 0 {
		opts.streams.Info("No projects found in workspace %s", opts.workspace)
		return nil
	}

	if opts.json {
		return outputListJSON(opts.streams, result.Values)
	}

	return outputListTable(opts.streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, projects []api.ProjectFull) error {
	output := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		output[i] = map[string]interface{}{
			"key":         p.Key,
			"name":        p.Name,
			"uuid":        p.UUID,
			"description": p.Description,
			"is_private":  p.IsPrivate,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputListTable(streams *iostreams.IOStreams, projects []api.ProjectFull) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	header := "KEY\tNAME\tDESCRIPTION\tVISIBILITY"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	for _, p := range projects {
		desc := truncateString(p.Description, 40)
		visibility := formatVisibility(streams, p.IsPrivate)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Key, p.Name, desc, visibility)
	}

	return w.Flush()
}

func formatVisibility(streams *iostreams.IOStreams, isPrivate bool) string {
	if isPrivate {
		if streams.ColorEnabled() {
			return iostreams.Yellow + "private" + iostreams.Reset
		}
		return "private"
	}

	if streams.ColorEnabled() {
		return iostreams.Green + "public" + iostreams.Reset
	}
	return "public"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
```

**Step 4: Create view command**

```go
// internal/cmd/project/view.go
package project

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/browser"
	"github.com/rbansal42/bb/internal/iostreams"
)

type viewOptions struct {
	streams    *iostreams.IOStreams
	workspace  string
	projectKey string
	web        bool
	json       bool
}

// NewCmdView creates the project view command
func NewCmdView(streams *iostreams.IOStreams) *cobra.Command {
	opts := &viewOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "view <project-key>",
		Short: "View a project",
		Long: `Display details of a Bitbucket project.

Shows project information including key, name, description, and links.`,
		Example: `  # View a project
  bb project view PROJ --workspace myworkspace

  # Open project in browser
  bb project view PROJ --workspace myworkspace --web

  # Output as JSON
  bb project view PROJ --workspace myworkspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.workspace == "" {
				return fmt.Errorf("workspace is required. Use --workspace or -w to specify")
			}
			opts.projectKey = args[0]
			return runView(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().BoolVar(&opts.web, "web", false, "Open the project in a web browser")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runView(ctx context.Context, opts *viewOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	project, err := client.GetProject(ctx, opts.workspace, opts.projectKey)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if opts.web {
		if err := browser.Open(project.Links.HTML.Href); err != nil {
			return fmt.Errorf("could not open browser: %w", err)
		}
		opts.streams.Success("Opened %s in your browser", project.Links.HTML.Href)
		return nil
	}

	if opts.json {
		data, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(opts.streams.Out, string(data))
		return nil
	}

	return displayProject(opts.streams, project)
}

func displayProject(streams *iostreams.IOStreams, p *api.ProjectFull) error {
	fmt.Fprintf(streams.Out, "%s (%s)\n\n", p.Name, p.Key)

	if p.Description != "" {
		fmt.Fprintf(streams.Out, "Description: %s\n", p.Description)
	} else {
		fmt.Fprintf(streams.Out, "Description: (no description)\n")
	}

	visibility := "public"
	if p.IsPrivate {
		visibility = "private"
	}
	fmt.Fprintf(streams.Out, "Visibility:  %s\n", visibility)

	fmt.Fprintf(streams.Out, "UUID:        %s\n", p.UUID)

	if !p.CreatedOn.IsZero() {
		fmt.Fprintf(streams.Out, "Created:     %s\n", p.CreatedOn.Format("Jan 2, 2006"))
	}

	if !p.UpdatedOn.IsZero() {
		fmt.Fprintf(streams.Out, "Updated:     %s\n", p.UpdatedOn.Format("Jan 2, 2006"))
	}

	fmt.Fprintln(streams.Out)
	fmt.Fprintf(streams.Out, "View in browser: %s\n", p.Links.HTML.Href)

	return nil
}
```

Note: Add missing api import to view.go:
```go
import (
	...
	"github.com/rbansal42/bb/internal/api"
	...
)
```

**Step 5: Create create command**

```go
// internal/cmd/project/create.go
package project

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type createOptions struct {
	streams     *iostreams.IOStreams
	workspace   string
	key         string
	name        string
	description string
	isPrivate   bool
	json        bool
}

// NewCmdCreate creates the project create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &createOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Long: `Create a new project in a Bitbucket workspace.

A project helps organize related repositories within a workspace.`,
		Example: `  # Create a basic project
  bb project create --key PROJ --name "My Project" --workspace myworkspace

  # Create a private project with description
  bb project create --key PROJ --name "My Project" --description "Project description" --private --workspace myworkspace`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.workspace == "" {
				return fmt.Errorf("workspace is required. Use --workspace or -w to specify")
			}
			if opts.key == "" {
				return fmt.Errorf("project key is required. Use --key or -k to specify")
			}
			if opts.name == "" {
				return fmt.Errorf("project name is required. Use --name or -n to specify")
			}
			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.workspace, "workspace", "w", "", "Workspace slug (required)")
	cmd.Flags().StringVarP(&opts.key, "key", "k", "", "Project key (required, e.g., PROJ)")
	cmd.Flags().StringVarP(&opts.name, "name", "n", "", "Project name (required)")
	cmd.Flags().StringVarP(&opts.description, "description", "d", "", "Project description")
	cmd.Flags().BoolVarP(&opts.isPrivate, "private", "p", false, "Make the project private")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	createOpts := &api.ProjectCreateOptions{
		Key:         opts.key,
		Name:        opts.name,
		Description: opts.description,
		IsPrivate:   opts.isPrivate,
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	project, err := client.CreateProject(ctx, opts.workspace, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	if opts.json {
		data, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(opts.streams.Out, string(data))
		return nil
	}

	opts.streams.Success("Created project %s in workspace %s", project.Key, opts.workspace)
	fmt.Fprintf(opts.streams.Out, "\nView in browser: %s\n", project.Links.HTML.Href)

	return nil
}
```

**Step 6: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 7: Commit**

```bash
git add internal/cmd/project/
git commit -m "feat(cmd): add project commands (list, view, create)"
```

---

## Task 10: Branch Commands

**Files:**
- Create: `internal/cmd/branch/branch.go`
- Create: `internal/cmd/branch/shared.go`
- Create: `internal/cmd/branch/list.go`
- Create: `internal/cmd/branch/create.go`
- Create: `internal/cmd/branch/delete.go`

**Step 1: Create branch parent command**

```go
// internal/cmd/branch/branch.go
package branch

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdBranch creates the branch command and its subcommands
func NewCmdBranch(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch <command>",
		Short: "Work with repository branches",
		Long: `List, create, and delete branches in a repository.

Branches allow parallel development on different features or fixes.`,
		Example: `  # List branches in the current repository
  bb branch list

  # Create a new branch
  bb branch create feature/new-feature --target main

  # Delete a branch
  bb branch delete feature/old-feature`,
		Aliases: []string{"br"},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdDelete(streams))

	return cmd
}
```

**Step 2: Create shared utilities**

```go
// internal/cmd/branch/shared.go
package branch

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/git"
)

// getAPIClient creates an authenticated API client
func getAPIClient() (*api.Client, error) {
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load hosts config: %w", err)
	}

	user := hosts.GetActiveUser(config.DefaultHost)
	if user == "" {
		return nil, fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
	}

	tokenData, _, err := config.GetTokenFromEnvOrKeyring(config.DefaultHost, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	token := tokenData
	if err := json.Unmarshal([]byte(tokenData), &tokenResp); err == nil && tokenResp.AccessToken != "" {
		token = tokenResp.AccessToken
	}

	return api.NewClient(api.WithToken(token)), nil
}

// parseRepository parses a repository string or detects from git remote
func parseRepository(repoFlag string) (workspace, repoSlug string, err error) {
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid repository format: %s (expected workspace/repo)", repoFlag)
		}
		if parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid repository format: %s (workspace and repo cannot be empty)", repoFlag)
		}
		return parts[0], parts[1], nil
	}

	remote, err := git.GetDefaultRemote()
	if err != nil {
		return "", "", fmt.Errorf("could not detect repository: %w\nUse --repo WORKSPACE/REPO to specify", err)
	}

	return remote.Workspace, remote.RepoSlug, nil
}
```

**Step 3: Create list command**

```go
// internal/cmd/branch/list.go
package branch

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type listOptions struct {
	streams   *iostreams.IOStreams
	repo      string
	workspace string
	repoSlug  string
	limit     int
	json      bool
}

// NewCmdList creates the branch list command
func NewCmdList(streams *iostreams.IOStreams) *cobra.Command {
	opts := &listOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branches in a repository",
		Long: `List all branches in a Bitbucket repository.

By default, uses the repository detected from the current directory's git remote.`,
		Example: `  # List branches in the current repository
  bb branch list

  # List branches in a specific repository
  bb branch list --repo myworkspace/myrepo

  # Output as JSON
  bb branch list --json`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			opts.workspace, opts.repoSlug, err = parseRepository(opts.repo)
			if err != nil {
				return err
			}
			return runList(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")
	cmd.Flags().IntVarP(&opts.limit, "limit", "l", 30, "Maximum number of branches to list")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runList(ctx context.Context, opts *listOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	listOpts := &api.BranchListOptions{
		Limit: opts.limit,
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := client.ListBranches(ctx, opts.workspace, opts.repoSlug, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(result.Values) == 0 {
		opts.streams.Info("No branches found in %s/%s", opts.workspace, opts.repoSlug)
		return nil
	}

	if opts.json {
		return outputListJSON(opts.streams, result.Values)
	}

	return outputListTable(opts.streams, result.Values)
}

func outputListJSON(streams *iostreams.IOStreams, branches []api.BranchFull) error {
	output := make([]map[string]interface{}, len(branches))
	for i, b := range branches {
		entry := map[string]interface{}{
			"name": b.Name,
			"type": b.Type,
		}
		if b.Target != nil {
			entry["commit"] = b.Target.Hash
			entry["message"] = b.Target.Message
		}
		output[i] = entry
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Fprintln(streams.Out, string(data))
	return nil
}

func outputListTable(streams *iostreams.IOStreams, branches []api.BranchFull) error {
	w := tabwriter.NewWriter(streams.Out, 0, 0, 2, ' ', 0)

	header := "NAME\tCOMMIT\tMESSAGE"
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}

	for _, b := range branches {
		commit := ""
		message := ""
		if b.Target != nil {
			if len(b.Target.Hash) > 7 {
				commit = b.Target.Hash[:7]
			} else {
				commit = b.Target.Hash
			}
			message = truncateString(b.Target.Message, 50)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", b.Name, commit, message)
	}

	return w.Flush()
}

func truncateString(s string, maxLen int) string {
	// Remove newlines
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
```

Note: Add strings import to list.go.

**Step 4: Create create command**

```go
// internal/cmd/branch/create.go
package branch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/iostreams"
)

type createOptions struct {
	streams    *iostreams.IOStreams
	repo       string
	workspace  string
	repoSlug   string
	branchName string
	target     string
	json       bool
}

// NewCmdCreate creates the branch create command
func NewCmdCreate(streams *iostreams.IOStreams) *cobra.Command {
	opts := &createOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "create <branch-name>",
		Short: "Create a new branch",
		Long: `Create a new branch in a Bitbucket repository.

The target can be a branch name, tag, or commit hash.`,
		Example: `  # Create a branch from main
  bb branch create feature/new-feature --target main

  # Create a branch from a specific commit
  bb branch create bugfix/fix-123 --target abc123def

  # Create in a specific repository
  bb branch create feature/test --target main --repo myworkspace/myrepo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.branchName = args[0]

			var err error
			opts.workspace, opts.repoSlug, err = parseRepository(opts.repo)
			if err != nil {
				return err
			}

			if opts.target == "" {
				return fmt.Errorf("target is required. Use --target to specify a branch, tag, or commit")
			}

			return runCreate(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")
	cmd.Flags().StringVarP(&opts.target, "target", "t", "", "Target branch, tag, or commit hash (required)")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")

	return cmd
}

func runCreate(ctx context.Context, opts *createOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// First, resolve the target to a commit hash if it's a branch name
	targetHash := opts.target

	// Try to get the branch to resolve to commit hash
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	branch, err := client.GetBranch(ctx, opts.workspace, opts.repoSlug, opts.target)
	if err == nil && branch.Target != nil {
		targetHash = branch.Target.Hash
	}
	// If branch not found, assume target is already a commit hash

	createOpts := &api.BranchCreateOptions{
		Name:   opts.branchName,
		Target: struct{ Hash string `json:"hash"` }{Hash: targetHash},
	}

	newBranch, err := client.CreateBranch(ctx, opts.workspace, opts.repoSlug, createOpts)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	if opts.json {
		data, err := json.MarshalIndent(newBranch, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(opts.streams.Out, string(data))
		return nil
	}

	opts.streams.Success("Created branch %s in %s/%s", newBranch.Name, opts.workspace, opts.repoSlug)
	if newBranch.Target != nil {
		fmt.Fprintf(opts.streams.Out, "Target commit: %s\n", newBranch.Target.Hash)
	}

	return nil
}
```

**Step 5: Create delete command**

```go
// internal/cmd/branch/delete.go
package branch

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

type deleteOptions struct {
	streams    *iostreams.IOStreams
	repo       string
	workspace  string
	repoSlug   string
	branchName string
	force      bool
}

// NewCmdDelete creates the branch delete command
func NewCmdDelete(streams *iostreams.IOStreams) *cobra.Command {
	opts := &deleteOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "delete <branch-name>",
		Short: "Delete a branch",
		Long: `Delete a branch from a Bitbucket repository.

By default, you will be prompted to confirm the deletion.
Use --force to skip the confirmation.`,
		Example: `  # Delete a branch (with confirmation)
  bb branch delete feature/old-feature

  # Delete without confirmation
  bb branch delete feature/old-feature --force

  # Delete in a specific repository
  bb branch delete feature/test --repo myworkspace/myrepo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.branchName = args[0]

			var err error
			opts.workspace, opts.repoSlug, err = parseRepository(opts.repo)
			if err != nil {
				return err
			}

			return runDelete(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(ctx context.Context, opts *deleteOptions) error {
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	// Confirm deletion unless --force is used
	if !opts.force {
		if opts.streams.IsTTY() {
			fmt.Fprintf(opts.streams.Out, "Delete branch %s from %s/%s? [y/N]: ",
				opts.branchName, opts.workspace, opts.repoSlug)

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				opts.streams.Info("Deletion cancelled")
				return nil
			}
		} else {
			return fmt.Errorf("cannot confirm deletion in non-interactive mode. Use --force to skip confirmation")
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = client.DeleteBranch(ctx, opts.workspace, opts.repoSlug, opts.branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	opts.streams.Success("Deleted branch %s from %s/%s", opts.branchName, opts.workspace, opts.repoSlug)

	return nil
}
```

**Step 6: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 7: Commit**

```bash
git add internal/cmd/branch/
git commit -m "feat(cmd): add branch commands (list, create, delete)"
```

---

## Task 11: Root Command Integration

**Files:**
- Modify: `internal/cmd/root.go`

**Step 1: Add imports and command registration**

Add these imports:
```go
"github.com/rbansal42/bb/internal/cmd/branch"
"github.com/rbansal42/bb/internal/cmd/project"
"github.com/rbansal42/bb/internal/cmd/workspace"
```

Add these commands in the `init()` function:
```go
rootCmd.AddCommand(branch.NewCmdBranch(GetStreams()))
rootCmd.AddCommand(project.NewCmdProject(GetStreams()))
rootCmd.AddCommand(workspace.NewCmdWorkspace(GetStreams()))
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: No errors

**Step 3: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/cmd/root.go
git commit -m "feat(cmd): register workspace, project, and branch commands"
```

---

## Task 12: Final Verification and Combined Commit

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Build and verify commands**

Run:
```bash
go build -o bb ./cmd/bb
./bb workspace --help
./bb project --help
./bb branch --help
```
Expected: Help text displays for all commands

**Step 3: Verify API tests**

Run: `go test ./internal/api/... -v`
Expected: All tests pass including new workspace, project, and branch tests

---

## Summary

This plan implements Phase 5a with:

**API Layer (6 files):**
- `internal/api/workspaces.go` - Workspace types and 3 methods
- `internal/api/workspaces_test.go` - Tests
- `internal/api/projects.go` - Project types and 5 methods
- `internal/api/projects_test.go` - Tests
- `internal/api/branches.go` - Branch types and 4 methods
- `internal/api/branches_test.go` - Tests

**Command Layer (13 files):**
- `internal/cmd/workspace/` - 4 files (workspace.go, shared.go, list.go, view.go, members.go)
- `internal/cmd/project/` - 5 files (project.go, shared.go, list.go, view.go, create.go)
- `internal/cmd/branch/` - 5 files (branch.go, shared.go, list.go, create.go, delete.go)

**Integration (1 file):**
- `internal/cmd/root.go` - Register new commands

**Total: 9 commands implemented**
- `bb workspace list/view/members`
- `bb project list/view/create`
- `bb branch list/create/delete`
