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

func TestListPipelines(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		opts          *PipelineListOptions
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
			expectedURL: "/repositories/myworkspace/myrepo/pipelines",
			response: `{
				"size": 2,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"uuid": "{pipeline-1}", "build_number": 1, "state": {"name": "COMPLETED"}},
					{"uuid": "{pipeline-2}", "build_number": 2, "state": {"name": "IN_PROGRESS"}}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  2,
		},
		{
			name:        "list with status filter",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &PipelineListOptions{Status: "SUCCESSFUL"},
			expectedURL: "/repositories/myworkspace/myrepo/pipelines",
			expectedQuery: map[string]string{
				"status": "SUCCESSFUL",
			},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"uuid": "{pipeline-1}", "build_number": 1, "state": {"name": "COMPLETED", "result": {"name": "SUCCESSFUL"}}}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with sort",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &PipelineListOptions{Sort: "-created_on"},
			expectedURL: "/repositories/myworkspace/myrepo/pipelines",
			expectedQuery: map[string]string{
				"sort": "-created_on",
			},
			response: `{
				"size": 1,
				"page": 1,
				"pagelen": 10,
				"values": [
					{"uuid": "{pipeline-1}", "build_number": 100}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  1,
		},
		{
			name:        "list with all filters",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &PipelineListOptions{Status: "FAILED", Sort: "created_on"},
			expectedURL: "/repositories/myworkspace/myrepo/pipelines",
			expectedQuery: map[string]string{
				"status": "FAILED",
				"sort":   "created_on",
			},
			response: `{
				"size": 5,
				"page": 1,
				"pagelen": 10,
				"values": []
			}`,
			statusCode: http.StatusOK,
			wantCount:  0,
		},
		{
			name:        "handles 401 unauthorized",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        nil,
			response:    `{"error": {"message": "Unauthorized", "detail": "Authentication required"}}`,
			statusCode:  http.StatusUnauthorized,
			wantErr:     true,
		},
		{
			name:        "handles 404 repository not found",
			workspace:   "myworkspace",
			repoSlug:    "nonexistent",
			opts:        nil,
			response:    `{"error": {"message": "Repository not found"}}`,
			statusCode:  http.StatusNotFound,
			wantErr:     true,
		},
		{
			name:        "handles 400 bad request",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        &PipelineListOptions{Status: "INVALID"},
			response:    `{"error": {"message": "Invalid status filter"}}`,
			statusCode:  http.StatusBadRequest,
			wantErr:     true,
		},
		{
			name:        "handles 500 internal server error",
			workspace:   "myworkspace",
			repoSlug:    "myrepo",
			opts:        nil,
			response:    `{"error": {"message": "Internal server error"}}`,
			statusCode:  http.StatusInternalServerError,
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

			result, err := client.ListPipelines(context.Background(), tt.workspace, tt.repoSlug, tt.opts)

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
				t.Errorf("expected %d pipelines, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetPipeline(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		repoSlug     string
		pipelineUUID string
		response     string
		statusCode   int
		wantErr      bool
		wantNumber   int
	}{
		{
			name:         "successfully get pipeline",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			response: `{
				"uuid": "{pipeline-uuid}",
				"build_number": 42,
				"state": {
					"type": "pipeline_state_completed",
					"name": "COMPLETED",
					"result": {
						"type": "pipeline_state_completed_successful",
						"name": "SUCCESSFUL"
					}
				},
				"target": {
					"type": "pipeline_ref_target",
					"ref_type": "branch",
					"ref_name": "main"
				},
				"trigger": {
					"type": "pipeline_trigger_push"
				},
				"created_on": "2024-01-15T10:30:00Z",
				"build_seconds_used": 120
			}`,
			statusCode: http.StatusOK,
			wantNumber: 42,
		},
		{
			name:         "pipeline not found",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{nonexistent}",
			response:     `{"error": {"message": "Pipeline not found"}}`,
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
		{
			name:         "unauthorized access",
			workspace:    "private-workspace",
			repoSlug:     "private-repo",
			pipelineUUID: "{pipeline-uuid}",
			response:     `{"error": {"message": "Unauthorized", "detail": "You do not have access to this repository"}}`,
			statusCode:   http.StatusUnauthorized,
			wantErr:      true,
		},
		{
			name:         "internal server error",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			response:     `{"error": {"message": "Internal server error"}}`,
			statusCode:   http.StatusInternalServerError,
			wantErr:      true,
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

			result, err := client.GetPipeline(context.Background(), tt.workspace, tt.repoSlug, tt.pipelineUUID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify URL contains workspace, repo, and pipeline UUID
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/pipelines/" + tt.pipelineUUID
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path to contain %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify HTTP method
			if receivedReq.Method != http.MethodGet {
				t.Errorf("expected GET method, got %s", receivedReq.Method)
			}

			// Verify response parsing
			if result.BuildNumber != tt.wantNumber {
				t.Errorf("expected build_number %d, got %d", tt.wantNumber, result.BuildNumber)
			}
		})
	}
}

func TestRunPipeline(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		repoSlug     string
		opts         *PipelineRunOptions
		expectedBody map[string]interface{}
		response     string
		statusCode   int
		wantErr      bool
		wantNumber   int
	}{
		{
			name:      "run pipeline with branch target",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &PipelineRunOptions{
				Target: &PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: "main",
				},
			},
			response: `{
				"uuid": "{new-pipeline-uuid}",
				"build_number": 100,
				"state": {
					"name": "PENDING"
				},
				"target": {
					"type": "pipeline_ref_target",
					"ref_type": "branch",
					"ref_name": "main"
				},
				"created_on": "2024-01-15T10:30:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantNumber: 100,
		},
		{
			name:      "run custom pipeline",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &PipelineRunOptions{
				Target: &PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: "main",
					Selector: &PipelineSelector{
						Type:    "custom",
						Pattern: "deploy-to-staging",
					},
				},
			},
			response: `{
				"uuid": "{custom-pipeline-uuid}",
				"build_number": 101,
				"state": {
					"name": "PENDING"
				},
				"target": {
					"type": "pipeline_ref_target",
					"ref_type": "branch",
					"ref_name": "main",
					"selector": {
						"type": "custom",
						"pattern": "deploy-to-staging"
					}
				},
				"created_on": "2024-01-15T10:30:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantNumber: 101,
		},
		{
			name:      "run pipeline with tag target",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &PipelineRunOptions{
				Target: &PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "tag",
					RefName: "v1.0.0",
				},
			},
			response: `{
				"uuid": "{tag-pipeline-uuid}",
				"build_number": 102,
				"state": {
					"name": "PENDING"
				},
				"target": {
					"type": "pipeline_ref_target",
					"ref_type": "tag",
					"ref_name": "v1.0.0"
				},
				"created_on": "2024-01-15T10:30:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantNumber: 102,
		},
		{
			name:      "run pipeline with commit target",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &PipelineRunOptions{
				Target: &PipelineTarget{
					Type: "pipeline_commit_target",
					Commit: &PipelineCommit{
						Type: "commit",
						Hash: "abc123def456",
					},
				},
			},
			response: `{
				"uuid": "{commit-pipeline-uuid}",
				"build_number": 103,
				"state": {
					"name": "PENDING"
				},
				"created_on": "2024-01-15T10:30:00Z"
			}`,
			statusCode: http.StatusCreated,
			wantNumber: 103,
		},
		{
			name:      "pipeline creation fails - pipelines disabled",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &PipelineRunOptions{
				Target: &PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: "main",
				},
			},
			response:   `{"error": {"message": "Pipelines are not enabled for this repository"}}`,
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:      "pipeline creation fails - branch not found",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			opts: &PipelineRunOptions{
				Target: &PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: "nonexistent-branch",
				},
			},
			response:   `{"error": {"message": "Branch not found"}}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			workspace:  "myworkspace",
			repoSlug:   "myrepo",
			opts:       &PipelineRunOptions{},
			response:   `{"error": {"message": "Unauthorized"}}`,
			statusCode: http.StatusUnauthorized,
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

			result, err := client.RunPipeline(context.Background(), tt.workspace, tt.repoSlug, tt.opts)

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
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/pipelines"
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

			// Verify target is present
			if _, exists := body["target"]; !exists {
				t.Error("expected target to be present in body")
			}

			// Verify result
			if result.BuildNumber != tt.wantNumber {
				t.Errorf("expected build_number %d, got %d", tt.wantNumber, result.BuildNumber)
			}
		})
	}
}

func TestStopPipeline(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		repoSlug     string
		pipelineUUID string
		statusCode   int
		response     string
		wantErr      bool
	}{
		{
			name:         "successful stop",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			statusCode:   http.StatusNoContent,
			response:     "",
			wantErr:      false,
		},
		{
			name:         "pipeline not found",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{nonexistent}",
			statusCode:   http.StatusNotFound,
			response:     `{"error": {"message": "Pipeline not found"}}`,
			wantErr:      true,
		},
		{
			name:         "pipeline already completed",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{completed-pipeline}",
			statusCode:   http.StatusBadRequest,
			response:     `{"error": {"message": "Pipeline is not running"}}`,
			wantErr:      true,
		},
		{
			name:         "unauthorized",
			workspace:    "other-workspace",
			repoSlug:     "other-repo",
			pipelineUUID: "{pipeline-uuid}",
			statusCode:   http.StatusUnauthorized,
			response:     `{"error": {"message": "Unauthorized"}}`,
			wantErr:      true,
		},
		{
			name:         "forbidden",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			statusCode:   http.StatusForbidden,
			response:     `{"error": {"message": "Forbidden", "detail": "You do not have permission to stop pipelines"}}`,
			wantErr:      true,
		},
		{
			name:         "internal server error",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			statusCode:   http.StatusInternalServerError,
			response:     `{"error": {"message": "Internal server error"}}`,
			wantErr:      true,
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

			err := client.StopPipeline(context.Background(), tt.workspace, tt.repoSlug, tt.pipelineUUID)

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
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/pipelines/" + tt.pipelineUUID + "/stopPipeline"
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}
		})
	}
}

func TestListPipelineSteps(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		repoSlug     string
		pipelineUUID string
		response     string
		statusCode   int
		wantErr      bool
		wantCount    int
	}{
		{
			name:         "list steps successfully",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			response: `{
				"size": 3,
				"page": 1,
				"pagelen": 10,
				"values": [
					{
						"uuid": "{step-1}",
						"name": "Build",
						"state": {"name": "COMPLETED", "result": {"name": "SUCCESSFUL"}},
						"image": {"name": "node:18"}
					},
					{
						"uuid": "{step-2}",
						"name": "Test",
						"state": {"name": "COMPLETED", "result": {"name": "SUCCESSFUL"}},
						"image": {"name": "node:18"}
					},
					{
						"uuid": "{step-3}",
						"name": "Deploy",
						"state": {"name": "IN_PROGRESS"},
						"image": {"name": "atlassian/default-image:4"}
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantCount:  3,
		},
		{
			name:         "list steps with empty pipeline",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{empty-pipeline}",
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
			name:         "pipeline not found",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{nonexistent}",
			response:     `{"error": {"message": "Pipeline not found"}}`,
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
		{
			name:         "unauthorized",
			workspace:    "other-workspace",
			repoSlug:     "other-repo",
			pipelineUUID: "{pipeline-uuid}",
			response:     `{"error": {"message": "Unauthorized"}}`,
			statusCode:   http.StatusUnauthorized,
			wantErr:      true,
		},
		{
			name:         "internal server error",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			response:     `{"error": {"message": "Internal server error"}}`,
			statusCode:   http.StatusInternalServerError,
			wantErr:      true,
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

			result, err := client.ListPipelineSteps(context.Background(), tt.workspace, tt.repoSlug, tt.pipelineUUID)

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

			// Verify URL
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/pipelines/" + tt.pipelineUUID + "/steps"
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify result
			if len(result.Values) != tt.wantCount {
				t.Errorf("expected %d steps, got %d", tt.wantCount, len(result.Values))
			}
		})
	}
}

func TestGetPipelineStepLog(t *testing.T) {
	tests := []struct {
		name         string
		workspace    string
		repoSlug     string
		pipelineUUID string
		stepUUID     string
		response     string
		contentType  string
		statusCode   int
		wantErr      bool
		wantLog      string
	}{
		{
			name:         "get step log successfully",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			stepUUID:     "{step-uuid}",
			response: `+ npm install
added 1234 packages in 45s

+ npm test
PASS src/tests/app.test.js
Tests: 42 passed, 42 total
Time: 5.234s`,
			contentType: "text/plain",
			statusCode:  http.StatusOK,
			wantLog: `+ npm install
added 1234 packages in 45s

+ npm test
PASS src/tests/app.test.js
Tests: 42 passed, 42 total
Time: 5.234s`,
		},
		{
			name:         "get empty log",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			stepUUID:     "{empty-step}",
			response:     "",
			contentType:  "text/plain",
			statusCode:   http.StatusOK,
			wantLog:      "",
		},
		{
			name:         "step not found",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			stepUUID:     "{nonexistent}",
			response:     `{"error": {"message": "Step not found"}}`,
			contentType:  "application/json",
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
		{
			name:         "pipeline not found",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{nonexistent}",
			stepUUID:     "{step-uuid}",
			response:     `{"error": {"message": "Pipeline not found"}}`,
			contentType:  "application/json",
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
		{
			name:         "unauthorized",
			workspace:    "other-workspace",
			repoSlug:     "other-repo",
			pipelineUUID: "{pipeline-uuid}",
			stepUUID:     "{step-uuid}",
			response:     `{"error": {"message": "Unauthorized"}}`,
			contentType:  "application/json",
			statusCode:   http.StatusUnauthorized,
			wantErr:      true,
		},
		{
			name:         "internal server error",
			workspace:    "myworkspace",
			repoSlug:     "myrepo",
			pipelineUUID: "{pipeline-uuid}",
			stepUUID:     "{step-uuid}",
			response:     `{"error": {"message": "Internal server error"}}`,
			contentType:  "application/json",
			statusCode:   http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedReq *http.Request

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedReq = r
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

			result, err := client.GetPipelineStepLog(context.Background(), tt.workspace, tt.repoSlug, tt.pipelineUUID, tt.stepUUID)

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

			// Verify URL
			expectedPath := "/repositories/" + tt.workspace + "/" + tt.repoSlug + "/pipelines/" + tt.pipelineUUID + "/steps/" + tt.stepUUID + "/log"
			if !strings.HasSuffix(receivedReq.URL.Path, expectedPath) {
				t.Errorf("expected URL path %q, got %s", expectedPath, receivedReq.URL.Path)
			}

			// Verify Accept header for text/plain
			acceptHeader := receivedReq.Header.Get("Accept")
			if acceptHeader != "text/plain" {
				t.Errorf("expected Accept header 'text/plain', got %q", acceptHeader)
			}

			// Verify result
			if result != tt.wantLog {
				t.Errorf("expected log %q, got %q", tt.wantLog, result)
			}
		})
	}
}

func TestPipelineParsing(t *testing.T) {
	// Test comprehensive pipeline response parsing with all fields
	responseJSON := `{
		"type": "pipeline",
		"uuid": "{complete-pipeline-uuid}",
		"build_number": 42,
		"creator": {
			"uuid": "{user-uuid}",
			"username": "testuser",
			"display_name": "Test User",
			"account_id": "user123"
		},
		"repository": {
			"uuid": "{repo-uuid}",
			"name": "myrepo",
			"full_name": "myworkspace/myrepo"
		},
		"target": {
			"type": "pipeline_ref_target",
			"ref_type": "branch",
			"ref_name": "main",
			"commit": {
				"type": "commit",
				"hash": "abc123def456"
			},
			"selector": {
				"type": "custom",
				"pattern": "deploy"
			}
		},
		"trigger": {
			"type": "pipeline_trigger_manual"
		},
		"state": {
			"type": "pipeline_state_completed",
			"name": "COMPLETED",
			"result": {
				"type": "pipeline_state_completed_successful",
				"name": "SUCCESSFUL"
			}
		},
		"created_on": "2024-01-15T10:30:00Z",
		"completed_on": "2024-01-15T10:35:00Z",
		"build_seconds_used": 300,
		"links": {
			"self": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/pipelines/{uuid}"},
			"steps": {"href": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/pipelines/{uuid}/steps"}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	pipeline, err := client.GetPipeline(context.Background(), "myworkspace", "myrepo", "{complete-pipeline-uuid}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are parsed correctly
	if pipeline.UUID != "{complete-pipeline-uuid}" {
		t.Errorf("expected UUID '{complete-pipeline-uuid}', got %q", pipeline.UUID)
	}

	if pipeline.BuildNumber != 42 {
		t.Errorf("expected build_number 42, got %d", pipeline.BuildNumber)
	}

	if pipeline.Type != "pipeline" {
		t.Errorf("expected type 'pipeline', got %q", pipeline.Type)
	}

	// Verify creator
	if pipeline.Creator == nil {
		t.Fatal("expected Creator to not be nil")
	}
	if pipeline.Creator.DisplayName != "Test User" {
		t.Errorf("expected creator display_name 'Test User', got %q", pipeline.Creator.DisplayName)
	}

	// Verify repository
	if pipeline.Repository == nil {
		t.Fatal("expected Repository to not be nil")
	}
	if pipeline.Repository.Name != "myrepo" {
		t.Errorf("expected repository name 'myrepo', got %q", pipeline.Repository.Name)
	}

	// Verify target
	if pipeline.Target == nil {
		t.Fatal("expected Target to not be nil")
	}
	if pipeline.Target.RefType != "branch" {
		t.Errorf("expected target ref_type 'branch', got %q", pipeline.Target.RefType)
	}
	if pipeline.Target.RefName != "main" {
		t.Errorf("expected target ref_name 'main', got %q", pipeline.Target.RefName)
	}
	if pipeline.Target.Commit == nil {
		t.Fatal("expected Target.Commit to not be nil")
	}
	if pipeline.Target.Commit.Hash != "abc123def456" {
		t.Errorf("expected commit hash 'abc123def456', got %q", pipeline.Target.Commit.Hash)
	}
	if pipeline.Target.Selector == nil {
		t.Fatal("expected Target.Selector to not be nil")
	}
	if pipeline.Target.Selector.Type != "custom" {
		t.Errorf("expected selector type 'custom', got %q", pipeline.Target.Selector.Type)
	}
	if pipeline.Target.Selector.Pattern != "deploy" {
		t.Errorf("expected selector pattern 'deploy', got %q", pipeline.Target.Selector.Pattern)
	}

	// Verify trigger
	if pipeline.Trigger == nil {
		t.Fatal("expected Trigger to not be nil")
	}
	if pipeline.Trigger.Type != "pipeline_trigger_manual" {
		t.Errorf("expected trigger type 'pipeline_trigger_manual', got %q", pipeline.Trigger.Type)
	}

	// Verify state
	if pipeline.State == nil {
		t.Fatal("expected State to not be nil")
	}
	if pipeline.State.Name != "COMPLETED" {
		t.Errorf("expected state name 'COMPLETED', got %q", pipeline.State.Name)
	}
	if pipeline.State.Result == nil {
		t.Fatal("expected State.Result to not be nil")
	}
	if pipeline.State.Result.Name != "SUCCESSFUL" {
		t.Errorf("expected state result name 'SUCCESSFUL', got %q", pipeline.State.Result.Name)
	}

	// Verify build_seconds_used
	if pipeline.BuildSecondsUsed != 300 {
		t.Errorf("expected build_seconds_used 300, got %d", pipeline.BuildSecondsUsed)
	}

	// Verify completed_on
	if pipeline.CompletedOn == nil {
		t.Fatal("expected CompletedOn to not be nil")
	}

	// Verify links
	if pipeline.Links == nil {
		t.Fatal("expected Links to not be nil")
	}
	if pipeline.Links.Self == nil {
		t.Fatal("expected Links.Self to not be nil")
	}
	if pipeline.Links.Steps == nil {
		t.Fatal("expected Links.Steps to not be nil")
	}
}

func TestPipelineStepParsing(t *testing.T) {
	// Test pipeline step response parsing
	responseJSON := `{
		"size": 1,
		"page": 1,
		"pagelen": 10,
		"values": [{
			"type": "pipeline_step",
			"uuid": "{step-uuid}",
			"name": "Build and Test",
			"started_on": "2024-01-15T10:30:00Z",
			"completed_on": "2024-01-15T10:32:00Z",
			"state": {
				"type": "pipeline_step_state_completed",
				"name": "COMPLETED",
				"result": {
					"type": "pipeline_step_state_completed_successful",
					"name": "SUCCESSFUL"
				}
			},
			"image": {
				"name": "node:18-alpine"
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

	result, err := client.ListPipelineSteps(context.Background(), "myworkspace", "myrepo", "{pipeline-uuid}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Values) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Values))
	}

	step := result.Values[0]

	// Verify all fields
	if step.UUID != "{step-uuid}" {
		t.Errorf("expected UUID '{step-uuid}', got %q", step.UUID)
	}

	if step.Name != "Build and Test" {
		t.Errorf("expected name 'Build and Test', got %q", step.Name)
	}

	if step.Type != "pipeline_step" {
		t.Errorf("expected type 'pipeline_step', got %q", step.Type)
	}

	// Verify state
	if step.State == nil {
		t.Fatal("expected State to not be nil")
	}
	if step.State.Name != "COMPLETED" {
		t.Errorf("expected state name 'COMPLETED', got %q", step.State.Name)
	}
	if step.State.Result == nil {
		t.Fatal("expected State.Result to not be nil")
	}
	if step.State.Result.Name != "SUCCESSFUL" {
		t.Errorf("expected state result name 'SUCCESSFUL', got %q", step.State.Result.Name)
	}

	// Verify image
	if step.Image == nil {
		t.Fatal("expected Image to not be nil")
	}
	if step.Image.Name != "node:18-alpine" {
		t.Errorf("expected image name 'node:18-alpine', got %q", step.Image.Name)
	}

	// Verify timestamps
	if step.StartedOn == nil {
		t.Fatal("expected StartedOn to not be nil")
	}
	if step.CompletedOn == nil {
		t.Fatal("expected CompletedOn to not be nil")
	}
}

func TestPipelineErrorHandling(t *testing.T) {
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
			response:       `{"error": {"message": "Pipeline not found"}}`,
			wantStatusCode: http.StatusNotFound,
			wantMessage:    "Pipeline not found",
		},
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			response:       `{"error": {"message": "Invalid pipeline configuration"}}`,
			wantStatusCode: http.StatusBadRequest,
			wantMessage:    "Invalid pipeline configuration",
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

			_, err := client.GetPipeline(context.Background(), "workspace", "repo", "{pipeline-uuid}")

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

func TestListPipelinesPagination(t *testing.T) {
	// Test that pagination response is properly parsed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"size": 100,
			"page": 2,
			"pagelen": 10,
			"next": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/pipelines?page=3",
			"previous": "https://api.bitbucket.org/2.0/repositories/myworkspace/myrepo/pipelines?page=1",
			"values": [
				{"uuid": "{pipeline-1}", "build_number": 11},
				{"uuid": "{pipeline-2}", "build_number": 12}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	result, err := client.ListPipelines(context.Background(), "myworkspace", "myrepo", nil)
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

func TestRunPipelineRequestBody(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"uuid": "{uuid}",
			"build_number": 1,
			"state": {"name": "PENDING"},
			"created_on": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))

	// Run with custom pipeline selector
	opts := &PipelineRunOptions{
		Target: &PipelineTarget{
			Type:    "pipeline_ref_target",
			RefType: "branch",
			RefName: "feature-branch",
			Selector: &PipelineSelector{
				Type:    "custom",
				Pattern: "deploy-to-prod",
			},
		},
	}

	_, err := client.RunPipeline(context.Background(), "workspace", "repo", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request body structure
	var body map[string]interface{}
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}

	// Verify target
	target, ok := body["target"].(map[string]interface{})
	if !ok {
		t.Fatal("expected target object in body")
	}

	if target["type"] != "pipeline_ref_target" {
		t.Errorf("expected target type 'pipeline_ref_target', got %v", target["type"])
	}

	if target["ref_type"] != "branch" {
		t.Errorf("expected target ref_type 'branch', got %v", target["ref_type"])
	}

	if target["ref_name"] != "feature-branch" {
		t.Errorf("expected target ref_name 'feature-branch', got %v", target["ref_name"])
	}

	// Verify selector
	selector, ok := target["selector"].(map[string]interface{})
	if !ok {
		t.Fatal("expected selector object in target")
	}

	if selector["type"] != "custom" {
		t.Errorf("expected selector type 'custom', got %v", selector["type"])
	}

	if selector["pattern"] != "deploy-to-prod" {
		t.Errorf("expected selector pattern 'deploy-to-prod', got %v", selector["pattern"])
	}
}
