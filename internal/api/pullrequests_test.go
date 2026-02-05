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

func TestListPullRequests(t *testing.T) {
	tests := []struct {
		name           string
		opts           *PRListOptions
		expectedURL    string
		expectedQuery  map[string]string
		response       string
		statusCode     int
		wantErr        bool
		wantCount      int
	}{
		{
			name: "basic list without options",
			opts: nil,
			expectedURL: "/repositories/myworkspace/myrepo/pullrequests",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"id": 1, "title": "First PR", "state": "OPEN"},
					{"id": 2, "title": "Second PR", "state": "MERGED"}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "list with state filter",
			opts: &PRListOptions{State: PRStateOpen},
			expectedURL: "/repositories/myworkspace/myrepo/pullrequests",
			expectedQuery: map[string]string{"state": "OPEN"},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 1, "title": "Open PR", "state": "OPEN"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "list with pagination",
			opts: &PRListOptions{Page: 2, Limit: 5},
			expectedURL: "/repositories/myworkspace/myrepo/pullrequests",
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
			name: "list with author filter",
			opts: &PRListOptions{Author: "testuser"},
			expectedURL: "/repositories/myworkspace/myrepo/pullrequests",
			expectedQuery: map[string]string{"q": `author.username="testuser"`},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [{"id": 3, "title": "User PR", "state": "OPEN"}]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "handles 401 unauthorized",
			opts: nil,
			response: `{"error": {"message": "Unauthorized", "detail": "Authentication required"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "handles 404 not found",
			opts: nil,
			response: `{"error": {"message": "Repository not found"}}`,
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

			result, err := client.ListPullRequests(context.Background(), "myworkspace", "myrepo", tt.opts)

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
				t.Errorf("expected %d pull requests, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetPullRequest(t *testing.T) {
	tests := []struct {
		name       string
		prID       int64
		response   string
		statusCode int
		wantErr    bool
		wantTitle  string
	}{
		{
			name: "successfully get PR",
			prID: 123,
			response: `{
				"id": 123,
				"title": "Test Pull Request",
				"description": "This is a test PR",
				"state": "OPEN",
				"author": {
					"display_name": "Test User",
					"uuid": "{user-uuid}"
				},
				"source": {
					"branch": {"name": "feature-branch"},
					"commit": {"hash": "abc123"}
				},
				"destination": {
					"branch": {"name": "main"},
					"commit": {"hash": "def456"}
				},
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z",
				"links": {
					"html": {"href": "https://bitbucket.org/workspace/repo/pull-requests/123"},
					"diff": {"href": "https://api.bitbucket.org/2.0/repositories/workspace/repo/pullrequests/123/diff"}
				}
			}`,
			statusCode: http.StatusOK,
			wantTitle:  "Test Pull Request",
		},
		{
			name:       "PR not found",
			prID:       999,
			response:   `{"error": {"message": "Pull request not found"}}`,
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

			result, err := client.GetPullRequest(context.Background(), "workspace", "repo", tt.prID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL contains PR ID
			expectedPath := "/repositories/workspace/repo/pullrequests/123"
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path to contain PR ID, got %s", receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify response parsing
			if result.Title != tt.wantTitle {
				t.Errorf("expected title %q, got %q", tt.wantTitle, result.Title)
			}

			if result.ID != tt.prID {
				t.Errorf("expected ID %d, got %d", tt.prID, result.ID)
			}
		})
	}
}

func TestCreatePullRequest(t *testing.T) {
	tests := []struct {
		name             string
		opts             *PRCreateOptions
		expectedBody     map[string]interface{}
		response         string
		statusCode       int
		wantErr          bool
		wantID           int64
	}{
		{
			name: "basic PR creation",
			opts: &PRCreateOptions{
				Title:             "New Feature",
				Description:       "This adds a new feature",
				SourceBranch:      "feature/my-feature",
				DestinationBranch: "main",
				CloseSourceBranch: true,
			},
			response: `{
				"id": 456,
				"title": "New Feature",
				"state": "OPEN",
				"source": {"branch": {"name": "feature/my-feature"}},
				"destination": {"branch": {"name": "main"}},
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantID:     456,
		},
		{
			name: "PR creation with reviewers",
			opts: &PRCreateOptions{
				Title:             "Review Required",
				SourceBranch:      "feature/review",
				DestinationBranch: "main",
				Reviewers:         []string{"{uuid-1}", "{uuid-2}"},
			},
			response: `{
				"id": 789,
				"title": "Review Required",
				"state": "OPEN",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-01T00:00:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantID:     789,
		},
		{
			name: "PR creation fails - branch not found",
			opts: &PRCreateOptions{
				Title:             "Invalid",
				SourceBranch:      "nonexistent",
				DestinationBranch: "main",
			},
			response:   `{"error": {"message": "Source branch not found"}}`,
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

			result, err := client.CreatePullRequest(context.Background(), "workspace", "repo", tt.opts)

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
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
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

			if body["title"] != tt.opts.Title {
				t.Errorf("expected title %q in body, got %q", tt.opts.Title, body["title"])
			}

			// Check source branch
			source, ok := body["source"].(map[string]interface{})
			if !ok {
				t.Error("expected source object in body")
			} else {
				branch, _ := source["branch"].(map[string]interface{})
				if branch["name"] != tt.opts.SourceBranch {
					t.Errorf("expected source branch %q, got %q", tt.opts.SourceBranch, branch["name"])
				}
			}

			// Verify result
			if result.ID != tt.wantID {
				t.Errorf("expected ID %d, got %d", tt.wantID, result.ID)
			}
		})
	}
}

func TestMergePullRequest(t *testing.T) {
	tests := []struct {
		name       string
		prID       int64
		opts       *PRMergeOptions
		response   string
		statusCode int
		wantErr    bool
		wantState  PRState
	}{
		{
			name: "basic merge",
			prID: 100,
			opts: nil,
			response: `{
				"id": 100,
				"title": "Merged PR",
				"state": "MERGED",
				"merge_commit": {"hash": "merged123"},
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantState:  PRStateMerged,
		},
		{
			name: "merge with squash strategy",
			prID: 101,
			opts: &PRMergeOptions{
				Message:       "Squash merge",
				MergeStrategy: MergeStrategySquash,
			},
			response: `{
				"id": 101,
				"state": "MERGED",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantState:  PRStateMerged,
		},
		{
			name: "merge with close source branch",
			prID: 102,
			opts: &PRMergeOptions{
				CloseSourceBranch: true,
				MergeStrategy:     MergeStrategyMergeCommit,
			},
			response: `{
				"id": 102,
				"state": "MERGED",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantState:  PRStateMerged,
		},
		{
			name:       "merge conflict",
			prID:       103,
			opts:       nil,
			response:   `{"error": {"message": "Cannot merge due to conflicts"}}`,
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

			result, err := client.MergePullRequest(context.Background(), "workspace", "repo", tt.prID, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL path contains merge endpoint
			expectedPath := "/repositories/workspace/repo/pullrequests/100/merge"
			if tt.prID == 100 && !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %q", expectedPath, receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			// Verify merge options in body if provided
			if tt.opts != nil && len(receivedBody) > 0 {
				var body map[string]interface{}
				if err := json.Unmarshal(receivedBody, &body); err != nil {
					t.Fatalf("failed to parse request body: %v", err)
				}

				if tt.opts.MergeStrategy != "" {
					if body["merge_strategy"] != string(tt.opts.MergeStrategy) {
						t.Errorf("expected merge_strategy %q, got %v", tt.opts.MergeStrategy, body["merge_strategy"])
					}
				}
			}

			// Verify result state
			if result.State != tt.wantState {
				t.Errorf("expected state %q, got %q", tt.wantState, result.State)
			}
		})
	}
}

func TestDeclinePullRequest(t *testing.T) {
	tests := []struct {
		name       string
		prID       int64
		response   string
		statusCode int
		wantErr    bool
		wantState  PRState
	}{
		{
			name: "successful decline",
			prID: 200,
			response: `{
				"id": 200,
				"title": "Declined PR",
				"state": "DECLINED",
				"reason": "No longer needed",
				"created_on": "2024-01-01T00:00:00Z",
				"updated_on": "2024-01-02T00:00:00Z"
			}`,
			statusCode: http.StatusOK,
			wantState:  PRStateDeclined,
		},
		{
			name:       "decline already merged PR",
			prID:       201,
			response:   `{"error": {"message": "Cannot decline a merged pull request"}}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "PR not found",
			prID:       999,
			response:   `{"error": {"message": "Pull request not found"}}`,
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

			result, err := client.DeclinePullRequest(context.Background(), "workspace", "repo", tt.prID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL path contains decline endpoint
			if !strings.Contains(receivedReq.URL.Path, "/decline") {
				t.Errorf("expected URL path to contain /decline, got %s", receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			// Verify result state
			if result.State != tt.wantState {
				t.Errorf("expected state %q, got %q", tt.wantState, result.State)
			}
		})
	}
}

func TestApprovePullRequest(t *testing.T) {
	tests := []struct {
		name       string
		prID       int64
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful approval",
			prID: 300,
			response: `{
				"user": {
					"display_name": "Approver",
					"uuid": "{approver-uuid}"
				},
				"role": "REVIEWER",
				"approved": true,
				"state": "approved"
			}`,
			statusCode: http.StatusOK,
		},
		{
			name:       "cannot approve own PR",
			prID:       301,
			response:   `{"error": {"message": "You cannot approve your own pull request"}}`,
			statusCode: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "already approved",
			prID:       302,
			response: `{
				"user": {"display_name": "Approver"},
				"approved": true
			}`,
			statusCode: http.StatusOK,
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

			result, err := client.ApprovePullRequest(context.Background(), "workspace", "repo", tt.prID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL path contains approve endpoint
			if !strings.Contains(receivedReq.URL.Path, "/approve") {
				t.Errorf("expected URL path to contain /approve, got %s", receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodPost {
				t.Errorf("expected POST method, got %s", receivedReq.Method)
			}

			// Verify approval status
			if !result.Approved {
				t.Error("expected Approved to be true")
			}
		})
	}
}

func TestGetPullRequestDiff(t *testing.T) {
	tests := []struct {
		name       string
		prID       int64
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful diff retrieval",
			prID: 400,
			response: `diff --git a/file.txt b/file.txt
index abc123..def456 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line 1
 line 2
+new line
 line 3`,
			statusCode: http.StatusOK,
		},
		{
			name:       "PR not found for diff",
			prID:       999,
			response:   `{"error": {"message": "Pull request not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				if tt.statusCode == http.StatusOK {
					w.Header().Set("Content-Type", "text/plain")
				} else {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.GetPullRequestDiff(context.Background(), "workspace", "repo", tt.prID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL path contains diff endpoint
			if !strings.Contains(receivedReq.URL.Path, "/diff") {
				t.Errorf("expected URL path to contain /diff, got %s", receivedReq.URL.Path)
			}

			// Verify Accept header for plain text
			if accept := receivedReq.Header.Get("Accept"); accept != "text/plain" {
				t.Errorf("expected Accept header text/plain, got %s", accept)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify diff content is returned as plain text string
			if !strings.Contains(result, "diff --git") {
				t.Error("expected diff content to contain 'diff --git'")
			}

			if result != tt.response {
				t.Errorf("expected response %q, got %q", tt.response, result)
			}
		})
	}
}

func TestUnapprovePullRequest(t *testing.T) {
	tests := []struct {
		name       string
		prID       int64
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful unapproval",
			prID:       500,
			statusCode: http.StatusNoContent,
		},
		{
			name:       "PR not found",
			prID:       999,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					w.Write([]byte(`{"error": {"message": "Not found"}}`))
				}
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			err := client.UnapprovePullRequest(context.Background(), "workspace", "repo", tt.prID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL path contains approve endpoint
			if !strings.Contains(receivedReq.URL.Path, "/approve") {
				t.Errorf("expected URL path to contain /approve, got %s", receivedReq.URL.Path)
			}

			// Verify HTTP method is DELETE
			if receivedReq.Method != http.MethodDelete {
				t.Errorf("expected DELETE method, got %s", receivedReq.Method)
			}
		})
	}
}

func TestRequestChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the endpoint
		if !strings.Contains(r.URL.Path, "/request-changes") {
			http.Error(w, "wrong endpoint", http.StatusBadRequest)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"user": {"display_name": "Reviewer"},
			"role": "REVIEWER",
			"approved": false,
			"state": "changes_requested"
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

	result, err := client.RequestChanges(context.Background(), "workspace", "repo", 600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Approved {
		t.Error("expected Approved to be false for changes requested")
	}

	if result.State != "changes_requested" {
		t.Errorf("expected state 'changes_requested', got %q", result.State)
	}
}

func TestPullRequestParsing(t *testing.T) {
	// Test full PR response parsing with all fields
	responseJSON := `{
		"id": 1,
		"title": "Complete Feature",
		"description": "A detailed description",
		"state": "OPEN",
		"author": {
			"display_name": "John Doe",
			"uuid": "{user-uuid}",
			"account_id": "123456"
		},
		"source": {
			"branch": {"name": "feature/complete"},
			"commit": {"hash": "abc123"},
			"repository": {
				"full_name": "workspace/repo"
			}
		},
		"destination": {
			"branch": {"name": "main"},
			"commit": {"hash": "def456"}
		},
		"close_source_branch": true,
		"created_on": "2024-06-15T10:30:00Z",
		"updated_on": "2024-06-16T14:45:00Z",
		"links": {
			"html": {"href": "https://bitbucket.org/workspace/repo/pull-requests/1"},
			"diff": {"href": "https://api.bitbucket.org/2.0/repositories/workspace/repo/pullrequests/1/diff"},
			"comments": {"href": "https://api.bitbucket.org/2.0/repositories/workspace/repo/pullrequests/1/comments"}
		},
		"comment_count": 5,
		"task_count": 2,
		"reviewers": [
			{"display_name": "Reviewer 1", "uuid": "{rev1-uuid}"},
			{"display_name": "Reviewer 2", "uuid": "{rev2-uuid}"}
		],
		"participants": [
			{"user": {"display_name": "Participant"}, "role": "PARTICIPANT", "approved": false}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	pr, err := client.GetPullRequest(context.Background(), "workspace", "repo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are parsed correctly
	if pr.ID != 1 {
		t.Errorf("expected ID 1, got %d", pr.ID)
	}

	if pr.Title != "Complete Feature" {
		t.Errorf("expected title 'Complete Feature', got %q", pr.Title)
	}

	if pr.State != PRStateOpen {
		t.Errorf("expected state OPEN, got %q", pr.State)
	}

	if pr.Author.DisplayName != "John Doe" {
		t.Errorf("expected author 'John Doe', got %q", pr.Author.DisplayName)
	}

	if pr.Source.Branch.Name != "feature/complete" {
		t.Errorf("expected source branch 'feature/complete', got %q", pr.Source.Branch.Name)
	}

	if pr.Destination.Branch.Name != "main" {
		t.Errorf("expected destination branch 'main', got %q", pr.Destination.Branch.Name)
	}

	if !pr.CloseSourceBranch {
		t.Error("expected CloseSourceBranch to be true")
	}

	if pr.CommentCount != 5 {
		t.Errorf("expected comment_count 5, got %d", pr.CommentCount)
	}

	if pr.TaskCount != 2 {
		t.Errorf("expected task_count 2, got %d", pr.TaskCount)
	}

	if len(pr.Reviewers) != 2 {
		t.Errorf("expected 2 reviewers, got %d", len(pr.Reviewers))
	}

	// Verify time parsing
	expectedCreated := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	if !pr.CreatedOn.Equal(expectedCreated) {
		t.Errorf("expected created_on %v, got %v", expectedCreated, pr.CreatedOn)
	}
}

func TestUpdatePullRequest(t *testing.T) {
	var receivedBody []byte
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 700,
			"title": "Updated Title",
			"description": "Updated description",
			"state": "OPEN",
			"created_on": "2024-01-01T00:00:00Z",
			"updated_on": "2024-01-02T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

	opts := &PRCreateOptions{
		Title:             "Updated Title",
		Description:       "Updated description",
		DestinationBranch: "develop",
	}

	result, err := client.UpdatePullRequest(context.Background(), "workspace", "repo", 700, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify HTTP method is PUT
	if receivedReq.Method != http.MethodPut {
		t.Errorf("expected PUT method, got %s", receivedReq.Method)
	}

	// Verify URL contains PR ID
	if !strings.Contains(receivedReq.URL.Path, "/700") {
		t.Errorf("expected URL path to contain /700, got %s", receivedReq.URL.Path)
	}

	// Verify body contains updated fields
	var body map[string]interface{}
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if body["title"] != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %v", body["title"])
	}

	if result.Title != "Updated Title" {
		t.Errorf("expected result title 'Updated Title', got %q", result.Title)
	}
}

func TestListPRComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/comments") {
			http.Error(w, "wrong endpoint", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 2,
			"page": 1,
			"pagelen": 10,
			"values": [
				{
					"id": 1,
					"content": {"raw": "First comment", "markup": "markdown"},
					"user": {"display_name": "Commenter 1"},
					"created_on": "2024-01-01T00:00:00Z",
					"updated_on": "2024-01-01T00:00:00Z"
				},
				{
					"id": 2,
					"content": {"raw": "Second comment"},
					"user": {"display_name": "Commenter 2"},
					"created_on": "2024-01-02T00:00:00Z",
					"updated_on": "2024-01-02T00:00:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	comments, err := client.ListPRComments(context.Background(), "workspace", "repo", 800)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(comments.Values) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments.Values))
	}

	if comments.Values[0].Content.Raw != "First comment" {
		t.Errorf("expected first comment 'First comment', got %q", comments.Values[0].Content.Raw)
	}
}

func TestAddPRComment(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": 100,
			"content": {"raw": "New comment", "markup": "markdown"},
			"user": {"display_name": "Me"},
			"created_on": "2024-01-01T00:00:00Z",
			"updated_on": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

	opts := &AddPRCommentOptions{
		Content: "New comment",
	}

	comment, err := client.AddPRComment(context.Background(), "workspace", "repo", 900, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify body structure
	var body map[string]interface{}
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	content, ok := body["content"].(map[string]interface{})
	if !ok {
		t.Error("expected content object in body")
	} else if content["raw"] != "New comment" {
		t.Errorf("expected raw content 'New comment', got %v", content["raw"])
	}

	if comment.ID != 100 {
		t.Errorf("expected comment ID 100, got %d", comment.ID)
	}
}

func TestGetPullRequestStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/statuses") {
			http.Error(w, "wrong endpoint", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 2,
			"page": 1,
			"pagelen": 10,
			"values": [
				{
					"uuid": "{status-1}",
					"key": "build",
					"name": "CI Build",
					"state": "SUCCESSFUL",
					"description": "Build passed",
					"created_on": "2024-01-01T00:00:00Z",
					"updated_on": "2024-01-01T00:00:00Z"
				},
				{
					"uuid": "{status-2}",
					"key": "tests",
					"name": "Test Suite",
					"state": "INPROGRESS",
					"created_on": "2024-01-01T00:00:00Z",
					"updated_on": "2024-01-01T00:00:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	statuses, err := client.GetPullRequestStatuses(context.Background(), "workspace", "repo", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statuses.Values) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses.Values))
	}

	if statuses.Values[0].State != "SUCCESSFUL" {
		t.Errorf("expected first status state 'SUCCESSFUL', got %q", statuses.Values[0].State)
	}

	if statuses.Values[1].State != "INPROGRESS" {
		t.Errorf("expected second status state 'INPROGRESS', got %q", statuses.Values[1].State)
	}
}
