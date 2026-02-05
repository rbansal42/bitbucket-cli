package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// SnippetFile represents a file in a snippet
type SnippetFile struct {
	Links struct {
		Self Link `json:"self"`
		HTML Link `json:"html"`
	} `json:"links"`
}

// Snippet represents a Bitbucket snippet
type Snippet struct {
	Type      string                 `json:"type"`
	ID        int                    `json:"id"`
	Title     string                 `json:"title"`
	Scm       string                 `json:"scm"`
	CreatedOn string                 `json:"created_on"`
	UpdatedOn string                 `json:"updated_on"`
	Owner     *User                  `json:"owner"`
	Creator   *User                  `json:"creator"`
	IsPrivate bool                   `json:"is_private"`
	Files     map[string]SnippetFile `json:"files"`
	Links     SnippetLinks           `json:"links"`
}

// SnippetLinks contains links for a snippet
type SnippetLinks struct {
	Self     Link `json:"self"`
	HTML     Link `json:"html"`
	Comments Link `json:"comments"`
	Watchers Link `json:"watchers"`
	Commits  Link `json:"commits"`
}

// SnippetListOptions for listing snippets
type SnippetListOptions struct {
	Role  string // owner, contributor, member
	Page  int
	Limit int
}

// ListSnippets lists snippets for a workspace
func (c *Client) ListSnippets(ctx context.Context, workspace string, opts *SnippetListOptions) (*Paginated[Snippet], error) {
	path := fmt.Sprintf("/snippets/%s", workspace)

	query := url.Values{}
	if opts != nil {
		if opts.Role != "" {
			query.Set("role", opts.Role)
		}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Limit > 0 {
			query.Set("pagelen", strconv.Itoa(opts.Limit))
		}
	}

	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[Snippet]](resp)
}

// GetSnippet retrieves a single snippet by encoded ID
func (c *Client) GetSnippet(ctx context.Context, workspace, encodedID string) (*Snippet, error) {
	path := fmt.Sprintf("/snippets/%s/%s", workspace, url.PathEscape(encodedID))

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Snippet](resp)
}

// CreateSnippet creates a new snippet with files
func (c *Client) CreateSnippet(ctx context.Context, workspace string, title string, isPrivate bool, files map[string]string) (*Snippet, error) {
	path := fmt.Sprintf("/snippets/%s", workspace)

	body, contentType, err := buildSnippetMultipartBody(title, isPrivate, files)
	if err != nil {
		return nil, fmt.Errorf("could not build multipart body: %w", err)
	}

	resp, err := c.doMultipart(ctx, http.MethodPost, path, body, contentType)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Snippet](resp)
}

// UpdateSnippet updates an existing snippet
func (c *Client) UpdateSnippet(ctx context.Context, workspace, encodedID string, title string, files map[string]string) (*Snippet, error) {
	path := fmt.Sprintf("/snippets/%s/%s", workspace, url.PathEscape(encodedID))

	body, contentType, err := buildSnippetMultipartBody(title, false, files)
	if err != nil {
		return nil, fmt.Errorf("could not build multipart body: %w", err)
	}

	resp, err := c.doMultipart(ctx, http.MethodPut, path, body, contentType)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Snippet](resp)
}

// DeleteSnippet deletes a snippet by encoded ID
func (c *Client) DeleteSnippet(ctx context.Context, workspace, encodedID string) error {
	path := fmt.Sprintf("/snippets/%s/%s", workspace, url.PathEscape(encodedID))

	_, err := c.Delete(ctx, path)
	return err
}

// GetSnippetFileContent retrieves the content of a file in a snippet
func (c *Client) GetSnippetFileContent(ctx context.Context, workspace, encodedID, filePath string) ([]byte, error) {
	path := fmt.Sprintf("/snippets/%s/%s/files/%s", workspace, url.PathEscape(encodedID), url.PathEscape(filePath))

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// buildSnippetMultipartBody creates a multipart form body for snippet create/update
func buildSnippetMultipartBody(title string, isPrivate bool, files map[string]string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add title if provided
	if title != "" {
		if err := writer.WriteField("title", title); err != nil {
			return nil, "", err
		}
	}

	// Add is_private field for create operations
	if isPrivate {
		if err := writer.WriteField("is_private", "true"); err != nil {
			return nil, "", err
		}
	}

	// Add files
	for filename, content := range files {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, "", err
		}
		if _, err := io.Copy(part, strings.NewReader(content)); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}

// doMultipart performs a multipart/form-data request
func (c *Client) doMultipart(ctx context.Context, method, path string, body *bytes.Buffer, contentType string) (*Response, error) {
	// Build URL
	reqURL, err := url.Parse(c.baseURL + "/" + strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid request URL: %w", err)
	}

	// Create HTTP request with the raw body
	httpReq, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("User-Agent", UserAgent)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", contentType)

	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
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
