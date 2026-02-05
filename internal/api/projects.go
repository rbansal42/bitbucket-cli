package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ProjectFull represents a Bitbucket project with full details
type ProjectFull struct {
	UUID        string        `json:"uuid"`
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	IsPrivate   bool          `json:"is_private"`
	CreatedOn   time.Time     `json:"created_on"`
	UpdatedOn   time.Time     `json:"updated_on"`
	Owner       *User         `json:"owner,omitempty"`
	Workspace   *Workspace    `json:"workspace,omitempty"`
	Links       ProjectLinks  `json:"links"`
}

// ProjectLinks contains links related to a project
type ProjectLinks struct {
	Self   Link `json:"self"`
	HTML   Link `json:"html"`
	Avatar Link `json:"avatar"`
}

// ProjectListOptions are options for listing projects
type ProjectListOptions struct {
	Sort  string // Sort field: name, -updated_on, etc.
	Query string // Filter query (Bitbucket query language)
	Page  int    // Page number
	Limit int    // Number of items per page (pagelen)
}

// ProjectCreateOptions are options for creating or updating a project
type ProjectCreateOptions struct {
	Key         string `json:"key,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
}

// ListProjects lists projects in a workspace
func (c *Client) ListProjects(ctx context.Context, workspaceSlug string, opts *ProjectListOptions) (*Paginated[ProjectFull], error) {
	path := fmt.Sprintf("/workspaces/%s/projects", workspaceSlug)

	query := url.Values{}
	if opts != nil {
		if opts.Sort != "" {
			query.Set("sort", opts.Sort)
		}
		if opts.Query != "" {
			query.Set("q", opts.Query)
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

	return ParseResponse[*Paginated[ProjectFull]](resp)
}

// GetProject retrieves a single project by key
func (c *Client) GetProject(ctx context.Context, workspaceSlug, projectKey string) (*ProjectFull, error) {
	path := fmt.Sprintf("/workspaces/%s/projects/%s", workspaceSlug, projectKey)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*ProjectFull](resp)
}

// CreateProject creates a new project in a workspace
func (c *Client) CreateProject(ctx context.Context, workspaceSlug string, opts *ProjectCreateOptions) (*ProjectFull, error) {
	path := fmt.Sprintf("/workspaces/%s/projects", workspaceSlug)

	resp, err := c.Post(ctx, path, opts)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*ProjectFull](resp)
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(ctx context.Context, workspaceSlug, projectKey string, opts *ProjectCreateOptions) (*ProjectFull, error) {
	path := fmt.Sprintf("/workspaces/%s/projects/%s", workspaceSlug, projectKey)

	resp, err := c.Put(ctx, path, opts)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*ProjectFull](resp)
}

// DeleteProject deletes a project
func (c *Client) DeleteProject(ctx context.Context, workspaceSlug, projectKey string) error {
	path := fmt.Sprintf("/workspaces/%s/projects/%s", workspaceSlug, projectKey)

	_, err := c.Delete(ctx, path)
	return err
}
