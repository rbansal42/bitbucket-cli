package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the base URL for Bitbucket Cloud API
	DefaultBaseURL = "https://api.bitbucket.org/2.0"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second

	// UserAgent is the User-Agent header sent with requests
	UserAgent = "bb-cli/1.0"
)

// Client is the Bitbucket API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	username   string // For Basic Auth with API tokens
	apiToken   string // For Basic Auth with API tokens
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// NewClient creates a new Bitbucket API client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithToken sets the authentication token (Bearer token for OAuth/Access Tokens)
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.token = token
	}
}

// WithBasicAuth sets username and API token for Basic Auth
// Used with Atlassian API tokens (email:api_token)
func WithBasicAuth(username, apiToken string) ClientOption {
	return func(c *Client) {
		c.username = username
		c.apiToken = apiToken
	}
}

// WithBaseURL sets a custom base URL
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(baseURL, "/")
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// APIError represents an error returned by the Bitbucket API
type APIError struct {
	StatusCode int
	Message    string            `json:"message"`
	Detail     string            `json:"detail"`
	Fields     map[string]string `json:"fields,omitempty"`
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("API error %d: %s - %s", e.StatusCode, e.Message, e.Detail)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// Request represents an API request
type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Body    interface{}
	Headers map[string]string
}

// Response represents an API response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Do performs an API request
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	// Build URL
	reqURL, err := url.Parse(c.baseURL + "/" + strings.TrimPrefix(req.Path, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid request URL: %w", err)
	}

	if req.Query != nil {
		reqURL.RawQuery = req.Query.Encode()
	}

	// Build request body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("could not marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Accept", "application/json")

	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set authentication
	if c.username != "" && c.apiToken != "" {
		// Basic Auth for Atlassian API tokens
		httpReq.SetBasicAuth(c.username, c.apiToken)
	} else if c.token != "" {
		// Bearer token for OAuth or Access Tokens
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       respBody,
	}

	// Check for errors
	if httpResp.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    http.StatusText(httpResp.StatusCode),
		}

		// Try to parse error response
		var errResp struct {
			Error struct {
				Message string            `json:"message"`
				Detail  string            `json:"detail"`
				Fields  map[string]string `json:"fields"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			apiErr.Message = errResp.Error.Message
			apiErr.Detail = errResp.Error.Detail
			apiErr.Fields = errResp.Error.Fields
		}

		return resp, apiErr
	}

	return resp, nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*Response, error) {
	return c.Do(ctx, &Request{
		Method: http.MethodGet,
		Path:   path,
		Query:  query,
	})
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, &Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, &Request{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	})
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, &Request{
		Method: http.MethodDelete,
		Path:   path,
	})
}

// Paginated represents a paginated response from Bitbucket
type Paginated[T any] struct {
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	PageLen  int    `json:"pagelen"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Values   []T    `json:"values"`
}

// ParseResponse parses a JSON response into the given type
func ParseResponse[T any](resp *Response) (T, error) {
	var result T
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return result, fmt.Errorf("could not parse response: %w", err)
	}
	return result, nil
}

// User represents a Bitbucket user
type User struct {
	UUID        string `json:"uuid"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	AccountID   string `json:"account_id"`
	Links       struct {
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// GetCurrentUser returns the authenticated user
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	resp, err := c.Get(ctx, "/user", nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*User](resp)
}
