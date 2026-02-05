package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Repository represents a Bitbucket repository with full details
type RepositoryFull struct {
	UUID        string            `json:"uuid"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	FullName    string            `json:"full_name"`
	Description string            `json:"description"`
	IsPrivate   bool              `json:"is_private"`
	ForkPolicy  string            `json:"fork_policy"` // allow_forks, no_public_forks, no_forks
	Language    string            `json:"language"`
	Size        int64             `json:"size"`
	CreatedOn   time.Time         `json:"created_on"`
	UpdatedOn   time.Time         `json:"updated_on"`
	Owner       *User             `json:"owner"`
	Project     *Project          `json:"project,omitempty"`
	Workspace   *Workspace        `json:"workspace"`
	MainBranch  *MainBranch       `json:"mainbranch,omitempty"`
	Parent      *ParentRepository `json:"parent,omitempty"`
	Links       RepositoryLinks   `json:"links"`
}

// ParentRepository represents the parent of a forked repository
type ParentRepository struct {
	UUID      string     `json:"uuid"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	FullName  string     `json:"full_name"`
	Workspace *Workspace `json:"workspace"`
}

// RepositoryLinks contains links related to a repository
type RepositoryLinks struct {
	Self   Link        `json:"self"`
	HTML   Link        `json:"html"`
	Clone  []CloneLink `json:"clone"`
	Avatar Link        `json:"avatar"`
}

// CloneLink represents a clone URL for a repository
type CloneLink struct {
	Href string `json:"href"`
	Name string `json:"name"` // ssh, https
}

// Project represents a Bitbucket project
type Project struct {
	UUID string `json:"uuid"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Workspace represents a Bitbucket workspace
type Workspace struct {
	UUID string `json:"uuid"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// MainBranch represents the main branch of a repository
type MainBranch struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// RepositoryListOptions are options for listing repositories
type RepositoryListOptions struct {
	Role  string // Filter by role: owner, admin, contributor, member
	Sort  string // Sort field: name, -updated_on, etc.
	Query string // Filter query (Bitbucket query language)
	Page  int    // Page number
	Limit int    // Number of items per page (pagelen)
}

// RepositoryCreateOptions are options for creating a repository
type RepositoryCreateOptions struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	IsPrivate   bool     `json:"is_private"`
	ForkPolicy  string   `json:"fork_policy,omitempty"` // allow_forks, no_public_forks, no_forks
	Language    string   `json:"language,omitempty"`
	Project     *Project `json:"project,omitempty"`
	MainBranch  string   `json:"-"` // Used internally, not sent directly
	HasIssues   bool     `json:"has_issues,omitempty"`
	HasWiki     bool     `json:"has_wiki,omitempty"`
}

// repositoryCreateRequest is the actual API request body for creating a repository
type repositoryCreateRequest struct {
	Scm         string `json:"scm"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
	ForkPolicy  string `json:"fork_policy,omitempty"`
	Language    string `json:"language,omitempty"`
	Project     *struct {
		Key string `json:"key"`
	} `json:"project,omitempty"`
	HasIssues bool `json:"has_issues,omitempty"`
	HasWiki   bool `json:"has_wiki,omitempty"`
}

// forkRepositoryRequest is the API request body for forking a repository
type forkRepositoryRequest struct {
	Name      string `json:"name,omitempty"`
	Workspace *struct {
		Slug string `json:"slug"`
	} `json:"workspace,omitempty"`
}

// ListRepositories lists repositories in a workspace
func (c *Client) ListRepositories(ctx context.Context, workspace string, opts *RepositoryListOptions) (*Paginated[RepositoryFull], error) {
	path := fmt.Sprintf("/repositories/%s", workspace)

	query := url.Values{}
	if opts != nil {
		if opts.Role != "" {
			query.Set("role", opts.Role)
		}
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

	return ParseResponse[*Paginated[RepositoryFull]](resp)
}

// GetRepository retrieves a single repository
func (c *Client) GetRepository(ctx context.Context, workspace, repoSlug string) (*RepositoryFull, error) {
	path := fmt.Sprintf("/repositories/%s/%s", workspace, repoSlug)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*RepositoryFull](resp)
}

// CreateRepository creates a new repository in a workspace
func (c *Client) CreateRepository(ctx context.Context, workspace string, opts *RepositoryCreateOptions) (*RepositoryFull, error) {
	path := fmt.Sprintf("/repositories/%s/%s", workspace, opts.Name)

	// Build request body
	reqBody := repositoryCreateRequest{
		Scm:         "git",
		Name:        opts.Name,
		Description: opts.Description,
		IsPrivate:   opts.IsPrivate,
		ForkPolicy:  opts.ForkPolicy,
		Language:    opts.Language,
		HasIssues:   opts.HasIssues,
		HasWiki:     opts.HasWiki,
	}

	if opts.Project != nil && opts.Project.Key != "" {
		reqBody.Project = &struct {
			Key string `json:"key"`
		}{Key: opts.Project.Key}
	}

	resp, err := c.Post(ctx, path, reqBody)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*RepositoryFull](resp)
}

// DeleteRepository deletes a repository
func (c *Client) DeleteRepository(ctx context.Context, workspace, repoSlug string) error {
	path := fmt.Sprintf("/repositories/%s/%s", workspace, repoSlug)

	_, err := c.Delete(ctx, path)
	return err
}

// ForkRepository creates a fork of a repository
func (c *Client) ForkRepository(ctx context.Context, workspace, repoSlug string, destWorkspace, name string) (*RepositoryFull, error) {
	path := fmt.Sprintf("/repositories/%s/%s/forks", workspace, repoSlug)

	reqBody := forkRepositoryRequest{}
	if name != "" {
		reqBody.Name = name
	}
	if destWorkspace != "" {
		reqBody.Workspace = &struct {
			Slug string `json:"slug"`
		}{Slug: destWorkspace}
	}

	resp, err := c.Post(ctx, path, reqBody)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*RepositoryFull](resp)
}
