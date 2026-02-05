package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Content represents rendered content with raw and HTML formats
type Content struct {
	Raw    string `json:"raw,omitempty"`
	Markup string `json:"markup,omitempty"`
	HTML   string `json:"html,omitempty"`
}

// IssueLinks contains links related to an issue
type IssueLinks struct {
	Self        *Link `json:"self,omitempty"`
	HTML        *Link `json:"html,omitempty"`
	Comments    *Link `json:"comments,omitempty"`
	Attachments *Link `json:"attachments,omitempty"`
}

// Issue represents a Bitbucket issue
type Issue struct {
	Type       string      `json:"type"`
	ID         int         `json:"id"`
	Title      string      `json:"title"`
	Content    *Content    `json:"content,omitempty"`
	State      string      `json:"state"`    // new, open, resolved, on hold, invalid, duplicate, wontfix, closed
	Kind       string      `json:"kind"`     // bug, enhancement, proposal, task
	Priority   string      `json:"priority"` // trivial, minor, major, critical, blocker
	Reporter   *User       `json:"reporter,omitempty"`
	Assignee   *User       `json:"assignee,omitempty"`
	Repository *Repository `json:"repository,omitempty"`
	CreatedOn  time.Time   `json:"created_on"`
	UpdatedOn  time.Time   `json:"updated_on"`
	Votes      int         `json:"votes"`
	Links      *IssueLinks `json:"links,omitempty"`
}

// IssueCommentLinks contains links related to an issue comment
type IssueCommentLinks struct {
	Self *Link `json:"self,omitempty"`
	HTML *Link `json:"html,omitempty"`
}

// IssueComment represents a comment on a Bitbucket issue
type IssueComment struct {
	ID        int                `json:"id"`
	Content   *Content           `json:"content,omitempty"`
	User      *User              `json:"user,omitempty"`
	CreatedOn time.Time          `json:"created_on"`
	UpdatedOn time.Time          `json:"updated_on"`
	Links     *IssueCommentLinks `json:"links,omitempty"`
}

// IssueListOptions are options for listing issues
type IssueListOptions struct {
	State    string // Filter by state
	Kind     string // Filter by kind
	Priority string // Filter by priority
	Assignee string // Filter by assignee
	Q        string // Search query
	Sort     string // Sort field
	Page     int    // Page number
	Limit    int    // Number of items per page (pagelen)
}

// IssueCreateOptions are options for creating an issue
type IssueCreateOptions struct {
	Title    string   `json:"title"`
	Content  *Content `json:"content,omitempty"`
	Kind     string   `json:"kind,omitempty"`
	Priority string   `json:"priority,omitempty"`
	Assignee *User    `json:"assignee,omitempty"`
}

// IssueUpdateOptions are options for updating an issue
type IssueUpdateOptions struct {
	Title    *string  `json:"title,omitempty"`
	Content  *Content `json:"content,omitempty"`
	State    *string  `json:"state,omitempty"`
	Kind     *string  `json:"kind,omitempty"`
	Priority *string  `json:"priority,omitempty"`
	Assignee *User    `json:"assignee,omitempty"`
}

// issueCreateRequest is the actual API request body for creating an issue
type issueCreateRequest struct {
	Title    string `json:"title"`
	Content  *struct {
		Raw string `json:"raw,omitempty"`
	} `json:"content,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Priority string `json:"priority,omitempty"`
	Assignee *struct {
		UUID string `json:"uuid,omitempty"`
	} `json:"assignee,omitempty"`
}

// issueUpdateRequest is the actual API request body for updating an issue
type issueUpdateRequest struct {
	Title    string `json:"title,omitempty"`
	Content  *struct {
		Raw string `json:"raw,omitempty"`
	} `json:"content,omitempty"`
	State    string `json:"state,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Priority string `json:"priority,omitempty"`
	Assignee *struct {
		UUID string `json:"uuid,omitempty"`
	} `json:"assignee,omitempty"`
}

// issueCommentRequest is the API request body for creating an issue comment
type issueCommentRequest struct {
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
}

// ListIssues lists issues for a repository
func (c *Client) ListIssues(ctx context.Context, workspace, repoSlug string, opts *IssueListOptions) (*Paginated[Issue], error) {
	path := fmt.Sprintf("/repositories/%s/%s/issues", workspace, repoSlug)

	query := url.Values{}
	if opts != nil {
		// Build query filter using Bitbucket query language
		var q string
		if opts.Q != "" {
			q = opts.Q
		} else {
			var filters []string
			if opts.State != "" {
				filters = append(filters, fmt.Sprintf("state=\"%s\"", opts.State))
			}
			if opts.Kind != "" {
				filters = append(filters, fmt.Sprintf("kind=\"%s\"", opts.Kind))
			}
			if opts.Priority != "" {
				filters = append(filters, fmt.Sprintf("priority=\"%s\"", opts.Priority))
			}
			if opts.Assignee != "" {
				filters = append(filters, fmt.Sprintf("assignee.username=\"%s\"", opts.Assignee))
			}
			if len(filters) > 0 {
				for i, f := range filters {
					if i == 0 {
						q = f
					} else {
						q += " AND " + f
					}
				}
			}
		}
		if q != "" {
			query.Set("q", q)
		}

		if opts.Sort != "" {
			query.Set("sort", opts.Sort)
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

	return ParseResponse[*Paginated[Issue]](resp)
}

// GetIssue gets a single issue by ID
func (c *Client) GetIssue(ctx context.Context, workspace, repoSlug string, issueID int) (*Issue, error) {
	path := fmt.Sprintf("/repositories/%s/%s/issues/%d", workspace, repoSlug, issueID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Issue](resp)
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(ctx context.Context, workspace, repoSlug string, opts *IssueCreateOptions) (*Issue, error) {
	path := fmt.Sprintf("/repositories/%s/%s/issues", workspace, repoSlug)

	// Build request body
	reqBody := issueCreateRequest{
		Title:    opts.Title,
		Kind:     opts.Kind,
		Priority: opts.Priority,
	}

	if opts.Content != nil && opts.Content.Raw != "" {
		reqBody.Content = &struct {
			Raw string `json:"raw,omitempty"`
		}{Raw: opts.Content.Raw}
	}

	if opts.Assignee != nil && opts.Assignee.UUID != "" {
		reqBody.Assignee = &struct {
			UUID string `json:"uuid,omitempty"`
		}{UUID: opts.Assignee.UUID}
	}

	resp, err := c.Post(ctx, path, reqBody)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Issue](resp)
}

// UpdateIssue updates an existing issue
func (c *Client) UpdateIssue(ctx context.Context, workspace, repoSlug string, issueID int, opts *IssueUpdateOptions) (*Issue, error) {
	path := fmt.Sprintf("/repositories/%s/%s/issues/%d", workspace, repoSlug, issueID)

	// Build request body - only include non-nil fields
	body := make(map[string]interface{})

	if opts.Title != nil {
		body["title"] = *opts.Title
	}
	if opts.Content != nil && opts.Content.Raw != "" {
		body["content"] = map[string]string{"raw": opts.Content.Raw}
	}
	if opts.State != nil {
		body["state"] = *opts.State
	}
	if opts.Kind != nil {
		body["kind"] = *opts.Kind
	}
	if opts.Priority != nil {
		body["priority"] = *opts.Priority
	}
	if opts.Assignee != nil {
		body["assignee"] = map[string]string{"uuid": opts.Assignee.UUID}
	}

	resp, err := c.Put(ctx, path, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Issue](resp)
}

// DeleteIssue deletes an issue
func (c *Client) DeleteIssue(ctx context.Context, workspace, repoSlug string, issueID int) error {
	path := fmt.Sprintf("/repositories/%s/%s/issues/%d", workspace, repoSlug, issueID)

	_, err := c.Delete(ctx, path)
	return err
}

// ListIssueComments lists comments on an issue
func (c *Client) ListIssueComments(ctx context.Context, workspace, repoSlug string, issueID int) (*Paginated[IssueComment], error) {
	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/comments", workspace, repoSlug, issueID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[IssueComment]](resp)
}

// CreateIssueComment adds a comment to an issue
func (c *Client) CreateIssueComment(ctx context.Context, workspace, repoSlug string, issueID int, body string) (*IssueComment, error) {
	path := fmt.Sprintf("/repositories/%s/%s/issues/%d/comments", workspace, repoSlug, issueID)

	reqBody := issueCommentRequest{}
	reqBody.Content.Raw = body

	resp, err := c.Post(ctx, path, reqBody)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*IssueComment](resp)
}
