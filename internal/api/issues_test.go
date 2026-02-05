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

func TestListIssues(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		opts          *IssueListOptions
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
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"id": 1, "title": "First issue", "state": "open", "kind": "bug", "priority": "major"},
					{"id": 2, "title": "Second issue", "state": "new", "kind": "enhancement", "priority": "minor"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with state filter",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{State: "open"},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"q": `state="open"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Open issue", "state": "open"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with kind filter",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{Kind: "bug"},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"q": `kind="bug"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Bug issue", "kind": "bug"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with priority filter",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{Priority: "critical"},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"q": `priority="critical"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Critical issue", "priority": "critical"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with assignee filter",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{Assignee: "johndoe"},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"q": `assignee.username="johndoe"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Assigned issue"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with multiple filters",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{State: "open", Kind: "bug", Priority: "major"},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"q": `state="open" AND kind="bug" AND priority="major"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Open major bug"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with custom query",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{Q: `title~"important"`},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"q": `title~"important"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Important feature"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with pagination",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{Page: 2, Limit: 5},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"page": "2", "pagelen": "5"},
			response: `{
				"size": 15,
				"page": 2,
				"pagelen": 5,
				"next": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues?page=3",
				"previous": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues?page=1",
				"values": [{"id": 6, "title": "Issue 6"}, {"id": 7, "title": "Issue 7"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with sort",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &IssueListOptions{Sort: "-updated_on"},
			expectedURL: "/repositories/myworkspace/myrepo/issues",
			expectedQuery: map[string]string{"sort": "-updated_on"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Recently updated"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "handles 401 unauthorized",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			opts:       nil,
			response:   `{"error": {"message": "Unauthorized", "detail": "Authentication required"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "handles 403 forbidden",
			workspace:  "myworkspace",
			repoSlug:   "private-repo",
			opts:       nil,
			response:   `{"error": {"message": "Forbidden", "detail": "You do not have access to this repository"}}`,
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "handles 404 repository not found",
			workspace:  "myworkspace",
			repoSlug:   "nonexistent",
			opts:       nil,
			response:   `{"error": {"message": "Repository not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "handles 500 server error",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			opts:       nil,
			response:   `{"error": {"message": "Internal server error"}}`,
			statusCode: http.StatusInternalServerError,
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

			result, err := client.ListIssues(context.Background(), tt.workspace, tt.repoSlug, tt.opts)

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
				t.Errorf("expected %d issues, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetIssue(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		issueID    int
		response   string
		statusCode int
		wantErr    bool
		wantTitle  string
	}{
		{
			name:      "successfully get issue",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   42,
			response: `{
				"type": "issue",
				"id": 42,
				"title": "Test Issue",
				"content": {"raw": "Issue description", "markup": "markdown", "html": "<p>Issue description</p>"},
				"state": "open",
				"kind": "bug",
				"priority": "major",
				"reporter": {"display_name": "Reporter User", "uuid": "{reporter-uuid}"},
				"assignee": {"display_name": "Assignee User", "uuid": "{assignee-uuid}"},
				"votes": 5,
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-15T12:00:00Z",
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/42"},
					"html": {"href": "https://bitbucket.org/myworkspace/myrepo/issues/42"},
					"comments": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/42/comments"},
					"attachments": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/42/attachments"}
				}
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "Test Issue",
		},
		{
			name:       "issue not found",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    999,
			response:   `{"error": {"message": "Issue not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "unauthorized access",
			workspace:  "private-workspace",
			repoSlug:   "private-repo",
			issueID:    1,
			response:   `{"error": {"message": "Unauthorized", "detail": "You do not have access to this repository"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "forbidden access",
			workspace:  "myworkspace",
			repoSlug:   "restricted-repo",
			issueID:    1,
			response:   `{"error": {"message": "Forbidden", "detail": "Issues are disabled for this repository"}}`,
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "server error",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			response:   `{"error": {"message": "Internal server error"}}`,
			statusCode: http.StatusInternalServerError,
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

			result, err := client.GetIssue(context.Background(), tt.workspace, tt.repoSlug, tt.issueID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL contains workspace, repo, and issue ID
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/issues/42"
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path to contain %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify response parsing
			if result.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, result.Title)
			}
		})
	}
}

func TestCreateIssue(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		opts       *IssueCreateOptions
		response   string
		statusCode int
		wantErr    bool
		wantTitle  string
	}{
		{
			name:      "create issue with all fields",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &IssueCreateOptions{
				Title:    "Full Issue",
				Content:  &Content{Raw: "Detailed description of the issue"},
				Kind:     "bug",
				Priority: "critical",
				Assignee: &User{UUID: "{user-uuid}"},
			},
			response: `{
				"type": "issue",
				"id": 1,
				"title": "Full Issue",
				"content": {"raw": "Detailed description of the issue"},
				"state": "new",
				"kind": "bug",
				"priority": "critical",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantTitle:  "Full Issue",
		},
		{
			name:      "create issue with minimal fields",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &IssueCreateOptions{
				Title: "Minimal Issue",
			},
			response: `{
				"type": "issue",
				"id": 2,
				"title": "Minimal Issue",
				"state": "new",
				"kind": "bug",
				"priority": "major",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantTitle:  "Minimal Issue",
		},
		{
			name:      "create enhancement issue",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &IssueCreateOptions{
				Title:    "Enhancement Request",
				Content:  &Content{Raw: "This is a feature request"},
				Kind:     "enhancement",
				Priority: "minor",
			},
			response: `{
				"type": "issue",
				"id": 3,
				"title": "Enhancement Request",
				"kind": "enhancement",
				"priority": "minor",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantTitle:  "Enhancement Request",
		},
		{
			name:      "create issue fails - validation error",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &IssueCreateOptions{
				Title: "", // Empty title should fail
			},
			response:   `{"error": {"message": "Validation error", "fields": {"title": "Title is required"}}}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:      "create issue fails - unauthorized",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &IssueCreateOptions{
				Title: "Test Issue",
			},
			response:   `{"error": {"message": "Unauthorized"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:      "create issue fails - issues disabled",
			workspace: "myworkspace",
			repoSlug:  "no-issues-repo",
			opts: &IssueCreateOptions{
				Title: "Test Issue",
			},
			response:   `{"error": {"message": "Issues are disabled for this repository"}}`,
			statusCode: http.StatusForbidden,
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

			result, err := client.CreateIssue(context.Background(), tt.workspace, tt.repoSlug, tt.opts)

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
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/issues"
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

			// Verify title
			if body["title"] != tt.opts.Title {
				t.Errorf("expected title %q in body, got %v", tt.opts.Title, body["title"])
			}

			// Verify content if provided
			if tt.opts.Content != nil && tt.opts.Content.Raw != "" {
				content, ok := body["content"].(map[string]interface{})
				if !ok {
					t.Error("expected content object in body")
				} else if content["raw"] != tt.opts.Content.Raw {
					t.Errorf("expected content raw %q, got %v", tt.opts.Content.Raw, content["raw"])
				}
			}

			// Verify assignee if provided
			if tt.opts.Assignee != nil && tt.opts.Assignee.UUID != "" {
				assignee, ok := body["assignee"].(map[string]interface{})
				if !ok {
					t.Error("expected assignee object in body")
				} else if assignee["uuid"] != tt.opts.Assignee.UUID {
					t.Errorf("expected assignee uuid %q, got %v", tt.opts.Assignee.UUID, assignee["uuid"])
				}
			}

			// Verify result
			if result.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, result.Title)
			}
		})
	}
}

func TestUpdateIssue(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		issueID    int
		opts       *IssueUpdateOptions
		response   string
		statusCode int
		wantErr    bool
		wantTitle  string
	}{
		{
			name:      "partial update - title only",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			opts: &IssueUpdateOptions{
				Title: strPtr("Updated Title"),
			},
			response: `{
				"type": "issue",
				"id": 1,
				"title": "Updated Title",
				"state": "open",
				"kind": "bug",
				"priority": "major",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "Updated Title",
		},
		{
			name:      "partial update - state only",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			opts: &IssueUpdateOptions{
				State: strPtr("resolved"),
			},
			response: `{
				"type": "issue",
				"id": 1,
				"title": "Original Title",
				"state": "resolved",
				"kind": "bug",
				"priority": "major",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "Original Title",
		},
		{
			name:      "full update - all fields",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			opts: &IssueUpdateOptions{
				Title:    strPtr("Fully Updated Issue"),
				Content:  &Content{Raw: "New description"},
				State:    strPtr("open"),
				Kind:     strPtr("enhancement"),
				Priority: strPtr("critical"),
				Assignee: &User{UUID: "{new-assignee-uuid}"},
			},
			response: `{
				"type": "issue",
				"id": 1,
				"title": "Fully Updated Issue",
				"content": {"raw": "New description"},
				"state": "open",
				"kind": "enhancement",
				"priority": "critical",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "Fully Updated Issue",
		},
		{
			name:      "update issue fails - not found",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   999,
			opts: &IssueUpdateOptions{
				Title: strPtr("Updated Title"),
			},
			response:   `{"error": {"message": "Issue not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:      "update issue fails - unauthorized",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			opts: &IssueUpdateOptions{
				Title: strPtr("Updated Title"),
			},
			response:   `{"error": {"message": "Unauthorized"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:      "update issue fails - forbidden",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			opts: &IssueUpdateOptions{
				Title: strPtr("Updated Title"),
			},
			response:   `{"error": {"message": "Forbidden", "detail": "You do not have permission to update this issue"}}`,
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:      "update issue fails - validation error",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			opts: &IssueUpdateOptions{
				State: strPtr("invalid_state"),
			},
			response:   `{"error": {"message": "Validation error", "fields": {"state": "Invalid state value"}}}`,
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

			result, err := client.UpdateIssue(context.Background(), tt.workspace, tt.repoSlug, tt.issueID, tt.opts)

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

			// Verify URL contains issue ID
			if !strings.Contains(receivedReq.URL.Path, "/issues/") {
				t.Errorf("expected URL path to contain /issues/, got %s", receivedReq.URL.Path)
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

			// Verify title if provided
			if tt.opts.Title != nil {
				if body["title"] != *tt.opts.Title {
					t.Errorf("expected title %q in body, got %v", *tt.opts.Title, body["title"])
				}
			}

			// Verify state if provided
			if tt.opts.State != nil {
				if body["state"] != *tt.opts.State {
					t.Errorf("expected state %q in body, got %v", *tt.opts.State, body["state"])
				}
			}

			// Verify result
			if result.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, result.Title)
			}
		})
	}
}

func TestDeleteIssue(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		issueID    int
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful deletion",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			statusCode: http.StatusNoContent,
			response:   "",
			wantErr:    false,
		},
		{
			name:       "issue not found",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    999,
			statusCode: http.StatusNotFound,
			response:   `{"error": {"message": "Issue not found"}}`,
			wantErr:    true,
		},
		{
			name:       "unauthorized deletion",
			workspace:  "other-workspace",
			repoSlug:   "other-repo",
			issueID:    1,
			statusCode: http.StatusUnauthorized,
			response:   `{"error": {"message": "Unauthorized", "detail": "You do not have permission to delete this issue"}}`,
			wantErr:    true,
		},
		{
			name:       "forbidden deletion",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			statusCode: http.StatusForbidden,
			response:   `{"error": {"message": "Forbidden", "detail": "Only repository admins can delete issues"}}`,
			wantErr:    true,
		},
		{
			name:       "server error",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			statusCode: http.StatusInternalServerError,
			response:   `{"error": {"message": "Internal server error"}}`,
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

			err := client.DeleteIssue(context.Background(), tt.workspace, tt.repoSlug, tt.issueID)

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

			// Verify URL contains workspace, repo, and issue ID
			if !strings.Contains(receivedReq.URL.Path, "/issues/") {
				t.Errorf("expected URL path to contain /issues/, got %s", receivedReq.URL.Path)
			}
		})
	}
}

func TestListIssueComments(t *testing.T) {
	tests := []struct {
		name        string
		workspace   string
		repoSlug    string
		issueID     int
		expectedURL string
		response    string
		statusCode  int
		wantErr     bool
		wantCount   int
	}{
		{
			name:        "list comments successfully",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			issueID:     1,
			expectedURL: "/repositories/myworkspace/myrepo/issues/1/comments",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"id": 100,
						"content": {"raw": "First comment", "html": "<p>First comment</p>"},
						"user": {"display_name": "User One", "uuid": "{user-1}"},
						"created_on": "2024-01-01T00:00:00Z",
						"updated_on": "2024-01-01T00:00:00Z"
					},
					{
						"id": 101,
						"content": {"raw": "Second comment", "html": "<p>Second comment</p>"},
						"user": {"display_name": "User Two", "uuid": "{user-2}"},
						"created_on": "2024-01-02T00:00:00Z",
						"updated_on": "2024-01-02T00:00:00Z"
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list comments with pagination",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			issueID:     1,
			expectedURL: "/repositories/myworkspace/myrepo/issues/1/comments",
			response: `{
				"size": 50,
				"page": 2,
				"pagelen": 10,
				"next": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/1/comments?page=3",
				"previous": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/1/comments?page=1",
				"values": [
					{"id": 110, "content": {"raw": "Comment 11"}},
					{"id": 111, "content": {"raw": "Comment 12"}}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "empty comments list",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			issueID:     1,
			expectedURL: "/repositories/myworkspace/myrepo/issues/1/comments",
			response: `{
				"size": 0,
				"page": 1,
				"pagelen": 10,
				"values": []
			}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "issue not found",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    999,
			response:   `{"error": {"message": "Issue not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			response:   `{"error": {"message": "Unauthorized"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "forbidden",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			response:   `{"error": {"message": "Forbidden"}}`,
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "server error",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			response:   `{"error": {"message": "Internal server error"}}`,
			statusCode: http.StatusInternalServerError,
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

			result, err := client.ListIssueComments(context.Background(), tt.workspace, tt.repoSlug, tt.issueID)

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

			// Verify URL path
			if tt.expectedURL != "" && !strings.HasSuffix(receivedReq.URL.Path, tt.expectedURL) {
				t.Errorf("expected URL path to end with %q, got %q", tt.expectedURL, receivedReq.URL.Path)
			}

			// Verify result count
			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d comments, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestCreateIssueComment(t *testing.T) {
	tests := []struct {
		name       string
		workspace  string
		repoSlug   string
		issueID    int
		body       string
		response   string
		statusCode int
		wantErr    bool
		wantID     int
	}{
		{
			name:      "create comment successfully",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			body:      "This is a new comment",
			response: `{
				"id": 200,
				"content": {"raw": "This is a new comment", "html": "<p>This is a new comment</p>"},
				"user": {"display_name": "Comment Author", "uuid": "{author-uuid}"},
				"created_on": "2024-01-15T10:00:00Z",
				"updated_on": "2024-01-15T10:00:00Z",
				"links": {
					"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/1/comments/200"},
					"html": {"href": "https://bitbucket.org/myworkspace/myrepo/issues/1#comment-200"}
				}
			}`,
			statusCode: http.StatusCreated,
			wantID:     200,
		},
		{
			name:      "create comment with markdown",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			issueID:   1,
			body:      "**Bold** and _italic_ text with `code`",
			response: `{
				"id": 201,
				"content": {"raw": "**Bold** and _italic_ text with ` + "`code`" + `", "html": "<p><strong>Bold</strong> and <em>italic</em> text with <code>code</code></p>"},
				"user": {"display_name": "Comment Author"},
				"created_on": "2024-01-15T10:00:00Z",
				"updated_on": "2024-01-15T10:00:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantID:     201,
		},
		{
			name:       "create comment fails - issue not found",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    999,
			body:       "Comment on nonexistent issue",
			response:   `{"error": {"message": "Issue not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "create comment fails - unauthorized",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			body:       "Unauthorized comment",
			response:   `{"error": {"message": "Unauthorized"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "create comment fails - forbidden",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			body:       "Forbidden comment",
			response:   `{"error": {"message": "Forbidden", "detail": "Comments are disabled for this issue"}}`,
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "create comment fails - empty body",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			body:       "",
			response:   `{"error": {"message": "Validation error", "fields": {"content": "Content is required"}}}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "create comment fails - server error",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			issueID:    1,
			body:       "Some comment",
			response:   `{"error": {"message": "Internal server error"}}`,
			statusCode: http.StatusInternalServerError,
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

			result, err := client.CreateIssueComment(context.Background(), tt.workspace, tt.repoSlug, tt.issueID, tt.body)

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

			// Verify URL contains comments endpoint
			if !strings.Contains(receivedReq.URL.Path, "/comments") {
				t.Errorf("expected URL path to contain /comments, got %s", receivedReq.URL.Path)
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

			// Verify content.raw
			content, ok := body["content"].(map[string]interface{})
			if !ok {
				t.Error("expected content object in body")
			} else if content["raw"] != tt.body {
				t.Errorf("expected content raw %q, got %v", tt.body, content["raw"])
			}

			// Verify result
			if result.ID != tt.wantID {
				t.Errorf("expected ID %d, got %d", tt.wantID, result.ID)
			}
		})
	}
}

func TestIssueParsing(t *testing.T) {
	// Test comprehensive issue response parsing with all fields
	responseJSON := `{
		"type": "issue",
		"id": 42,
		"title": "Complete Issue for Testing",
		"content": {
			"raw": "This is the **issue** description",
			"markup": "markdown",
			"html": "<p>This is the <strong>issue</strong> description</p>"
		},
		"state": "open",
		"kind": "bug",
		"priority": "critical",
		"reporter": {
			"uuid": "{reporter-uuid}",
			"username": "reporter",
			"display_name": "Issue Reporter",
			"account_id": "reporter123"
		},
		"assignee": {
			"uuid": "{assignee-uuid}",
			"username": "assignee",
			"display_name": "Issue Assignee",
			"account_id": "assignee456"
		},
		"repository": {
			"uuid": "{repo-uuid}",
			"name": "myrepo",
			"full_name": "myworkspace/myrepo"
		},
		"votes": 10,
		"created_on": "2024-01-15T10:30:00Z",
		"updated_on": "2024-02-20T14:45:00Z",
		"links": {
			"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/42"},
			"html": {"href": "https://bitbucket.org/myworkspace/myrepo/issues/42"},
			"comments": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/42/comments"},
			"attachments": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/42/attachments"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	issue, err := client.GetIssue(context.Background(), "myworkspace", "myrepo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are parsed correctly
	if issue.Type != "issue" {
		t.Errorf("expected type 'issue', got %q", issue.Type)
	}

	if issue.ID != 42 {
		t.Errorf("expected ID 42, got %d", issue.ID)
	}

	if issue.Title != "Complete Issue for Testing" {
		t.Errorf("expected title 'Complete Issue for Testing', got %q", issue.Title)
	}

	if issue.State != "open" {
		t.Errorf("expected state 'open', got %q", issue.State)
	}

	if issue.Kind != "bug" {
		t.Errorf("expected kind 'bug', got %q", issue.Kind)
	}

	if issue.Priority != "critical" {
		t.Errorf("expected priority 'critical', got %q", issue.Priority)
	}

	if issue.Votes != 10 {
		t.Errorf("expected votes 10, got %d", issue.Votes)
	}

	// Verify content
	if issue.Content == nil {
		t.Fatal("expected Content to not be nil")
	}
	if issue.Content.Raw != "This is the **issue** description" {
		t.Errorf("expected content raw 'This is the **issue** description', got %q", issue.Content.Raw)
	}
	if issue.Content.Markup != "markdown" {
		t.Errorf("expected content markup 'markdown', got %q", issue.Content.Markup)
	}
	if issue.Content.HTML != "<p>This is the <strong>issue</strong> description</p>" {
		t.Errorf("expected content html to be set, got %q", issue.Content.HTML)
	}

	// Verify reporter
	if issue.Reporter == nil {
		t.Fatal("expected Reporter to not be nil")
	}
	if issue.Reporter.DisplayName != "Issue Reporter" {
		t.Errorf("expected reporter display_name 'Issue Reporter', got %q", issue.Reporter.DisplayName)
	}

	// Verify assignee
	if issue.Assignee == nil {
		t.Fatal("expected Assignee to not be nil")
	}
	if issue.Assignee.DisplayName != "Issue Assignee" {
		t.Errorf("expected assignee display_name 'Issue Assignee', got %q", issue.Assignee.DisplayName)
	}

	// Verify repository
	if issue.Repository == nil {
		t.Fatal("expected Repository to not be nil")
	}
	if issue.Repository.FullName != "myworkspace/myrepo" {
		t.Errorf("expected repository full_name 'myworkspace/myrepo', got %q", issue.Repository.FullName)
	}

	// Verify links
	if issue.Links == nil {
		t.Fatal("expected Links to not be nil")
	}
	if issue.Links.Self == nil || issue.Links.Self.Href == "" {
		t.Error("expected links.self to be set")
	}
	if issue.Links.HTML == nil || issue.Links.HTML.Href == "" {
		t.Error("expected links.html to be set")
	}
	if issue.Links.Comments == nil || issue.Links.Comments.Href == "" {
		t.Error("expected links.comments to be set")
	}

	// Verify time parsing
	expectedCreated := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !issue.CreatedOn.Equal(expectedCreated) {
		t.Errorf("expected created_on %v, got %v", expectedCreated, issue.CreatedOn)
	}

	expectedUpdated := time.Date(2024, 2, 20, 14, 45, 0, 0, time.UTC)
	if !issue.UpdatedOn.Equal(expectedUpdated) {
		t.Errorf("expected updated_on %v, got %v", expectedUpdated, issue.UpdatedOn)
	}
}

func TestIssueCommentParsing(t *testing.T) {
	// Test comprehensive issue comment response parsing
	responseJSON := `{
		"size": 1,
		"page": 1,
		"pagelen": 10,
		"values": [{
			"id": 100,
			"content": {
				"raw": "This is a **comment** with markdown",
				"markup": "markdown",
				"html": "<p>This is a <strong>comment</strong> with markdown</p>"
			},
			"user": {
				"uuid": "{user-uuid}",
				"username": "commenter",
				"display_name": "Comment Author",
				"account_id": "commenter123"
			},
			"created_on": "2024-01-20T09:15:00Z",
			"updated_on": "2024-01-20T10:30:00Z",
			"links": {
				"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues/1/comments/100"},
				"html": {"href": "https://bitbucket.org/myworkspace/myrepo/issues/1#comment-100"}
			}
		}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListIssueComments(context.Background(), "myworkspace", "myrepo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Values) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result.Values))
	}

	comment := result.Values[0]

	// Verify all fields are parsed correctly
	if comment.ID != 100 {
		t.Errorf("expected ID 100, got %d", comment.ID)
	}

	// Verify content
	if comment.Content == nil {
		t.Fatal("expected Content to not be nil")
	}
	if comment.Content.Raw != "This is a **comment** with markdown" {
		t.Errorf("expected content raw 'This is a **comment** with markdown', got %q", comment.Content.Raw)
	}

	// Verify user
	if comment.User == nil {
		t.Fatal("expected User to not be nil")
	}
	if comment.User.DisplayName != "Comment Author" {
		t.Errorf("expected user display_name 'Comment Author', got %q", comment.User.DisplayName)
	}

	// Verify links
	if comment.Links == nil {
		t.Fatal("expected Links to not be nil")
	}
	if comment.Links.Self == nil || comment.Links.Self.Href == "" {
		t.Error("expected links.self to be set")
	}

	// Verify time parsing
	expectedCreated := time.Date(2024, 1, 20, 9, 15, 0, 0, time.UTC)
	if !comment.CreatedOn.Equal(expectedCreated) {
		t.Errorf("expected created_on %v, got %v", expectedCreated, comment.CreatedOn)
	}

	expectedUpdated := time.Date(2024, 1, 20, 10, 30, 0, 0, time.UTC)
	if !comment.UpdatedOn.Equal(expectedUpdated) {
		t.Errorf("expected updated_on %v, got %v", expectedUpdated, comment.UpdatedOn)
	}
}

func TestIssueErrorHandling(t *testing.T) {
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
			response:       `{"error": {"message": "Forbidden", "detail": "Issues are disabled"}}`,
			wantStatusCode: http.StatusForbidden,
			wantMessage:    "Forbidden",
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			response:       `{"error": {"message": "Issue not found"}}`,
			wantStatusCode: http.StatusNotFound,
			wantMessage:    "Issue not found",
		},
		{
			name:           "400 Bad Request with fields",
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

			_, err := client.GetIssue(context.Background(), "workspace", "repo", 1)

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

func TestListIssuesPagination(t *testing.T) {
	// Test that pagination response is properly parsed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 100,
			"page": 2,
			"pagelen": 10,
			"next": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues?page=3",
			"previous": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/issues?page=1",
			"values": [
				{"id": 11, "title": "Issue 11"},
				{"id": 12, "title": "Issue 12"}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListIssues(context.Background(), "myworkspace", "myrepo", &IssueListOptions{Page: 2, Limit: 10})
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
