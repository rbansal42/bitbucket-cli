package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// BranchFull represents a Bitbucket branch with full details
type BranchFull struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Target *BranchHead `json:"target"`
	Links  BranchLinks `json:"links"`
}

// BranchHead represents the target commit of a branch
type BranchHead struct {
	Hash    string `json:"hash"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Author  struct {
		Raw  string `json:"raw"`
		User *User  `json:"user,omitempty"`
	} `json:"author"`
	Date  string `json:"date"`
	Links struct {
		Self Link `json:"self"`
		HTML Link `json:"html"`
	} `json:"links"`
}

// BranchLinks contains links related to a branch
type BranchLinks struct {
	Self    Link `json:"self"`
	Commits Link `json:"commits"`
	HTML    Link `json:"html"`
}

// BranchListOptions are options for listing branches
type BranchListOptions struct {
	Sort  string // Sort field: name, -name, etc.
	Query string // Filter query (Bitbucket query language)
	Page  int    // Page number
	Limit int    // Number of items per page (pagelen)
}

// BranchCreateOptions are options for creating a branch
type BranchCreateOptions struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

// ListBranches lists branches for a repository
func (c *Client) ListBranches(ctx context.Context, workspace, repoSlug string, opts *BranchListOptions) (*Paginated[BranchFull], error) {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches", workspace, repoSlug)

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

	return ParseResponse[*Paginated[BranchFull]](resp)
}

// GetBranch retrieves a single branch by name
func (c *Client) GetBranch(ctx context.Context, workspace, repoSlug, branchName string) (*BranchFull, error) {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", workspace, repoSlug, url.PathEscape(branchName))

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*BranchFull](resp)
}

// CreateBranch creates a new branch
func (c *Client) CreateBranch(ctx context.Context, workspace, repoSlug string, opts *BranchCreateOptions) (*BranchFull, error) {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches", workspace, repoSlug)

	resp, err := c.Post(ctx, path, opts)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*BranchFull](resp)
}

// DeleteBranch deletes a branch by name
func (c *Client) DeleteBranch(ctx context.Context, workspace, repoSlug, branchName string) error {
	path := fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", workspace, repoSlug, url.PathEscape(branchName))

	_, err := c.Delete(ctx, path)
	return err
}
