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
		expectedURL   string
		expectedQuery map[string]string
		response      string
		statusCode    int
		wantErr       bool
		wantCount     int
	}{
		{
			name:        "basic list without options",
			opts:        nil,
			expectedURL: "/user/permissions/workspaces",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"permission": "owner",
						"workspace": {"uuid": "{ws-1}", "slug": "workspace1", "name": "Workspace 1"}
					},
					{
						"permission": "member",
						"workspace": {"uuid": "{ws-2}", "slug": "workspace2", "name": "Workspace 2"}
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with role filter",
			opts:        &WorkspaceListOptions{Role: "owner"},
			expectedURL: "/user/permissions/workspaces",
			expectedQuery: map[string]string{"role": "owner"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"permission": "owner",
						"workspace": {"uuid": "{ws-1}", "slug": "owned-workspace", "name": "Owned Workspace"}
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with pagination",
			opts:        &WorkspaceListOptions{Page: 2, Limit: 5},
			expectedURL: "/user/permissions/workspaces",
			expectedQuery: map[string]string{"page": "2", "pagelen": "5"},
			response: `{
				"size": 10,
				"page": 2,
				"pagelen": 5,
				"next": "https://api.bitbucket.org/2.0/user/permissions/workspaces?page=3",
				"values": [
					{
						"permission": "member",
						"workspace": {"uuid": "{ws-6}", "slug": "workspace6", "name": "Workspace 6"}
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "handles 401 unauthorized",
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
				t.Errorf("expected %d workspaces, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetWorkspace(t *testing.T) {
	tests := []struct {
		name          string
		workspaceSlug string
		response      string
		statusCode    int
		wantErr       bool
		wantName      string
	}{
		{
			name:          "successfully get workspace",
			workspaceSlug: "myworkspace",
			response: `{
				"uuid": "{ws-uuid}",
				"slug": "myworkspace",
				"name": "My Workspace",
				"type": "workspace",
				"is_private": true,
				"created_on": "2024-01-01T00:00:00Z",
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace"},
					"html": {"href": "https://bitbucket.org/myworkspace"},
					"avatar": {"href": "https://bitbucket.org/myworkspace/avatar"},
					"members": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace/members"},
					"projects": {"href": "https://api.bitbucket.org/2.0/workspaces/myworkspace/projects"},
					"repositories": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace"}
				}
			}`,
			statusCode: http.StatusOK,
			wantName:   "My Workspace",
		},
		{
			name:          "workspace not found",
			workspaceSlug: "nonexistent",
			response:      `{"error": {"message": "Workspace not found"}}`,
			statusCode:    http.StatusNotFound,
			wantErr:       true,
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

			result, err := client.GetWorkspace(context.Background(), tt.workspaceSlug)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL contains workspace slug
			expectedPath := "/workspaces/" + tt.workspaceSlug
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

func TestListWorkspaceMembers(t *testing.T) {
	tests := []struct {
		name          string
		workspaceSlug string
		opts          *WorkspaceMemberListOptions
		expectedURL   string
		expectedQuery map[string]string
		response      string
		statusCode    int
		wantErr       bool
		wantCount     int
	}{
		{
			name:          "successfully list members",
			workspaceSlug: "myworkspace",
			opts:          nil,
			expectedURL:   "/workspaces/myworkspace/permissions",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"permission": "owner",
						"user": {"uuid": "{user-1}", "display_name": "User One"},
						"workspace": {"uuid": "{ws-1}", "slug": "myworkspace"},
						"added_on": "2024-01-01T00:00:00Z"
					},
					{
						"permission": "member",
						"user": {"uuid": "{user-2}", "display_name": "User Two"},
						"workspace": {"uuid": "{ws-1}", "slug": "myworkspace"},
						"added_on": "2024-01-15T00:00:00Z"
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:          "list members with pagination",
			workspaceSlug: "myworkspace",
			opts:          &WorkspaceMemberListOptions{Page: 2, Limit: 10},
			expectedURL:   "/workspaces/myworkspace/permissions",
			expectedQuery: map[string]string{"page": "2", "pagelen": "10"},
			response: `{
				"size": 50,
				"page": 2,
				"pagelen": 10,
				"values": []
			}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:          "workspace not found",
			workspaceSlug: "nonexistent",
			opts:          nil,
			response:      `{"error": {"message": "Workspace not found"}}`,
			statusCode:    http.StatusNotFound,
			wantErr:       true,
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

			result, err := client.ListWorkspaceMembers(context.Background(), tt.workspaceSlug, tt.opts)

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
				t.Errorf("expected %d members, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestWorkspaceFullParsing(t *testing.T) {
	// Test comprehensive workspace response parsing with all fields
	responseJSON := `{
		"uuid": "{complete-ws-uuid}",
		"slug": "complete-workspace",
		"name": "Complete Workspace",
		"type": "workspace",
		"is_private": true,
		"created_on": "2024-01-15T10:30:00Z",
		"links": {
			"self": {"href": "https://api.bitbucket.org/2.0/workspaces/complete-workspace"},
			"html": {"href": "https://bitbucket.org/complete-workspace"},
			"avatar": {"href": "https://bitbucket.org/workspaces/complete-workspace/avatar"},
			"members": {"href": "https://api.bitbucket.org/2.0/workspaces/complete-workspace/members"},
			"projects": {"href": "https://api.bitbucket.org/2.0/workspaces/complete-workspace/projects"},
			"repositories": {"href": "https://api.bitbucket.org/2.0/repositories/complete-workspace"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	ws, err := client.GetWorkspace(context.Background(), "complete-workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are parsed correctly
	if ws.UUID != "{complete-ws-uuid}" {
		t.Errorf("expected UUID '{complete-ws-uuid}', got %q", ws.UUID)
	}

	if ws.Slug != "complete-workspace" {
		t.Errorf("expected slug 'complete-workspace', got %q", ws.Slug)
	}

	if ws.Name != "Complete Workspace" {
		t.Errorf("expected name 'Complete Workspace', got %q", ws.Name)
	}

	if ws.Type != "workspace" {
		t.Errorf("expected type 'workspace', got %q", ws.Type)
	}

	if !ws.IsPrivate {
		t.Error("expected IsPrivate to be true")
	}

	// Verify links
	if ws.Links.Self.Href != "https://api.bitbucket.org/2.0/workspaces/complete-workspace" {
		t.Errorf("expected self link, got %q", ws.Links.Self.Href)
	}

	if ws.Links.HTML.Href != "https://bitbucket.org/complete-workspace" {
		t.Errorf("expected html link, got %q", ws.Links.HTML.Href)
	}

	if ws.Links.Members.Href != "https://api.bitbucket.org/2.0/workspaces/complete-workspace/members" {
		t.Errorf("expected members link, got %q", ws.Links.Members.Href)
	}

	if ws.Links.Projects.Href != "https://api.bitbucket.org/2.0/workspaces/complete-workspace/projects" {
		t.Errorf("expected projects link, got %q", ws.Links.Projects.Href)
	}

	if ws.Links.Repositories.Href != "https://api.bitbucket.org/2.0/repositories/complete-workspace" {
		t.Errorf("expected repositories link, got %q", ws.Links.Repositories.Href)
	}
}

func TestWorkspaceMembershipParsing(t *testing.T) {
	// Test workspace membership parsing
	responseJSON := `{
		"size": 1,
		"page": 1,
		"pagelen": 10,
		"values": [
			{
				"permission": "owner",
				"user": {
					"uuid": "{user-uuid}",
					"username": "testuser",
					"display_name": "Test User",
					"account_id": "account123"
				},
				"workspace": {
					"uuid": "{ws-uuid}",
					"slug": "testworkspace",
					"name": "Test Workspace"
				},
				"added_on": "2024-01-01T12:00:00Z",
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/workspaces/testworkspace/permissions/{user-uuid}"}
				}
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListWorkspaceMembers(context.Background(), "testworkspace", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Values) != 1 {
		t.Fatalf("expected 1 member, got %d", len(result.Values))
	}

	member := result.Values[0]

	if member.Permission != "owner" {
		t.Errorf("expected permission 'owner', got %q", member.Permission)
	}

	if member.User == nil {
		t.Fatal("expected User to not be nil")
	}

	if member.User.DisplayName != "Test User" {
		t.Errorf("expected user display_name 'Test User', got %q", member.User.DisplayName)
	}

	if member.Workspace.Slug != "testworkspace" {
		t.Errorf("expected workspace slug 'testworkspace', got %q", member.Workspace.Slug)
	}
}
