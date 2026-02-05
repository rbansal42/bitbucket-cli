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
		name          string
		workspace     string
		repoSlug      string
		opts          *BranchListOptions
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
			repoSlug:    "myrepo",
			opts:        nil,
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"name": "main",
						"type": "branch",
						"target": {
							"hash": "abc123",
							"type": "commit",
							"message": "Initial commit"
						},
						"links": {
							"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches/main"},
							"commits": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commits/main"},
							"html": {"href": "https://bitbucket.org/myworkspace/myrepo/branch/main"}
						}
					},
					{
						"name": "develop",
						"type": "branch",
						"target": {
							"hash": "def456",
							"type": "commit",
							"message": "Add feature"
						},
						"links": {
							"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches/develop"},
							"commits": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commits/develop"},
							"html": {"href": "https://bitbucket.org/myworkspace/myrepo/branch/develop"}
						}
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with pagination",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &BranchListOptions{Page: 2, Limit: 5},
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches",
			expectedQuery: map[string]string{"page": "2", "pagelen": "5"},
			response: `{
				"size": 15,
				"page": 2,
				"pagelen": 5,
				"next": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches?page=3",
				"previous": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches?page=1",
				"values": [
					{"name": "feature-1", "type": "branch", "target": {"hash": "111111", "type": "commit"}},
					{"name": "feature-2", "type": "branch", "target": {"hash": "222222", "type": "commit"}},
					{"name": "feature-3", "type": "branch", "target": {"hash": "333333", "type": "commit"}},
					{"name": "feature-4", "type": "branch", "target": {"hash": "444444", "type": "commit"}},
					{"name": "feature-5", "type": "branch", "target": {"hash": "555555", "type": "commit"}}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  5,
		},
		{
			name:        "repository not found",
			workspace:   "myworkspace",
			repoSlug:    "nonexistent",
			opts:        nil,
			expectedURL: "/repositories/myworkspace/nonexistent/refs/branches",
			response:    `{"error": {"message": "Repository not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "list with sort",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &BranchListOptions{Sort: "-name"},
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches",
			expectedQuery: map[string]string{"sort": "-name"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"name": "z-branch", "type": "branch", "target": {"hash": "zzz", "type": "commit"}}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with query filter",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &BranchListOptions{Query: "name~\"feature\""},
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches",
			expectedQuery: map[string]string{"q": "name~\"feature\""},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"name": "feature-branch", "type": "branch", "target": {"hash": "feat123", "type": "commit"}}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
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

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify result
			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d branches, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetBranch(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		branchName  string
		expectedURL string
		response    string
		statusCode  int
		wantErr     bool
		wantName    string
	}{
		{
			name:        "success",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "main",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/main",
			response: `{
				"name": "main",
				"type": "branch",
				"target": {
					"hash": "abc123def456",
					"type": "commit",
					"message": "Latest commit on main",
					"author": {
						"raw": "Test User <test@example.com>",
						"user": {
							"uuid": "{user-uuid}",
							"username": "testuser",
							"display_name": "Test User"
						}
					},
					"date": "2024-01-15T10:30:00+00:00",
					"links": {
						"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commit/abc123def456"},
						"html": {"href": "https://bitbucket.org/myworkspace/myrepo/commits/abc123def456"}
					}
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches/main"},
					"commits": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commits/main"},
					"html": {"href": "https://bitbucket.org/myworkspace/myrepo/branch/main"}
				}
			}`,
			statusCode: http.StatusOK,
			wantName:   "main",
		},
		{
			name:        "branch with slash in name",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "feature/my-feature",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/feature%2Fmy-feature",
			response: `{
				"name": "feature/my-feature",
				"type": "branch",
				"target": {
					"hash": "feat123",
					"type": "commit",
					"message": "Feature commit"
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches/feature%2Fmy-feature"},
					"commits": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commits/feature%2Fmy-feature"},
					"html": {"href": "https://bitbucket.org/myworkspace/myrepo/branch/feature%2Fmy-feature"}
				}
			}`,
			statusCode: http.StatusOK,
			wantName:   "feature/my-feature",
		},
		{
			name:        "not found",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "nonexistent-branch",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/nonexistent-branch",
			response:    `{"error": {"message": "Branch not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "branch with nested slashes",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "feature/user/task-123",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/feature%2Fuser%2Ftask-123",
			response: `{
				"name": "feature/user/task-123",
				"type": "branch",
				"target": {
					"hash": "nested123",
					"type": "commit"
				},
				"links": {
					"self": {"href": "..."},
					"commits": {"href": "..."},
					"html": {"href": "..."}
				}
			}`,
			statusCode: http.StatusOK,
			wantName:   "feature/user/task-123",
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

			// Verify URL path (with escaped branch name)
			if tt.expectedURL != "" && !strings.HasSuffix(receivedReq.URL.RawPath, tt.expectedURL) && !strings.HasSuffix(receivedReq.URL.Path, tt.expectedURL) {
				// Check both RawPath (escaped) and Path (unescaped) since behavior may vary
				gotPath := receivedReq.URL.RawPath
				if gotPath == "" {
					gotPath = receivedReq.URL.Path
				}
				t.Errorf("expected URL path to end with %q, got %q", tt.expectedURL, gotPath)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify result
			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
			}
		})
	}
}

func TestCreateBranch(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		repoSlug     string
		opts         *BranchCreateOptions
		expectedBody map[string]interface{}
		response     string
		statusCode   int
		wantErr      bool
		wantName     string
	}{
		{
			name:      "success",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &BranchCreateOptions{
				Name:   "new-feature",
				Target: struct{ Hash string `json:"hash"` }{Hash: "abc123def456"},
			},
			response: `{
				"name": "new-feature",
				"type": "branch",
				"target": {
					"hash": "abc123def456",
					"type": "commit",
					"message": "Source commit"
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches/new-feature"},
					"commits": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commits/new-feature"},
					"html": {"href": "https://bitbucket.org/myworkspace/myrepo/branch/new-feature"}
				}
			}`,
			statusCode: http.StatusCreated,
			wantName:   "new-feature",
		},
		{
			name:      "already exists",
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
			name:      "invalid target",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &BranchCreateOptions{
				Name:   "new-branch",
				Target: struct{ Hash string `json:"hash"` }{Hash: "invalid-hash"},
			},
			response:   `{"error": {"message": "Invalid target commit hash"}}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:      "create branch with slash in name",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &BranchCreateOptions{
				Name:   "feature/new-thing",
				Target: struct{ Hash string `json:"hash"` }{Hash: "def456"},
			},
			response: `{
				"name": "feature/new-thing",
				"type": "branch",
				"target": {
					"hash": "def456",
					"type": "commit"
				},
				"links": {
					"self": {"href": "..."},
					"commits": {"href": "..."},
					"html": {"href": "..."}
				}
			}`,
			statusCode: http.StatusCreated,
			wantName:   "feature/new-thing",
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

			// Verify HTTP method is POST
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			// Verify URL path
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/refs/branches"
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

			// Verify name
			if body["name"] != tt.opts.Name {
				t.Errorf("expected name %q in body, got %v", tt.opts.Name, body["name"])
			}

			// Verify target hash
			target, ok := body["target"].(map[string]interface{})
			if !ok {
				t.Error("expected target object in body")
			} else if target["hash"] != tt.opts.Target.Hash {
				t.Errorf("expected target hash %q, got %v", tt.opts.Target.Hash, target["hash"])
			}

			// Verify result
			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
			}
		})
	}
}

func TestDeleteBranch(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		branchName  string
		expectedURL string
		statusCode  int
		response    string
		wantErr     bool
	}{
		{
			name:        "success",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "feature-to-delete",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/feature-to-delete",
			statusCode:  http.StatusNoContent,
			response:    "",
			wantErr:     false,
		},
		{
			name:        "not found",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "nonexistent-branch",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/nonexistent-branch",
			statusCode:  http.StatusNotFound,
			response:    `{"error": {"message": "Branch not found"}}`,
			wantErr:     true,
		},
		{
			name:        "cannot delete main",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "main",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/main",
			statusCode:  http.StatusForbidden,
			response:    `{"error": {"message": "Cannot delete the main branch"}}`,
			wantErr:     true,
		},
		{
			name:        "delete branch with slash in name",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			branchName:  "feature/old-feature",
			expectedURL: "/repositories/myworkspace/myrepo/refs/branches/feature%2Fold-feature",
			statusCode:  http.StatusNoContent,
			response:    "",
			wantErr:     false,
		},
		{
			name:        "unauthorized deletion",
			workspace:   "other-workspace",
			repoSlug:    "protected-repo",
			branchName:  "protected-branch",
			statusCode:  http.StatusUnauthorized,
			response:    `{"error": {"message": "Unauthorized", "detail": "You do not have permission to delete this branch"}}`,
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

			// Verify HTTP method is DELETE
			if receivedReq.Method != http.MethodDelete {
				t.Errorf("expected DELETE method, got %s", receivedReq.Method)
			}

			// Verify URL path (with escaped branch name if contains slash)
			if tt.expectedURL != "" {
				gotPath := receivedReq.URL.RawPath
				if gotPath == "" {
					gotPath = receivedReq.URL.Path
				}
				if !strings.HasSuffix(gotPath, tt.expectedURL) && !strings.HasSuffix(receivedReq.URL.Path, strings.ReplaceAll(tt.expectedURL, "%2F", "/")) {
					t.Errorf("expected URL path to end with %q, got %q", tt.expectedURL, gotPath)
				}
			}
		})
	}
}

func TestBranchParsing(t *testing.T) {
	// Test comprehensive branch response parsing with all fields
	responseJSON := `{
		"name": "feature/complete-test",
		"type": "branch",
		"target": {
			"hash": "abc123def456789",
			"type": "commit",
			"message": "Complete commit message for testing",
			"author": {
				"raw": "Test Author <author@example.com>",
				"user": {
					"uuid": "{author-uuid}",
					"username": "testauthor",
					"display_name": "Test Author",
					"account_id": "author123"
				}
			},
			"date": "2024-01-20T15:45:00+00:00",
			"links": {
				"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commit/abc123def456789"},
				"html": {"href": "https://bitbucket.org/myworkspace/myrepo/commits/abc123def456789"}
			}
		},
		"links": {
			"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches/feature%2Fcomplete-test"},
			"commits": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/commits/feature%2Fcomplete-test"},
			"html": {"href": "https://bitbucket.org/myworkspace/myrepo/branch/feature%2Fcomplete-test"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	branch, err := client.GetBranch(context.Background(), "myworkspace", "myrepo", "feature/complete-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are parsed correctly
	if branch.Name != "feature/complete-test" {
		t.Errorf("expected name 'feature/complete-test', got %q", branch.Name)
	}

	if branch.Type != "branch" {
		t.Errorf("expected type 'branch', got %q", branch.Type)
	}

	// Verify target
	if branch.Target == nil {
		t.Fatal("expected Target to not be nil")
	}

	if branch.Target.Hash != "abc123def456789" {
		t.Errorf("expected target hash 'abc123def456789', got %q", branch.Target.Hash)
	}

	if branch.Target.Type != "commit" {
		t.Errorf("expected target type 'commit', got %q", branch.Target.Type)
	}

	if branch.Target.Message != "Complete commit message for testing" {
		t.Errorf("expected message 'Complete commit message for testing', got %q", branch.Target.Message)
	}

	if branch.Target.Author.Raw != "Test Author <author@example.com>" {
		t.Errorf("expected author raw 'Test Author <author@example.com>', got %q", branch.Target.Author.Raw)
	}

	if branch.Target.Author.User == nil {
		t.Fatal("expected Author.User to not be nil")
	}

	if branch.Target.Author.User.DisplayName != "Test Author" {
		t.Errorf("expected author display_name 'Test Author', got %q", branch.Target.Author.User.DisplayName)
	}

	if branch.Target.Date != "2024-01-20T15:45:00+00:00" {
		t.Errorf("expected date '2024-01-20T15:45:00+00:00', got %q", branch.Target.Date)
	}

	// Verify links
	if branch.Links.Self.Href == "" {
		t.Error("expected Links.Self.Href to not be empty")
	}

	if branch.Links.Commits.Href == "" {
		t.Error("expected Links.Commits.Href to not be empty")
	}

	if branch.Links.HTML.Href == "" {
		t.Error("expected Links.HTML.Href to not be empty")
	}

	// Verify target links
	if branch.Target.Links.Self.Href == "" {
		t.Error("expected Target.Links.Self.Href to not be empty")
	}

	if branch.Target.Links.HTML.Href == "" {
		t.Error("expected Target.Links.HTML.Href to not be empty")
	}
}

func TestBranchErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		response       string
		wantStatusCode int
		wantMessage    string
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
			response:       `{"error": {"message": "Branch not found"}}`,
			wantStatusCode: http.StatusNotFound,
			wantMessage:    "Branch not found",
		},
		{
			name:           "409 Conflict",
			statusCode:     http.StatusConflict,
			response:       `{"error": {"message": "Branch already exists"}}`,
			wantStatusCode: http.StatusConflict,
			wantMessage:    "Branch already exists",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			response:       `{"error": {"message": "Cannot delete the main branch"}}`,
			wantStatusCode: http.StatusForbidden,
			wantMessage:    "Cannot delete the main branch",
		},
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			response:       `{"error": {"message": "Invalid branch name"}}`,
			wantStatusCode: http.StatusBadRequest,
			wantMessage:    "Invalid branch name",
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

			_, err := client.GetBranch(context.Background(), "workspace", "repo", "branch")

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

func TestListBranchesPagination(t *testing.T) {
	// Test that pagination response is properly parsed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 50,
			"page": 2,
			"pagelen": 10,
			"next": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches?page=3",
			"previous": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/refs/branches?page=1",
			"values": [
				{"name": "branch-1", "type": "branch", "target": {"hash": "111", "type": "commit"}},
				{"name": "branch-2", "type": "branch", "target": {"hash": "222", "type": "commit"}}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListBranches(context.Background(), "myworkspace", "myrepo", &BranchListOptions{Page: 2, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Size != 50 {
		t.Errorf("expected size 50, got %d", result.Size)
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
