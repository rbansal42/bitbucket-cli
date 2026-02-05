package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListSnippets(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		opts          *SnippetListOptions
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
			expectedURL: "/snippets/myworkspace",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"type": "snippet",
						"id": 123,
						"title": "First Snippet",
						"scm": "git",
						"is_private": false,
						"owner": {"display_name": "Test User", "uuid": "{user-uuid}"},
						"files": {
							"hello.py": {
								"links": {
									"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/123/files/hello.py"},
									"html": {"href": "https://bitbucket.org/snippets/myworkspace/123/hello.py"}
								}
							}
						},
						"links": {
							"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/123"},
							"html": {"href": "https://bitbucket.org/snippets/myworkspace/123"},
							"comments": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/123/comments"},
							"watchers": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/123/watchers"},
							"commits": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/123/commits"}
						}
					},
					{
						"type": "snippet",
						"id": 456,
						"title": "Second Snippet",
						"scm": "git",
						"is_private": true,
						"owner": {"display_name": "Test User", "uuid": "{user-uuid}"},
						"files": {},
						"links": {
							"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/456"},
							"html": {"href": "https://bitbucket.org/snippets/myworkspace/456"}
						}
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with role filter",
			workspace:   "myworkspace",
			opts:        &SnippetListOptions{Role: "owner"},
			expectedURL: "/snippets/myworkspace",
			expectedQuery: map[string]string{"role": "owner"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"type": "snippet",
						"id": 123,
						"title": "My Snippet",
						"scm": "git",
						"is_private": false
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with pagination",
			workspace:   "myworkspace",
			opts:        &SnippetListOptions{Page: 2, Limit: 5},
			expectedURL: "/snippets/myworkspace",
			expectedQuery: map[string]string{"page": "2", "pagelen": "5"},
			response: `{
				"size": 15,
				"page": 2,
				"pagelen": 5,
				"next": "https://api.bitbucket.org/2.0/snippets/myworkspace?page=3",
				"previous": "https://api.bitbucket.org/2.0/snippets/myworkspace?page=1",
				"values": [
					{"type": "snippet", "id": 6, "title": "Snippet 6"},
					{"type": "snippet", "id": 7, "title": "Snippet 7"},
					{"type": "snippet", "id": 8, "title": "Snippet 8"},
					{"type": "snippet", "id": 9, "title": "Snippet 9"},
					{"type": "snippet", "id": 10, "title": "Snippet 10"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  5,
		},
		{
			name:        "list with contributor role",
			workspace:   "myworkspace",
			opts:        &SnippetListOptions{Role: "contributor"},
			expectedURL: "/snippets/myworkspace",
			expectedQuery: map[string]string{"role": "contributor"},
			response: `{
				"size": 3,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"type": "snippet", "id": 1, "title": "Contrib 1"},
					{"type": "snippet", "id": 2, "title": "Contrib 2"},
					{"type": "snippet", "id": 3, "title": "Contrib 3"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  3,
		},
		{
			name:       "workspace not found",
			workspace:  "nonexistent",
			opts:       nil,
			response:   `{"error": {"message": "Workspace not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			workspace:  "myworkspace",
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

			result, err := client.ListSnippets(context.Background(), tt.workspace, tt.opts)

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
				t.Errorf("expected %d snippets, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetSnippet(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		encodedID   string
		expectedURL string
		response    string
		statusCode  int
		wantErr     bool
		wantTitle   string
		wantID      int
	}{
		{
			name:        "success",
			workspace:   "myworkspace",
			encodedID:   "abc123",
			expectedURL: "/snippets/myworkspace/abc123",
			response: `{
				"type": "snippet",
				"id": 123,
				"title": "My Test Snippet",
				"scm": "git",
				"created_on": "2024-01-15T10:30:00+00:00",
				"updated_on": "2024-01-20T14:45:00+00:00",
				"is_private": false,
				"owner": {
					"uuid": "{owner-uuid}",
					"username": "testowner",
					"display_name": "Test Owner"
				},
				"creator": {
					"uuid": "{creator-uuid}",
					"username": "testcreator",
					"display_name": "Test Creator"
				},
				"files": {
					"main.py": {
						"links": {
							"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/abc123/files/main.py"},
							"html": {"href": "https://bitbucket.org/snippets/myworkspace/abc123/main.py"}
						}
					},
					"utils.py": {
						"links": {
							"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/abc123/files/utils.py"},
							"html": {"href": "https://bitbucket.org/snippets/myworkspace/abc123/utils.py"}
						}
					}
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/abc123"},
					"html": {"href": "https://bitbucket.org/snippets/myworkspace/abc123"},
					"comments": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/abc123/comments"},
					"watchers": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/abc123/watchers"},
					"commits": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/abc123/commits"}
				}
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "My Test Snippet",
			wantID:     123,
		},
		{
			name:        "not found",
			workspace:   "myworkspace",
			encodedID:   "nonexistent",
			expectedURL: "/snippets/myworkspace/nonexistent",
			response:    `{"error": {"message": "Snippet not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "private snippet forbidden",
			workspace:   "otherworkspace",
			encodedID:   "private123",
			expectedURL: "/snippets/otherworkspace/private123",
			response:    `{"error": {"message": "Forbidden", "detail": "You do not have permission to view this snippet"}}`,
			statusCode:  http.StatusForbidden,
			wantErr:     true,
		},
		{
			name:        "unauthorized",
			workspace:   "myworkspace",
			encodedID:   "abc123",
			response:    `{"error": {"message": "Unauthorized"}}`,
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

			result, err := client.GetSnippet(context.Background(), tt.workspace, tt.encodedID)

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

			// Verify result
			if result.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, result.Title)
			}

			if result.ID != tt.wantID {
				t.Errorf("expected ID %d, got %d", tt.wantID, result.ID)
			}
		})
	}
}

func TestCreateSnippet(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		title      string
		isPrivate  bool
		files      map[string]string
		response   string
		statusCode int
		wantErr    bool
		wantTitle  string
		wantID     int
	}{
		{
			name:      "success with single file",
			workspace: "myworkspace",
			title:     "New Snippet",
			isPrivate: false,
			files:     map[string]string{"hello.py": "print('Hello, World!')"},
			response: `{
				"type": "snippet",
				"id": 789,
				"title": "New Snippet",
				"scm": "git",
				"is_private": false,
				"created_on": "2024-01-20T10:00:00+00:00",
				"updated_on": "2024-01-20T10:00:00+00:00",
				"owner": {"display_name": "Test User"},
				"files": {
					"hello.py": {
						"links": {
							"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/789/files/hello.py"}
						}
					}
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/789"},
					"html": {"href": "https://bitbucket.org/snippets/myworkspace/789"}
				}
			}`,
			statusCode: http.StatusCreated,
			wantTitle:  "New Snippet",
			wantID:     789,
		},
		{
			name:      "success with multiple files",
			workspace: "myworkspace",
			title:     "Multi-file Snippet",
			isPrivate: true,
			files: map[string]string{
				"main.py":  "from utils import helper\nhelper()",
				"utils.py": "def helper():\n    print('Helper function')",
			},
			response: `{
				"type": "snippet",
				"id": 790,
				"title": "Multi-file Snippet",
				"scm": "git",
				"is_private": true,
				"files": {
					"main.py": {"links": {"self": {"href": "..."}}},
					"utils.py": {"links": {"self": {"href": "..."}}}
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/790"},
					"html": {"href": "https://bitbucket.org/snippets/myworkspace/790"}
				}
			}`,
			statusCode: http.StatusCreated,
			wantTitle:  "Multi-file Snippet",
			wantID:     790,
		},
		{
			name:       "validation error - no files",
			workspace:  "myworkspace",
			title:      "Empty Snippet",
			isPrivate:  false,
			files:      map[string]string{},
			response:   `{"error": {"message": "Validation error", "fields": {"files": "At least one file is required"}}}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			workspace:  "myworkspace",
			title:      "Unauthorized Snippet",
			isPrivate:  false,
			files:      map[string]string{"test.py": "# test"},
			response:   `{"error": {"message": "Unauthorized"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "forbidden - workspace access",
			workspace:  "restricted-workspace",
			title:      "Forbidden Snippet",
			isPrivate:  false,
			files:      map[string]string{"test.py": "# test"},
			response:   `{"error": {"message": "Forbidden", "detail": "You do not have permission to create snippets in this workspace"}}`,
			statusCode: http.StatusForbidden,
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

			result, err := client.CreateSnippet(context.Background(), tt.workspace, tt.title, tt.isPrivate, tt.files)

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
			expectedPath := "/snippets/" + tt.workspace
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify Content-Type is multipart/form-data
			contentType := receivedReq.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				t.Errorf("expected Content-Type to start with multipart/form-data, got %s", contentType)
			}

			// Verify result
			if result.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, result.Title)
			}

			if result.ID != tt.wantID {
				t.Errorf("expected ID %d, got %d", tt.wantID, result.ID)
			}
		})
	}
}

func TestUpdateSnippet(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		encodedID  string
		title      string
		files      map[string]string
		response   string
		statusCode int
		wantErr    bool
		wantTitle  string
	}{
		{
			name:      "success - update title and files",
			workspace: "myworkspace",
			encodedID: "abc123",
			title:     "Updated Snippet Title",
			files:     map[string]string{"updated.py": "print('Updated!')"},
			response: `{
				"type": "snippet",
				"id": 123,
				"title": "Updated Snippet Title",
				"scm": "git",
				"is_private": false,
				"updated_on": "2024-01-25T10:00:00+00:00",
				"files": {
					"updated.py": {"links": {"self": {"href": "..."}}}
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/abc123"},
					"html": {"href": "https://bitbucket.org/snippets/myworkspace/abc123"}
				}
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "Updated Snippet Title",
		},
		{
			name:      "success - update files only",
			workspace: "myworkspace",
			encodedID: "def456",
			title:     "",
			files:     map[string]string{"newfile.py": "# new content"},
			response: `{
				"type": "snippet",
				"id": 456,
				"title": "Original Title",
				"scm": "git",
				"files": {
					"newfile.py": {"links": {"self": {"href": "..."}}}
				},
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/def456"}
				}
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "Original Title",
		},
		{
			name:       "not found",
			workspace:  "myworkspace",
			encodedID:  "nonexistent",
			title:      "Updated Title",
			files:      map[string]string{"test.py": "# test"},
			response:   `{"error": {"message": "Snippet not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "forbidden - not owner",
			workspace:  "myworkspace",
			encodedID:  "others123",
			title:      "Trying to Update",
			files:      map[string]string{"test.py": "# test"},
			response:   `{"error": {"message": "Forbidden", "detail": "You do not have permission to modify this snippet"}}`,
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			workspace:  "myworkspace",
			encodedID:  "abc123",
			title:      "Update Title",
			files:      map[string]string{"test.py": "# test"},
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

			result, err := client.UpdateSnippet(context.Background(), tt.workspace, tt.encodedID, tt.title, tt.files)

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

			// Verify URL path contains snippet ID
			expectedPath := "/snippets/" + tt.workspace + "/" + tt.encodedID
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify Content-Type is multipart/form-data
			contentType := receivedReq.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				t.Errorf("expected Content-Type to start with multipart/form-data, got %s", contentType)
			}

			// Verify result
			if result.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, result.Title)
			}
		})
	}
}

func TestDeleteSnippet(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		encodedID   string
		expectedURL string
		statusCode  int
		response    string
		wantErr     bool
	}{
		{
			name:        "success",
			workspace:   "myworkspace",
			encodedID:   "abc123",
			expectedURL: "/snippets/myworkspace/abc123",
			statusCode:  http.StatusNoContent,
			response:    "",
			wantErr:     false,
		},
		{
			name:        "not found",
			workspace:   "myworkspace",
			encodedID:   "nonexistent",
			expectedURL: "/snippets/myworkspace/nonexistent",
			statusCode:  http.StatusNotFound,
			response:    `{"error": {"message": "Snippet not found"}}`,
			wantErr:     true,
		},
		{
			name:        "forbidden - not owner",
			workspace:   "myworkspace",
			encodedID:   "others123",
			expectedURL: "/snippets/myworkspace/others123",
			statusCode:  http.StatusForbidden,
			response:    `{"error": {"message": "Forbidden", "detail": "You do not have permission to delete this snippet"}}`,
			wantErr:     true,
		},
		{
			name:        "unauthorized",
			workspace:   "myworkspace",
			encodedID:   "abc123",
			statusCode:  http.StatusUnauthorized,
			response:    `{"error": {"message": "Unauthorized"}}`,
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

			err := client.DeleteSnippet(context.Background(), tt.workspace, tt.encodedID)

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

			// Verify URL path
			if tt.expectedURL != "" && !strings.HasSuffix(receivedReq.URL.Path, tt.expectedURL) {
				t.Errorf("expected URL path to end with %q, got %q", tt.expectedURL, receivedReq.URL.Path)
			}
		})
	}
}

func TestGetSnippetFileContent(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		encodedID   string
		filePath    string
		expectedURL string
		response    string
		statusCode  int
		wantErr     bool
		wantContent string
	}{
		{
			name:        "success - get file content",
			workspace:   "myworkspace",
			encodedID:   "abc123",
			filePath:    "hello.py",
			expectedURL: "/snippets/myworkspace/abc123/files/hello.py",
			response:    "print('Hello, World!')",
			statusCode:  http.StatusOK,
			wantContent: "print('Hello, World!')",
		},
		{
			name:        "success - get file with path",
			workspace:   "myworkspace",
			encodedID:   "def456",
			filePath:    "src/main.py",
			expectedURL: "/snippets/myworkspace/def456/files/src%2Fmain.py",
			response:    "# Main module\nimport sys",
			statusCode:  http.StatusOK,
			wantContent: "# Main module\nimport sys",
		},
		{
			name:        "file not found",
			workspace:   "myworkspace",
			encodedID:   "abc123",
			filePath:    "nonexistent.py",
			expectedURL: "/snippets/myworkspace/abc123/files/nonexistent.py",
			response:    `{"error": {"message": "File not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "snippet not found",
			workspace:   "myworkspace",
			encodedID:   "nonexistent",
			filePath:    "file.py",
			response:    `{"error": {"message": "Snippet not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "forbidden",
			workspace:   "otherworkspace",
			encodedID:   "private123",
			filePath:    "secret.py",
			response:    `{"error": {"message": "Forbidden"}}`,
			statusCode:  http.StatusForbidden,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				if tt.statusCode >= 400 {
					w.Header().Set("Content-Type", "application/json")
				} else {
					w.Header().Set("Content-Type", "text/plain")
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.GetSnippetFileContent(context.Background(), tt.workspace, tt.encodedID, tt.filePath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify URL path (check both raw and decoded paths)
			if tt.expectedURL != "" {
				gotPath := receivedReq.URL.RawPath
				if gotPath == "" {
					gotPath = receivedReq.URL.Path
				}
				if !strings.HasSuffix(gotPath, tt.expectedURL) && !strings.HasSuffix(receivedReq.URL.Path, strings.ReplaceAll(tt.expectedURL, "%2F", "/")) {
					t.Errorf("expected URL path to end with %q, got %q (raw: %q)", tt.expectedURL, receivedReq.URL.Path, gotPath)
				}
			}

			// Verify result content
			if string(result) != tt.wantContent {
				t.Errorf("expected content %q, got %q", tt.wantContent, string(result))
			}
		})
	}
}

func TestSnippetParsing(t *testing.T) {
	// Test comprehensive snippet response parsing with all fields
	responseJSON := `{
		"type": "snippet",
		"id": 999,
		"title": "Complete Test Snippet",
		"scm": "git",
		"created_on": "2024-01-15T10:30:00+00:00",
		"updated_on": "2024-02-20T14:45:00+00:00",
		"is_private": true,
		"owner": {
			"uuid": "{owner-uuid}",
			"username": "owneruser",
			"display_name": "Owner User",
			"account_id": "owner123"
		},
		"creator": {
			"uuid": "{creator-uuid}",
			"username": "creatoruser",
			"display_name": "Creator User",
			"account_id": "creator456"
		},
		"files": {
			"main.py": {
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/999/files/main.py"},
					"html": {"href": "https://bitbucket.org/snippets/myworkspace/999/main.py"}
				}
			},
			"config.json": {
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/999/files/config.json"},
					"html": {"href": "https://bitbucket.org/snippets/myworkspace/999/config.json"}
				}
			}
		},
		"links": {
			"self": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/999"},
			"html": {"href": "https://bitbucket.org/snippets/myworkspace/999"},
			"comments": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/999/comments"},
			"watchers": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/999/watchers"},
			"commits": {"href": "https://api.bitbucket.org/2.0/snippets/myworkspace/999/commits"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	snippet, err := client.GetSnippet(context.Background(), "myworkspace", "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify basic fields
	if snippet.Type != "snippet" {
		t.Errorf("expected type 'snippet', got %q", snippet.Type)
	}

	if snippet.ID != 999 {
		t.Errorf("expected ID 999, got %d", snippet.ID)
	}

	if snippet.Title != "Complete Test Snippet" {
		t.Errorf("expected title 'Complete Test Snippet', got %q", snippet.Title)
	}

	if snippet.Scm != "git" {
		t.Errorf("expected scm 'git', got %q", snippet.Scm)
	}

	if !snippet.IsPrivate {
		t.Error("expected is_private to be true")
	}

	if snippet.CreatedOn != "2024-01-15T10:30:00+00:00" {
		t.Errorf("expected created_on '2024-01-15T10:30:00+00:00', got %q", snippet.CreatedOn)
	}

	if snippet.UpdatedOn != "2024-02-20T14:45:00+00:00" {
		t.Errorf("expected updated_on '2024-02-20T14:45:00+00:00', got %q", snippet.UpdatedOn)
	}

	// Verify owner
	if snippet.Owner == nil {
		t.Fatal("expected Owner to not be nil")
	}
	if snippet.Owner.DisplayName != "Owner User" {
		t.Errorf("expected owner display_name 'Owner User', got %q", snippet.Owner.DisplayName)
	}

	// Verify creator
	if snippet.Creator == nil {
		t.Fatal("expected Creator to not be nil")
	}
	if snippet.Creator.DisplayName != "Creator User" {
		t.Errorf("expected creator display_name 'Creator User', got %q", snippet.Creator.DisplayName)
	}

	// Verify files
	if len(snippet.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(snippet.Files))
	}

	mainPy, ok := snippet.Files["main.py"]
	if !ok {
		t.Error("expected main.py in files")
	} else {
		if mainPy.Links.Self.Href == "" {
			t.Error("expected main.py self link to be set")
		}
	}

	configJson, ok := snippet.Files["config.json"]
	if !ok {
		t.Error("expected config.json in files")
	} else {
		if configJson.Links.HTML.Href == "" {
			t.Error("expected config.json html link to be set")
		}
	}

	// Verify links
	if snippet.Links.Self.Href == "" {
		t.Error("expected links.self to be set")
	}
	if snippet.Links.HTML.Href == "" {
		t.Error("expected links.html to be set")
	}
	if snippet.Links.Comments.Href == "" {
		t.Error("expected links.comments to be set")
	}
	if snippet.Links.Watchers.Href == "" {
		t.Error("expected links.watchers to be set")
	}
	if snippet.Links.Commits.Href == "" {
		t.Error("expected links.commits to be set")
	}
}

func TestSnippetErrorHandling(t *testing.T) {
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
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			response:       `{"error": {"message": "Forbidden", "detail": "You do not have permission"}}`,
			wantStatusCode: http.StatusForbidden,
			wantMessage:    "Forbidden",
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			response:       `{"error": {"message": "Snippet not found"}}`,
			wantStatusCode: http.StatusNotFound,
			wantMessage:    "Snippet not found",
		},
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			response:       `{"error": {"message": "Validation error", "fields": {"title": "Title is required"}}}`,
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

			_, err := client.GetSnippet(context.Background(), "workspace", "abc123")

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

func TestListSnippetsPagination(t *testing.T) {
	// Test that pagination response is properly parsed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 50,
			"page": 2,
			"pagelen": 10,
			"next": "https://api.bitbucket.org/2.0/snippets/myworkspace?page=3",
			"previous": "https://api.bitbucket.org/2.0/snippets/myworkspace?page=1",
			"values": [
				{"type": "snippet", "id": 11, "title": "Snippet 11"},
				{"type": "snippet", "id": 12, "title": "Snippet 12"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListSnippets(context.Background(), "myworkspace", &SnippetListOptions{Page: 2, Limit: 10})
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

func TestCreateSnippetMultipartBody(t *testing.T) {
	// Test that multipart body includes title field
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check title field
		title := r.FormValue("title")
		if title != "Test Title" {
			t.Errorf("expected title 'Test Title', got %q", title)
		}

		// Check is_private field
		isPrivate := r.FormValue("is_private")
		if isPrivate != "true" {
			t.Errorf("expected is_private 'true', got %q", isPrivate)
		}

		// Check file
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("failed to get file: %v", err)
		} else {
			defer file.Close()
			if header.Filename != "test.py" {
				t.Errorf("expected filename 'test.py', got %q", header.Filename)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"type": "snippet", "id": 1, "title": "Test Title"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

	_, err := client.CreateSnippet(context.Background(), "myworkspace", "Test Title", true, map[string]string{
		"test.py": "print('test')",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
