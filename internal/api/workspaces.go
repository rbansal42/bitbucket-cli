package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// WorkspaceFull represents a Bitbucket workspace with full details
type WorkspaceFull struct {
	UUID      string         `json:"uuid"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	Type      string         `json:"type"`
	IsPrivate bool           `json:"is_private"`
	CreatedOn time.Time      `json:"created_on"`
	Links     WorkspaceLinks `json:"links"`
}

// WorkspaceLinks contains links related to a workspace
type WorkspaceLinks struct {
	Avatar       Link `json:"avatar"`
	HTML         Link `json:"html"`
	Members      Link `json:"members"`
	Owners       Link `json:"owners"`
	Projects     Link `json:"projects"`
	Repositories Link `json:"repositories"`
	Self         Link `json:"self"`
}

// WorkspaceMembership represents a user's membership in a workspace (from user/permissions/workspaces)
type WorkspaceMembership struct {
	Permission string         `json:"permission"`
	User       *User          `json:"user,omitempty"`
	Workspace  *WorkspaceFull `json:"workspace"`
	AddedOn    time.Time      `json:"added_on,omitempty"`
	Links      struct {
		Self Link `json:"self"`
	} `json:"links"`
}

// WorkspaceMember represents a member of a workspace (from workspaces/{workspace}/permissions)
type WorkspaceMember struct {
	Permission string         `json:"permission"`
	User       *User          `json:"user"`
	Workspace  *WorkspaceFull `json:"workspace,omitempty"`
	AddedOn    time.Time      `json:"added_on,omitempty"`
	Links      struct {
		Self Link `json:"self"`
	} `json:"links"`
}

// WorkspaceListOptions are options for listing workspaces
type WorkspaceListOptions struct {
	Role  string // Filter by role: owner, collaborator, member
	Sort  string // Sort field
	Query string // Filter query
	Page  int    // Page number
	Limit int    // Number of items per page (pagelen)
}

// WorkspaceMemberListOptions are options for listing workspace members
type WorkspaceMemberListOptions struct {
	Query string // Filter query
	Page  int    // Page number
	Limit int    // Number of items per page (pagelen)
}

// ListWorkspaces lists workspaces the authenticated user is a member of
func (c *Client) ListWorkspaces(ctx context.Context, opts *WorkspaceListOptions) (*Paginated[WorkspaceMembership], error) {
	path := "/user/permissions/workspaces"

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

	return ParseResponse[*Paginated[WorkspaceMembership]](resp)
}

// GetWorkspace retrieves a single workspace by slug
func (c *Client) GetWorkspace(ctx context.Context, workspaceSlug string) (*WorkspaceFull, error) {
	path := fmt.Sprintf("/workspaces/%s", workspaceSlug)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*WorkspaceFull](resp)
}

// ListWorkspaceMembers lists members of a workspace
func (c *Client) ListWorkspaceMembers(ctx context.Context, workspaceSlug string, opts *WorkspaceMemberListOptions) (*Paginated[WorkspaceMember], error) {
	path := fmt.Sprintf("/workspaces/%s/permissions", workspaceSlug)

	query := url.Values{}
	if opts != nil {
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

	return ParseResponse[*Paginated[WorkspaceMember]](resp)
}
