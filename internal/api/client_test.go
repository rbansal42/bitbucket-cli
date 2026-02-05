package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_UsesDefaultBaseURL(t *testing.T) {
	client := NewClient()

	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL to be %q, got %q", DefaultBaseURL, client.baseURL)
	}
}

func TestWithToken_SetsToken(t *testing.T) {
	token := "test-token-123"
	client := NewClient(WithToken(token))

	if client.token != token {
		t.Errorf("expected token to be %q, got %q", token, client.token)
	}
}

func TestWithBaseURL_SetsCustomURL(t *testing.T) {
	customURL := "https://custom.bitbucket.org/api"
	client := NewClient(WithBaseURL(customURL))

	if client.baseURL != customURL {
		t.Errorf("expected baseURL to be %q, got %q", customURL, client.baseURL)
	}
}

func TestWithBaseURL_TrimsTrailingSlash(t *testing.T) {
	customURL := "https://custom.bitbucket.org/api/"
	expectedURL := "https://custom.bitbucket.org/api"
	client := NewClient(WithBaseURL(customURL))

	if client.baseURL != expectedURL {
		t.Errorf("expected baseURL to be %q (without trailing slash), got %q", expectedURL, client.baseURL)
	}
}

func TestClientDo_SendsCorrectHeaders(t *testing.T) {
	token := "my-auth-token"

	var receivedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithToken(token),
	)

	_, err := client.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check Authorization header
	expectedAuth := "Bearer " + token
	if got := receivedReq.Header.Get("Authorization"); got != expectedAuth {
		t.Errorf("expected Authorization header %q, got %q", expectedAuth, got)
	}

	// Check User-Agent header
	if got := receivedReq.Header.Get("User-Agent"); got != UserAgent {
		t.Errorf("expected User-Agent header %q, got %q", UserAgent, got)
	}

	// Check Accept header
	expectedAccept := "application/json"
	if got := receivedReq.Header.Get("Accept"); got != expectedAccept {
		t.Errorf("expected Accept header %q, got %q", expectedAccept, got)
	}
}

func TestClientDo_SendsContentTypeForBody(t *testing.T) {
	var receivedReq *http.Request
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	body := map[string]string{"key": "value"}
	_, err := client.Do(context.Background(), &Request{
		Method: http.MethodPost,
		Path:   "/test",
		Body:   body,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check Content-Type header
	expectedContentType := "application/json"
	if got := receivedReq.Header.Get("Content-Type"); got != expectedContentType {
		t.Errorf("expected Content-Type header %q, got %q", expectedContentType, got)
	}

	// Verify body was sent correctly
	var parsedBody map[string]string
	if err := json.Unmarshal(receivedBody, &parsedBody); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if parsedBody["key"] != "value" {
		t.Errorf("expected body key to be 'value', got %q", parsedBody["key"])
	}
}

func TestClientDo_HandlesSuccessfulJSONResponse(t *testing.T) {
	expectedResponse := map[string]interface{}{
		"id":   "123",
		"name": "test-repo",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	resp, err := client.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/repos/123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("expected id to be '123', got %v", result["id"])
	}
	if result["name"] != "test-repo" {
		t.Errorf("expected name to be 'test-repo', got %v", result["name"])
	}
}

func TestClientDo_HandlesErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"message": "Repository not found", "detail": "The requested repository does not exist"}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	resp, err := client.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/repos/nonexistent",
	})

	// Response should still be returned
	if resp == nil {
		t.Fatal("expected response to be returned even on error")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	// Error should be APIError
	if err == nil {
		t.Fatal("expected error to be returned")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected APIError StatusCode to be %d, got %d", http.StatusNotFound, apiErr.StatusCode)
	}
	if apiErr.Message != "Repository not found" {
		t.Errorf("expected APIError Message to be 'Repository not found', got %q", apiErr.Message)
	}
	if apiErr.Detail != "The requested repository does not exist" {
		t.Errorf("expected APIError Detail to be 'The requested repository does not exist', got %q", apiErr.Detail)
	}
}

func TestClientDo_HandlesErrorResponseWithFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "Validation error", "fields": {"name": "Name is required"}}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Do(context.Background(), &Request{
		Method: http.MethodPost,
		Path:   "/repos",
		Body:   map[string]string{},
	})

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.Fields == nil {
		t.Fatal("expected APIError Fields to be set")
	}
	if apiErr.Fields["name"] != "Name is required" {
		t.Errorf("expected field error for 'name' to be 'Name is required', got %q", apiErr.Fields["name"])
	}
}

func TestClientDo_HandlesNonJSONErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/test",
	})

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected error to be *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected APIError StatusCode to be %d, got %d", http.StatusInternalServerError, apiErr.StatusCode)
	}
	// Should fall back to http.StatusText
	if apiErr.Message != "Internal Server Error" {
		t.Errorf("expected APIError Message to be 'Internal Server Error', got %q", apiErr.Message)
	}
}

func TestParseResponse_UnmarshalsJSON(t *testing.T) {
	type TestData struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	resp := &Response{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id": "abc123", "name": "Test Name"}`),
	}

	result, err := ParseResponse[TestData](resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "abc123" {
		t.Errorf("expected ID to be 'abc123', got %q", result.ID)
	}
	if result.Name != "Test Name" {
		t.Errorf("expected Name to be 'Test Name', got %q", result.Name)
	}
}

func TestParseResponse_HandlesPointerType(t *testing.T) {
	type TestData struct {
		Value int `json:"value"`
	}

	resp := &Response{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"value": 42}`),
	}

	result, err := ParseResponse[*TestData](resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to not be nil")
	}
	if result.Value != 42 {
		t.Errorf("expected Value to be 42, got %d", result.Value)
	}
}

func TestParseResponse_ReturnsErrorOnInvalidJSON(t *testing.T) {
	type TestData struct {
		ID string `json:"id"`
	}

	resp := &Response{
		StatusCode: http.StatusOK,
		Body:       []byte(`invalid json`),
	}

	_, err := ParseResponse[TestData](resp)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAPIError_ErrorString(t *testing.T) {
	tests := []struct {
		name     string
		apiErr   *APIError
		expected string
	}{
		{
			name: "with detail",
			apiErr: &APIError{
				StatusCode: 404,
				Message:    "Not Found",
				Detail:     "Repository does not exist",
			},
			expected: "API error 404: Not Found - Repository does not exist",
		},
		{
			name: "without detail",
			apiErr: &APIError{
				StatusCode: 500,
				Message:    "Internal Server Error",
			},
			expected: "API error 500: Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apiErr.Error(); got != tt.expected {
				t.Errorf("expected error string %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestClientDo_SendsQueryParameters(t *testing.T) {
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	query := make(map[string][]string)
	query["page"] = []string{"1"}
	query["pagelen"] = []string{"10"}

	_, err := client.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/repos",
		Query:  query,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.URL.Query().Get("page") != "1" {
		t.Errorf("expected page query param to be '1', got %q", receivedReq.URL.Query().Get("page"))
	}
	if receivedReq.URL.Query().Get("pagelen") != "10" {
		t.Errorf("expected pagelen query param to be '10', got %q", receivedReq.URL.Query().Get("pagelen"))
	}
}

func TestClientDo_NoAuthorizationHeaderWithoutToken(t *testing.T) {
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Authorization header should not be set
	if got := receivedReq.Header.Get("Authorization"); got != "" {
		t.Errorf("expected no Authorization header, got %q", got)
	}
}

func TestClientDo_CustomHeaders(t *testing.T) {
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	_, err := client.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/test",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := receivedReq.Header.Get("X-Custom-Header"); got != "custom-value" {
		t.Errorf("expected X-Custom-Header to be 'custom-value', got %q", got)
	}
}

func TestClientGet_ConvenienceMethod(t *testing.T) {
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	resp, err := client.Get(context.Background(), "/test-path", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.Method != http.MethodGet {
		t.Errorf("expected method GET, got %s", receivedReq.Method)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestClientPost_ConvenienceMethod(t *testing.T) {
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "new"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	body := map[string]string{"name": "test"}
	resp, err := client.Post(context.Background(), "/items", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.Method != http.MethodPost {
		t.Errorf("expected method POST, got %s", receivedReq.Method)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status code %d, got %d", http.StatusCreated, resp.StatusCode)
	}
}

func TestClientPut_ConvenienceMethod(t *testing.T) {
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"updated": true}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	body := map[string]string{"name": "updated"}
	resp, err := client.Put(context.Background(), "/items/123", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.Method != http.MethodPut {
		t.Errorf("expected method PUT, got %s", receivedReq.Method)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestClientDelete_ConvenienceMethod(t *testing.T) {
	var receivedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	resp, err := client.Delete(context.Background(), "/items/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.Method != http.MethodDelete {
		t.Errorf("expected method DELETE, got %s", receivedReq.Method)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status code %d, got %d", http.StatusNoContent, resp.StatusCode)
	}
}
