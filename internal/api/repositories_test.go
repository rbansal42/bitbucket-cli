package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListRepositories(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		opts          *RepositoryListOptions
		expectedURL   string
		expectedQuery map[string]string
		response      string
		statusCode    int
		wantErr       bool
		wantCount     int
	}{
		{
			name:        "basic list without options",
			workspace:   "myworkspace",
			opts:        nil,
			expectedURL: "/repositories/myworkspace",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"uuid": "{repo-1}", "name": "repo1", "full_name": "myworkspace/repo1", "slug": "repo1"},
					{"uuid": "{repo-2}", "name": "repo2", "full_name": "myworkspace/repo2", "slug": "repo2"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with role filter",
			workspace:   "myworkspace",
			opts:        &RepositoryListOptions{Role: "owner"},
			expectedURL: "/repositories/myworkspace",
			expectedQuery: map[string]string{"role": "owner"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"uuid": "{repo-1}", "name": "owned-repo", "full_name": "myworkspace/owned-repo"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with pagination",
			workspace:   "myworkspace",
			opts:        &RepositoryListOptions{Page: 2, Limit: 5},
			expectedURL: "/repositories/myworkspace",
			expectedQuery: map[string]string{"page": "2", "pagelen": "5"},
			response: `{
				"size": 10,
				"page": 2,
				"pagelen": 5,
				"next": "https://api.bitbucket.org/2.0/repositories/myworkspace?page=3",
				"values": []
			}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:        "list with sort",
			workspace:   "myworkspace",
			opts:        &RepositoryListOptions{Sort: "-updated_on"},
			expectedURL: "/repositories/myworkspace",
			expectedQuery: map[string]string{"sort": "-updated_on"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"uuid": "{repo-1}", "name": "recent-repo"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with query filter",
			workspace:   "myworkspace",
			opts:        &RepositoryListOptions{Query: "name~\"test\""},
			expectedURL: "/repositories/myworkspace",
			expectedQuery: map[string]string{"q": "name~\"test\""},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"uuid": "{repo-1}", "name": "test-repo"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "handles 401 unauthorized",
			workspace:   "myworkspace",
			opts:        nil,
			response:    `{"error": {"message": "Unauthorized", "detail": "Authentication required"}}`,
			statusCode:  http.StatusUnauthorized,
			wantErr:     true,
		},
		{
			name:        "handles 404 workspace not found",
			workspace:   "nonexistent",
			opts:        nil,
			response:    `{"error": {"message": "Workspace not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
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

			result, err := client.ListRepositories(context.Background(), tt.workspace, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL path
			if tt.expectedURL != "" && !strings.HasSuffix(receivedReq.URL.Path, tt.expectedURL) {
				t.Errorf("expected URL path to end with %q, got %q", tt.expectedURL, receivedReq.URL.Path)
			}

			// Verify query parameters
			for key, expected := range tt.expectedQuery {
				actual := receivedReq.URL.Query().Get(key)
				if actual != expected {
					t.Errorf("expected query param %s=%q, got %q", key, expected, actual)
				}
			}

			// Verify result
			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d repositories, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetRepository(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		response   string
		statusCode int
		wantErr    bool
		wantName   string
	}{
		{
			name:      "successfully get repository",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			response: `{
				"uuid": "{repo-uuid}",
				"name": "myrepo",
				"slug": "myrepo",
				"full_name": "myworkspace/myrepo",
				"description": "A test repository",
				"is_private": true,
				"fork_policy": "allow_forks",
				"language": "go",
				"size": 1024000,
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-15T12:00:00Z",
				"owner": {
					"display_name": "Test User",
					"uuid": "{user-uuid}"
				},
				"workspace": {
					"uuid": "{ws-uuid}",
					"slug": "myworkspace",
					"name": "My Workspace"
				},
				"mainbranch": {
					"name": "main",
					"type": "branch"
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo"},
					"html": {"href": "https://bitbucket.org/myworkspace/myrepo"},
					"clone": [
						{"href": "https://bitbucket.org/myworkspace/myrepo.git", "name": "https"},
						{"href": "git@bitbucket.org:myworkspace/myrepo.git", "name": "ssh"}
					],
					"avatar": {"href": "https://bitbucket.org/myworkspace/myrepo/avatar"}
				}
			}`,
			statusCode: http.StatusOK,
			wantName:   "myrepo",
		},
		{
			name:       "repository not found",
			workspace:  "myworkspace",
			repoSlug:   "nonexistent",
			response:   `{"error": {"message": "Repository not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "unauthorized access",
			workspace:  "private-workspace",
			repoSlug:   "private-repo",
			response:   `{"error": {"message": "Unauthorized", "detail": "You do not have access to this repository"}}`,
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

			result, err := client.GetRepository(context.Background(), tt.workspace, tt.repoSlug)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL contains workspace and repo
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path to contain %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify response parsing
			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
			}
		})
	}
}

func TestCreateRepository(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		opts         *RepositoryCreateOptions
		expectedBody map[string]interface{}
		response     string
		statusCode   int
		wantErr      bool
		wantName     string
	}{
		{
			name:      "basic repository creation",
			workspace: "myworkspace",
			opts: &RepositoryCreateOptions{
				Name:        "new-repo",
				Description: "A new repository",
				IsPrivate:   true,
			},
			response: `{
				"uuid": "{new-repo-uuid}",
				"name": "new-repo",
				"slug": "new-repo",
				"full_name": "myworkspace/new-repo",
				"description": "A new repository",
				"is_private": true,
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantName:   "new-repo",
		},
		{
			name:      "repository with all options",
			workspace: "myworkspace",
			opts: &RepositoryCreateOptions{
				Name:        "full-repo",
				Description: "Repository with all options",
				IsPrivate:   false,
				ForkPolicy:  "no_public_forks",
				Language:    "python",
				Project:     &Project{Key: "PROJ"},
				HasIssues:   true,
				HasWiki:     true,
			},
			response: `{
				"uuid": "{full-repo-uuid}",
				"name": "full-repo",
				"slug": "full-repo",
				"full_name": "myworkspace/full-repo",
				"is_private": false,
				"fork_policy": "no_public_forks",
				"language": "python",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantName:   "full-repo",
		},
		{
			name:      "repository creation fails - already exists",
			workspace: "myworkspace",
			opts: &RepositoryCreateOptions{
				Name:      "existing-repo",
				IsPrivate: true,
			},
			response:   `{"error": {"message": "Repository with this name already exists"}}`,
			statusCode: http.StatusConflict,
			wantErr:    true,
		},
		{
			name:      "repository creation fails - validation error",
			workspace: "myworkspace",
			opts: &RepositoryCreateOptions{
				Name:      "invalid repo name",
				IsPrivate: true,
			},
			response:   `{"error": {"message": "Validation error", "fields": {"name": "Name contains invalid characters"}}}`,
			statusCode: http.StatusBadRequest,
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

			result, err := client.CreateRepository(context.Background(), tt.workspace, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify HTTP method is POST
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			// Verify URL contains workspace and repo name
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.opts.Name
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify Content-Type
			if ct := receivedReq.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			// Verify request body structure
			var body map[string]interface{}
			if err := json.Unmarshal(receivedBody, &body); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			// Verify scm is set to "git"
			if body["scm"] != "git" {
				t.Errorf("expected scm to be 'git', got %v", body["scm"])
			}

			// Verify name
			if body["name"] != tt.opts.Name {
				t.Errorf("expected name %q in body, got %v", tt.opts.Name, body["name"])
			}

			// Verify is_private
			if body["is_private"] != tt.opts.IsPrivate {
				t.Errorf("expected is_private %v in body, got %v", tt.opts.IsPrivate, body["is_private"])
			}

			// Verify project if provided
			if tt.opts.Project != nil && tt.opts.Project.Key != "" {
				project, ok := body["project"].(map[string]interface{})
				if !ok {
					t.Error("expected project object in body")
				} else if project["key"] != tt.opts.Project.Key {
					t.Errorf("expected project key %q, got %v", tt.opts.Project.Key, project["key"])
				}
			}

			// Verify result
			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
			}
		})
	}
}

func TestDeleteRepository(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful deletion",
			workspace:  "myworkspace",
			repoSlug:   "repo-to-delete",
			statusCode: http.StatusNoContent,
			response:   "",
			wantErr:    false,
		},
		{
			name:       "repository not found",
			workspace:  "myworkspace",
			repoSlug:   "nonexistent",
			statusCode: http.StatusNotFound,
			response:   `{"error": {"message": "Repository not found"}}`,
			wantErr:    true,
		},
		{
			name:       "unauthorized deletion",
			workspace:  "other-workspace",
			repoSlug:   "protected-repo",
			statusCode: http.StatusUnauthorized,
			response:   `{"error": {"message": "Unauthorized", "detail": "You do not have permission to delete this repository"}}`,
			wantErr:    true,
		},
		{
			name:       "forbidden deletion",
			workspace:  "myworkspace",
			repoSlug:   "admin-only-repo",
			statusCode: http.StatusForbidden,
			response:   `{"error": {"message": "Forbidden", "detail": "Only repository admins can delete"}}`,
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

			err := client.DeleteRepository(context.Background(), tt.workspace, tt.repoSlug)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify HTTP method is DELETE
			if receivedReq.Method != http.MethodDelete {
				t.Errorf("expected DELETE method, got %s", receivedReq.Method)
			}

			// Verify URL contains workspace and repo
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}
		})
	}
}

func TestForkRepository(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		destWorkspace string
		forkName      string
		response      string
		statusCode    int
		wantErr       bool
		wantName      string
	}{
		{
			name:          "basic fork",
			workspace:     "source-workspace",
			repoSlug:      "source-repo",
			destWorkspace: "",
			forkName:      "",
			response: `{
				"uuid": "{fork-uuid}",
				"name": "source-repo",
				"slug": "source-repo",
				"full_name": "my-workspace/source-repo",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantName:   "source-repo",
		},
		{
			name:          "fork with custom name",
			workspace:     "source-workspace",
			repoSlug:      "source-repo",
			destWorkspace: "",
			forkName:      "my-fork",
			response: `{
				"uuid": "{fork-uuid}",
				"name": "my-fork",
				"slug": "my-fork",
				"full_name": "my-workspace/my-fork",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantName:   "my-fork",
		},
		{
			name:          "fork to different workspace",
			workspace:     "source-workspace",
			repoSlug:      "source-repo",
			destWorkspace: "dest-workspace",
			forkName:      "forked-repo",
			response: `{
				"uuid": "{fork-uuid}",
				"name": "forked-repo",
				"slug": "forked-repo",
				"full_name": "dest-workspace/forked-repo",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantName:   "forked-repo",
		},
		{
			name:          "fork already exists",
			workspace:     "source-workspace",
			repoSlug:      "source-repo",
			destWorkspace: "",
			forkName:      "existing-fork",
			response:      `{"error": {"message": "Repository with this name already exists in the target workspace"}}`,
			statusCode:    http.StatusConflict,
			wantErr:       true,
		},
		{
			name:          "source repository not found",
			workspace:     "source-workspace",
			repoSlug:      "nonexistent",
			destWorkspace: "",
			forkName:      "",
			response:      `{"error": {"message": "Repository not found"}}`,
			statusCode:    http.StatusNotFound,
			wantErr:       true,
		},
		{
			name:          "forking not allowed",
			workspace:     "source-workspace",
			repoSlug:      "no-fork-repo",
			destWorkspace: "",
			forkName:      "",
			response:      `{"error": {"message": "Forking is not allowed for this repository"}}`,
			statusCode:    http.StatusForbidden,
			wantErr:       true,
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

			result, err := client.ForkRepository(context.Background(), tt.workspace, tt.repoSlug, tt.destWorkspace, tt.forkName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify HTTP method is POST
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			// Verify URL contains fork endpoint
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/forks"
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify request body structure
			var body map[string]interface{}
			if err := json.Unmarshal(receivedBody, &body); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			// Verify name if provided
			if tt.forkName != "" {
				if body["name"] != tt.forkName {
					t.Errorf("expected name %q in body, got %v", tt.forkName, body["name"])
				}
			}

			// Verify workspace if provided
			if tt.destWorkspace != "" {
				workspace, ok := body["workspace"].(map[string]interface{})
				if !ok {
					t.Error("expected workspace object in body")
				} else if workspace["slug"] != tt.destWorkspace {
					t.Errorf("expected workspace slug %q, got %v", tt.destWorkspace, workspace["slug"])
				}
			}

			// Verify result
			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
			}
		})
	}
}

func TestRepositoryParsing(t *testing.T) {
	// Test comprehensive repository response parsing with all fields
	responseJSON := `{
		"uuid": "{complete-repo-uuid}",
		"name": "complete-repo",
		"slug": "complete-repo",
		"full_name": "myworkspace/complete-repo",
		"description": "A complete repository for testing",
		"is_private": true,
		"fork_policy": "no_forks",
		"language": "go",
		"size": 2048576,
		"created_on": "2024-01-15T10:30:00Z",
		"updated_on": "2024-02-20T14:45:00Z",
		"owner": {
			"uuid": "{owner-uuid}",
			"username": "owner",
			"display_name": "Repository Owner",
			"account_id": "owner123"
		},
		"project": {
			"uuid": "{project-uuid}",
			"key": "PROJ",
			"name": "Project Name"
		},
		"workspace": {
			"uuid": "{ws-uuid}",
			"slug": "myworkspace",
			"name": "My Workspace"
		},
		"mainbranch": {
			"name": "main",
			"type": "branch"
		},
		"links": {
			"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/complete-repo"},
			"html": {"href": "https://bitbucket.org/myworkspace/complete-repo"},
			"clone": [
				{"href": "https://bitbucket.org/myworkspace/complete-repo.git", "name": "https"},
				{"href": "git@bitbucket.org:myworkspace/complete-repo.git", "name": "ssh"}
			],
			"avatar": {"href": "https://bitbucket.org/myworkspace/complete-repo/avatar/32/"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	repo, err := client.GetRepository(context.Background(), "myworkspace", "complete-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are parsed correctly
	if repo.UUID != "{complete-repo-uuid}" {
		t.Errorf("expected UUID '{complete-repo-uuid}', got %q", repo.UUID)
	}

	if repo.Name != "complete-repo" {
		t.Errorf("expected name 'complete-repo', got %q", repo.Name)
	}

	if repo.Slug != "complete-repo" {
		t.Errorf("expected slug 'complete-repo', got %q", repo.Slug)
	}

	if repo.FullName != "myworkspace/complete-repo" {
		t.Errorf("expected full_name 'myworkspace/complete-repo', got %q", repo.FullName)
	}

	if repo.Description != "A complete repository for testing" {
		t.Errorf("expected description 'A complete repository for testing', got %q", repo.Description)
	}

	if !repo.IsPrivate {
		t.Error("expected IsPrivate to be true")
	}

	if repo.ForkPolicy != "no_forks" {
		t.Errorf("expected fork_policy 'no_forks', got %q", repo.ForkPolicy)
	}

	if repo.Language != "go" {
		t.Errorf("expected language 'go', got %q", repo.Language)
	}

	if repo.Size != 2048576 {
		t.Errorf("expected size 2048576, got %d", repo.Size)
	}

	// Verify owner
	if repo.Owner == nil {
		t.Fatal("expected Owner to not be nil")
	}
	if repo.Owner.DisplayName != "Repository Owner" {
		t.Errorf("expected owner display_name 'Repository Owner', got %q", repo.Owner.DisplayName)
	}

	// Verify project
	if repo.Project == nil {
		t.Fatal("expected Project to not be nil")
	}
	if repo.Project.Key != "PROJ" {
		t.Errorf("expected project key 'PROJ', got %q", repo.Project.Key)
	}

	// Verify workspace
	if repo.Workspace == nil {
		t.Fatal("expected Workspace to not be nil")
	}
	if repo.Workspace.Slug != "myworkspace" {
		t.Errorf("expected workspace slug 'myworkspace', got %q", repo.Workspace.Slug)
	}

	// Verify main branch
	if repo.MainBranch == nil {
		t.Fatal("expected MainBranch to not be nil")
	}
	if repo.MainBranch.Name != "main" {
		t.Errorf("expected mainbranch name 'main', got %q", repo.MainBranch.Name)
	}

	// Verify clone links
	if len(repo.Links.Clone) != 2 {
		t.Errorf("expected 2 clone links, got %d", len(repo.Links.Clone))
	}

	// Find HTTPS clone link
	var httpsFound, sshFound bool
	for _, clone := range repo.Links.Clone {
		if clone.Name == "https" {
			httpsFound = true
			if !strings.Contains(clone.Href, "https://") {
				t.Errorf("expected HTTPS clone URL to contain https://, got %q", clone.Href)
			}
		}
		if clone.Name == "ssh" {
			sshFound = true
			if !strings.Contains(clone.Href, "git@") {
				t.Errorf("expected SSH clone URL to contain git@, got %q", clone.Href)
			}
		}
	}
	if !httpsFound {
		t.Error("expected to find HTTPS clone link")
	}
	if !sshFound {
		t.Error("expected to find SSH clone link")
	}

	// Verify time parsing
	expectedCreated := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !repo.CreatedOn.Equal(expectedCreated) {
		t.Errorf("expected created_on %v, got %v", expectedCreated, repo.CreatedOn)
	}

	expectedUpdated := time.Date(2024, 2, 20, 14, 45, 0, 0, time.UTC)
	if !repo.UpdatedOn.Equal(expectedUpdated) {
		t.Errorf("expected updated_on %v, got %v", expectedUpdated, repo.UpdatedOn)
	}
}

func TestRepositoryErrorHandling(t *testing.T) {
	tests := []struct {
		name            string
		statusCode      int
		response        string
		wantStatusCode  int
		wantMessage     string
	}{
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			response:       `{"error": {"message": "Unauthorized", "detail": "Invalid token"}}`,
			wantStatusCode: http.StatusUnauthorized,
			wantMessage:    "Unauthorized",
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			response:       `{"error": {"message": "Repository not found"}}`,
			wantStatusCode: http.StatusNotFound,
			wantMessage:    "Repository not found",
		},
		{
			name:           "409 Conflict",
			statusCode:     http.StatusConflict,
			response:       `{"error": {"message": "Repository already exists"}}`,
			wantStatusCode: http.StatusConflict,
			wantMessage:    "Repository already exists",
		},
		{
			name:           "400 Bad Request with fields",
			statusCode:     http.StatusBadRequest,
			response:       `{"error": {"message": "Validation error", "fields": {"name": "Invalid name"}}}`,
			wantStatusCode: http.StatusBadRequest,
			wantMessage:    "Validation error",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			response:       `{"error": {"message": "Internal server error"}}`,
			wantStatusCode: http.StatusInternalServerError,
			wantMessage:    "Internal server error",
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

			_, err := client.GetRepository(context.Background(), "workspace", "repo")

			if err == nil {
				t.Fatal("expected error but got nil")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected error to be *APIError, got %T", err)
			}

			if apiErr.StatusCode != tt.wantStatusCode {
				t.Errorf("expected status code %d, got %d", tt.wantStatusCode, apiErr.StatusCode)
			}

			if apiErr.Message != tt.wantMessage {
				t.Errorf("expected message %q, got %q", tt.wantMessage, apiErr.Message)
			}
		})
	}
}

func TestListRepositoriesPagination(t *testing.T) {
	// Test that pagination response is properly parsed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 100,
			"page": 2,
			"pagelen": 10,
			"next": "https://api.bitbucket.org/2.0/repositories/myworkspace?page=3",
			"previous": "https://api.bitbucket.org/2.0/repositories/myworkspace?page=1",
			"values": [
				{"uuid": "{repo-1}", "name": "repo1"},
				{"uuid": "{repo-2}", "name": "repo2"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListRepositories(context.Background(), "myworkspace", &RepositoryListOptions{Page: 2, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Size != 100 {
		t.Errorf("expected size 100, got %d", result.Size)
	}

	if result.Page != 2 {
		t.Errorf("expected page 2, got %d", result.Page)
	}

	if result.PageLen != 10 {
		t.Errorf("expected pagelen 10, got %d", result.PageLen)
	}

	if result.Next == "" {
		t.Error("expected next URL to be set")
	}

	if result.Previous == "" {
		t.Error("expected previous URL to be set")
	}

	if len(result.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(result.Values))
	}
}

func TestCreateRepositoryRequiredFields(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"uuid": "{uuid}",
			"name": "minimal-repo",
			"slug": "minimal-repo",
			"full_name": "workspace/minimal-repo",
			"is_private": false,
			"created_on": "2024-01-01T00:00:00Z",
			"updated_on": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

	// Create with minimal options
	opts := &RepositoryCreateOptions{
		Name:      "minimal-repo",
		IsPrivate: false,
	}

	_, err := client.CreateRepository(context.Background(), "workspace", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify required fields are present
	var body map[string]interface{}
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}

	// scm should always be "git"
	if body["scm"] != "git" {
		t.Errorf("expected scm to be 'git', got %v", body["scm"])
	}

	// name should be present
	if body["name"] != "minimal-repo" {
		t.Errorf("expected name 'minimal-repo', got %v", body["name"])
	}

	// is_private should be present (even if false)
	if _, exists := body["is_private"]; !exists {
		t.Error("expected is_private to be present in body")
	}
}
