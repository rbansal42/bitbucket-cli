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

func TestListProjects(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		opts          *ProjectListOptions
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
			expectedURL: "/workspaces/myworkspace/projects",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"uuid": "{proj-1}", "key": "PROJ1", "name": "Project One"},
					{"uuid": "{proj-2}", "key": "PROJ2", "name": "Project Two"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with pagination",
			workspace:   "myworkspace",
			opts:        &ProjectListOptions{Page: 2, Limit: 5},
			expectedURL: "/workspaces/myworkspace/projects",
			expectedQuery: map[string]string{"page": "2", "pagelen": "5"},
			response: `{
				"size": 15,
				"page": 2,
				"pagelen": 5,
				"next": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects?page=3",
				"previous": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects?page=1",
				"values": [
					{"uuid": "{proj-6}", "key": "PROJ6", "name": "Project Six"},
					{"uuid": "{proj-7}", "key": "PROJ7", "name": "Project Seven"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with sort",
			workspace:   "myworkspace",
			opts:        &ProjectListOptions{Sort: "-updated_on"},
			expectedURL: "/workspaces/myworkspace/projects",
			expectedQuery: map[string]string{"sort": "-updated_on"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"uuid": "{proj-1}", "key": "RECENT", "name": "Recent Project"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with query filter",
			workspace:   "myworkspace",
			opts:        &ProjectListOptions{Query: "name~\"test\""},
			expectedURL: "/workspaces/myworkspace/projects",
			expectedQuery: map[string]string{"q": "name~\"test\""},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"uuid": "{proj-1}", "key": "TEST", "name": "Test Project"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "workspace not found",
			workspace:   "nonexistent",
			opts:        nil,
			response:    `{"error": {"message": "Workspace not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "unauthorized",
			workspace:   "myworkspace",
			opts:        nil,
			response:    `{"error": {"message": "Unauthorized", "detail": "Authentication required"}}`,
			statusCode:  http.StatusUnauthorized,
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

			// Verify URL path
			if tt.expectedURL != "" && !strings.HasSuffix(receivedReq.URL.Path, tt.expectedURL) {
				t.Errorf("expected URL path to end with %q, got %q", tt.expectedURL, receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
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
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects/PROJ"},
					"html": {"href": "https://bitbucket.org/myworkspace/projects/PROJ"},
					"avatar": {"href": "https://bitbucket.org/account/myworkspace/projects/PROJ/avatar/32"}
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
		{
			name:       "workspace not found",
			workspace:  "nonexistent",
			projectKey: "PROJ",
			response:   `{"error": {"message": "Workspace not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "unauthorized access",
			workspace:  "private-workspace",
			projectKey: "PROJ",
			response:   `{"error": {"message": "Unauthorized", "detail": "You do not have access to this project"}}`,
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

			// Verify URL contains workspace and project key
			expectedPath := "/workspaces/" + tt.workspace + "/projects/" + tt.projectKey
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path to contain %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify response parsing
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
			name:      "basic project creation",
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
				"is_private": true,
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z",
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects/NEWPROJ"},
					"html": {"href": "https://bitbucket.org/myworkspace/projects/NEWPROJ"},
					"avatar": {"href": "https://bitbucket.org/account/myworkspace/projects/NEWPROJ/avatar/32"}
				}
			}`,
			statusCode: http.StatusCreated,
			wantKey:    "NEWPROJ",
		},
		{
			name:      "public project creation",
			workspace: "myworkspace",
			opts: &ProjectCreateOptions{
				Key:       "PUBLIC",
				Name:      "Public Project",
				IsPrivate: false,
			},
			response: `{
				"uuid": "{public-proj-uuid}",
				"key": "PUBLIC",
				"name": "Public Project",
				"is_private": false,
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantKey:    "PUBLIC",
		},
		{
			name:      "project already exists - conflict",
			workspace: "myworkspace",
			opts: &ProjectCreateOptions{
				Key:       "EXISTING",
				Name:      "Existing Project",
				IsPrivate: true,
			},
			response:   `{"error": {"message": "Project with this key already exists"}}`,
			statusCode: http.StatusConflict,
			wantErr:    true,
		},
		{
			name:      "validation error - invalid key",
			workspace: "myworkspace",
			opts: &ProjectCreateOptions{
				Key:       "invalid key",
				Name:      "Invalid Project",
				IsPrivate: true,
			},
			response:   `{"error": {"message": "Validation error", "fields": {"key": "Project key must be alphanumeric and uppercase"}}}`,
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

			// Verify HTTP method is POST
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			// Verify URL
			expectedPath := "/workspaces/" + tt.workspace + "/projects"
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify Content-Type
			if ct := receivedReq.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			// Verify request body
			var body map[string]interface{}
			if err := json.Unmarshal(receivedBody, &body); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			if body["key"] != tt.opts.Key {
				t.Errorf("expected key %q in body, got %v", tt.opts.Key, body["key"])
			}

			if body["name"] != tt.opts.Name {
				t.Errorf("expected name %q in body, got %v", tt.opts.Name, body["name"])
			}

			if body["is_private"] != tt.opts.IsPrivate {
				t.Errorf("expected is_private %v in body, got %v", tt.opts.IsPrivate, body["is_private"])
			}

			// Verify result
			if result.Key != tt.wantKey {
				t.Errorf("expected key %q, got %q", tt.wantKey, result.Key)
			}
		})
	}
}

func TestUpdateProject(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		projectKey string
		opts       *ProjectCreateOptions
		response   string
		statusCode int
		wantErr    bool
		wantName   string
	}{
		{
			name:       "successful update",
			workspace:  "myworkspace",
			projectKey: "PROJ",
			opts: &ProjectCreateOptions{
				Name:        "Updated Project Name",
				Description: "Updated description",
				IsPrivate:   true,
			},
			response: `{
				"uuid": "{proj-uuid}",
				"key": "PROJ",
				"name": "Updated Project Name",
				"description": "Updated description",
				"is_private": true,
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-15T12:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantName:   "Updated Project Name",
		},
		{
			name:       "project not found",
			workspace:  "myworkspace",
			projectKey: "NONEXISTENT",
			opts: &ProjectCreateOptions{
				Name:      "Updated Name",
				IsPrivate: true,
			},
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

			result, err := client.UpdateProject(context.Background(), tt.workspace, tt.projectKey, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify HTTP method is PUT
			if receivedReq.Method != http.MethodPut {
				t.Errorf("expected PUT method, got %s", receivedReq.Method)
			}

			// Verify URL
			expectedPath := "/workspaces/" + tt.workspace + "/projects/" + tt.projectKey
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify result
			if result.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, result.Name)
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
		{
			name:       "unauthorized deletion",
			workspace:  "other-workspace",
			projectKey: "PROJ",
			statusCode: http.StatusUnauthorized,
			response:   `{"error": {"message": "Unauthorized", "detail": "You do not have permission to delete this project"}}`,
			wantErr:    true,
		},
		{
			name:       "forbidden - project has repositories",
			workspace:  "myworkspace",
			projectKey: "HASREPOS",
			statusCode: http.StatusForbidden,
			response:   `{"error": {"message": "Forbidden", "detail": "Cannot delete project with repositories"}}`,
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

			// Verify HTTP method is DELETE
			if receivedReq.Method != http.MethodDelete {
				t.Errorf("expected DELETE method, got %s", receivedReq.Method)
			}

			// Verify URL contains workspace and project key
			expectedPath := "/workspaces/" + tt.workspace + "/projects/" + tt.projectKey
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}
		})
	}
}

func TestProjectParsing(t *testing.T) {
	// Test comprehensive project response parsing with all fields
	responseJSON := `{
		"uuid": "{complete-proj-uuid}",
		"key": "COMPLETE",
		"name": "Complete Project",
		"description": "A complete project for testing",
		"is_private": true,
		"created_on": "2024-01-15T10:30:00Z",
		"updated_on": "2024-02-20T14:45:00Z",
		"owner": {
			"uuid": "{owner-uuid}",
			"username": "owner",
			"display_name": "Project Owner",
			"account_id": "owner123"
		},
		"workspace": {
			"uuid": "{ws-uuid}",
			"slug": "myworkspace",
			"name": "My Workspace"
		},
		"links": {
			"self": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects/COMPLETE"},
			"html": {"href": "https://bitbucket.org/myworkspace/projects/COMPLETE"},
			"avatar": {"href": "https://bitbucket.org/account/myworkspace/projects/COMPLETE/avatar/32/"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	proj, err := client.GetProject(context.Background(), "myworkspace", "COMPLETE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are parsed correctly
	if proj.UUID != "{complete-proj-uuid}" {
		t.Errorf("expected UUID '{complete-proj-uuid}', got %q", proj.UUID)
	}

	if proj.Key != "COMPLETE" {
		t.Errorf("expected key 'COMPLETE', got %q", proj.Key)
	}

	if proj.Name != "Complete Project" {
		t.Errorf("expected name 'Complete Project', got %q", proj.Name)
	}

	if proj.Description != "A complete project for testing" {
		t.Errorf("expected description 'A complete project for testing', got %q", proj.Description)
	}

	if !proj.IsPrivate {
		t.Error("expected IsPrivate to be true")
	}

	// Verify owner
	if proj.Owner == nil {
		t.Fatal("expected Owner to not be nil")
	}
	if proj.Owner.DisplayName != "Project Owner" {
		t.Errorf("expected owner display_name 'Project Owner', got %q", proj.Owner.DisplayName)
	}

	// Verify workspace
	if proj.Workspace == nil {
		t.Fatal("expected Workspace to not be nil")
	}
	if proj.Workspace.Slug != "myworkspace" {
		t.Errorf("expected workspace slug 'myworkspace', got %q", proj.Workspace.Slug)
	}

	// Verify links
	if proj.Links.Self.Href == "" {
		t.Error("expected self link to be set")
	}
	if proj.Links.HTML.Href == "" {
		t.Error("expected HTML link to be set")
	}
	if proj.Links.Avatar.Href == "" {
		t.Error("expected avatar link to be set")
	}

	// Verify time parsing
	expectedCreated := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !proj.CreatedOn.Equal(expectedCreated) {
		t.Errorf("expected created_on %v, got %v", expectedCreated, proj.CreatedOn)
	}

	expectedUpdated := time.Date(2024, 2, 20, 14, 45, 0, 0, time.UTC)
	if !proj.UpdatedOn.Equal(expectedUpdated) {
		t.Errorf("expected updated_on %v, got %v", expectedUpdated, proj.UpdatedOn)
	}
}

func TestProjectErrorHandling(t *testing.T) {
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
			response:       `{"error": {"message": "Project not found"}}`,
			wantStatusCode: http.StatusNotFound,
			wantMessage:    "Project not found",
		},
		{
			name:           "409 Conflict",
			statusCode:     http.StatusConflict,
			response:       `{"error": {"message": "Project already exists"}}`,
			wantStatusCode: http.StatusConflict,
			wantMessage:    "Project already exists",
		},
		{
			name:           "400 Bad Request with fields",
			statusCode:     http.StatusBadRequest,
			response:       `{"error": {"message": "Validation error", "fields": {"key": "Invalid key"}}}`,
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

			_, err := client.GetProject(context.Background(), "workspace", "PROJ")

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

func TestListProjectsPagination(t *testing.T) {
	// Test that pagination response is properly parsed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 50,
			"page": 2,
			"pagelen": 10,
			"next": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects?page=3",
			"previous": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects?page=1",
			"values": [
				{"uuid": "{proj-1}", "key": "PROJ1", "name": "Project 1"},
				{"uuid": "{proj-2}", "key": "PROJ2", "name": "Project 2"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListProjects(context.Background(), "myworkspace", &ProjectListOptions{Page: 2, Limit: 10})
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
