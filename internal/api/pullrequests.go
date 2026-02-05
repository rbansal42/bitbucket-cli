package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PRState represents the state of a pull request
type PRState string

const (
	PRStateOpen     PRState = "OPEN"
	PRStateMerged   PRState = "MERGED"
	PRStateDeclined PRState = "DECLINED"
)

// MergeStrategy represents the merge strategy for a pull request
type MergeStrategy string

const (
	MergeStrategyMergeCommit MergeStrategy = "merge_commit"
	MergeStrategySquash      MergeStrategy = "squash"
	MergeStrategyFastForward MergeStrategy = "fast_forward"
)

// Link represents a hyperlink
type Link struct {
	Href string `json:"href"`
}

// PRLinks contains links related to a pull request
type PRLinks struct {
	Self     Link `json:"self"`
	HTML     Link `json:"html"`
	Commits  Link `json:"commits"`
	Approve  Link `json:"approve"`
	Diff     Link `json:"diff"`
	DiffStat Link `json:"diffstat"`
	Comments Link `json:"comments"`
	Activity Link `json:"activity"`
	Merge    Link `json:"merge"`
	Decline  Link `json:"decline"`
}

// Commit represents a git commit
type Commit struct {
	Hash  string `json:"hash"`
	Links struct {
		Self Link `json:"self"`
		HTML Link `json:"html"`
	} `json:"links"`
}

// Branch represents a git branch
type Branch struct {
	Name string `json:"name"`
}

// Repository represents a Bitbucket repository
type Repository struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Slug      string `json:"slug"`
	Links     struct {
		Self   Link `json:"self"`
		HTML   Link `json:"html"`
		Avatar Link `json:"avatar"`
	} `json:"links"`
}

// PRRef represents a source or destination reference
type PRRef struct {
	Branch     Branch      `json:"branch"`
	Commit     Commit      `json:"commit"`
	Repository *Repository `json:"repository,omitempty"`
}

// Participant represents a pull request participant
type Participant struct {
	User             User   `json:"user"`
	Role             string `json:"role"` // PARTICIPANT, REVIEWER
	Approved         bool   `json:"approved"`
	State            string `json:"state,omitempty"`           // approved, changes_requested, null
	ParticipatedOn   string `json:"participated_on,omitempty"` // ISO 8601 timestamp
}

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	ID                int64        `json:"id"`
	Title             string       `json:"title"`
	Description       string       `json:"description"`
	State             PRState      `json:"state"`
	Author            User         `json:"author"`
	Source            PRRef        `json:"source"`
	Destination       PRRef        `json:"destination"`
	MergeCommit       *Commit      `json:"merge_commit,omitempty"`
	CloseSourceBranch bool         `json:"close_source_branch"`
	ClosedBy          *User        `json:"closed_by,omitempty"`
	Reason            string       `json:"reason,omitempty"`
	CreatedOn         time.Time    `json:"created_on"`
	UpdatedOn         time.Time    `json:"updated_on"`
	Links             PRLinks      `json:"links"`
	Participants      []Participant `json:"participants,omitempty"`
	Reviewers         []User       `json:"reviewers,omitempty"`
	CommentCount      int          `json:"comment_count"`
	TaskCount         int          `json:"task_count"`
}

// PRComment represents a comment on a pull request
type PRComment struct {
	ID        int64     `json:"id"`
	Content   struct {
		Raw    string `json:"raw"`
		Markup string `json:"markup"`
		HTML   string `json:"html"`
	} `json:"content"`
	User      User      `json:"user"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Inline    *struct {
		From int    `json:"from,omitempty"`
		To   int    `json:"to,omitempty"`
		Path string `json:"path"`
	} `json:"inline,omitempty"`
	Parent *struct {
		ID int64 `json:"id"`
	} `json:"parent,omitempty"`
	Links struct {
		Self Link `json:"self"`
		HTML Link `json:"html"`
	} `json:"links"`
}

// PRListOptions are options for listing pull requests
type PRListOptions struct {
	State  PRState // Filter by state (OPEN, MERGED, DECLINED)
	Author string  // Filter by author username
	Page   int     // Page number
	Limit  int     // Number of items per page (pagelen)
}

// PRCreateOptions are options for creating a pull request
type PRCreateOptions struct {
	Title             string   `json:"title"`
	Description       string   `json:"description,omitempty"`
	SourceBranch      string   `json:"-"` // Used to build source object
	SourceRepo        string   `json:"-"` // Optional: for cross-repo PRs
	DestinationBranch string   `json:"-"` // Used to build destination object
	CloseSourceBranch bool     `json:"close_source_branch"`
	Reviewers         []string `json:"-"` // List of user UUIDs
}

// prCreateRequest is the actual API request body for creating a PR
type prCreateRequest struct {
	Title             string `json:"title"`
	Description       string `json:"description,omitempty"`
	Source            struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Repository *struct {
			FullName string `json:"full_name"`
		} `json:"repository,omitempty"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	CloseSourceBranch bool `json:"close_source_branch"`
	Reviewers         []struct {
		UUID string `json:"uuid"`
	} `json:"reviewers,omitempty"`
}

// PRMergeOptions are options for merging a pull request
type PRMergeOptions struct {
	Message           string        `json:"message,omitempty"`
	CloseSourceBranch bool          `json:"close_source_branch"`
	MergeStrategy     MergeStrategy `json:"merge_strategy,omitempty"`
}

// ListPullRequests lists pull requests for a repository
func (c *Client) ListPullRequests(ctx context.Context, workspace, repoSlug string, opts *PRListOptions) (*Paginated[PullRequest], error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repoSlug)

	query := url.Values{}
	if opts != nil {
		if opts.State != "" {
			query.Set("state", string(opts.State))
		}
		if opts.Author != "" {
			// Use q parameter for author filtering
			query.Set("q", fmt.Sprintf("author.username=\"%s\"", opts.Author))
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

	return ParseResponse[*Paginated[PullRequest]](resp)
}

// GetPullRequest retrieves a single pull request
func (c *Client) GetPullRequest(ctx context.Context, workspace, repoSlug string, prID int64) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", workspace, repoSlug, prID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// CreatePullRequest creates a new pull request
func (c *Client) CreatePullRequest(ctx context.Context, workspace, repoSlug string, opts *PRCreateOptions) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repoSlug)

	// Build request body
	reqBody := prCreateRequest{
		Title:             opts.Title,
		Description:       opts.Description,
		CloseSourceBranch: opts.CloseSourceBranch,
	}
	reqBody.Source.Branch.Name = opts.SourceBranch
	reqBody.Destination.Branch.Name = opts.DestinationBranch

	if opts.SourceRepo != "" {
		reqBody.Source.Repository = &struct {
			FullName string `json:"full_name"`
		}{FullName: opts.SourceRepo}
	}

	if len(opts.Reviewers) > 0 {
		for _, uuid := range opts.Reviewers {
			reqBody.Reviewers = append(reqBody.Reviewers, struct {
				UUID string `json:"uuid"`
			}{UUID: uuid})
		}
	}

	resp, err := c.Post(ctx, path, reqBody)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// MergePullRequest merges a pull request
func (c *Client) MergePullRequest(ctx context.Context, workspace, repoSlug string, prID int64, opts *PRMergeOptions) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge", workspace, repoSlug, prID)

	var body interface{}
	if opts != nil {
		body = opts
	}

	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// DeclinePullRequest declines a pull request
func (c *Client) DeclinePullRequest(ctx context.Context, workspace, repoSlug string, prID int64) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline", workspace, repoSlug, prID)

	resp, err := c.Post(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// ApprovePullRequest approves a pull request
func (c *Client) ApprovePullRequest(ctx context.Context, workspace, repoSlug string, prID int64) (*Participant, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve", workspace, repoSlug, prID)

	resp, err := c.Post(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Participant](resp)
}

// UnapprovePullRequest removes approval from a pull request
func (c *Client) UnapprovePullRequest(ctx context.Context, workspace, repoSlug string, prID int64) error {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve", workspace, repoSlug, prID)

	_, err := c.Delete(ctx, path)
	return err
}

// RequestChanges requests changes on a pull request
func (c *Client) RequestChanges(ctx context.Context, workspace, repoSlug string, prID int64) (*Participant, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/request-changes", workspace, repoSlug, prID)

	resp, err := c.Post(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Participant](resp)
}

// GetPullRequestDiff retrieves the diff of a pull request
func (c *Client) GetPullRequestDiff(ctx context.Context, workspace, repoSlug string, prID int64) (string, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diff", workspace, repoSlug, prID)

	resp, err := c.Do(ctx, &Request{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Accept": "text/plain",
		},
	})
	if err != nil {
		return "", err
	}

	return string(resp.Body), nil
}

// ListPRComments lists comments on a pull request
func (c *Client) ListPRComments(ctx context.Context, workspace, repoSlug string, prID int64) (*Paginated[PRComment], error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, prID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[PRComment]](resp)
}

// AddPRCommentOptions are options for adding a comment to a pull request
type AddPRCommentOptions struct {
	Content string `json:"-"`      // The comment text
	ParentID int64 `json:"-"`      // Optional: ID of parent comment for replies
	Path     string `json:"-"`     // Optional: file path for inline comments
	Line     int    `json:"-"`     // Optional: line number for inline comments
}

// addPRCommentRequest is the actual API request body for adding a comment
type addPRCommentRequest struct {
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	Parent *struct {
		ID int64 `json:"id"`
	} `json:"parent,omitempty"`
	Inline *struct {
		To   int    `json:"to"`
		Path string `json:"path"`
	} `json:"inline,omitempty"`
}

// AddPRComment adds a comment to a pull request
func (c *Client) AddPRComment(ctx context.Context, workspace, repoSlug string, prID int64, opts *AddPRCommentOptions) (*PRComment, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, prID)

	reqBody := addPRCommentRequest{}
	reqBody.Content.Raw = opts.Content

	if opts.ParentID > 0 {
		reqBody.Parent = &struct {
			ID int64 `json:"id"`
		}{ID: opts.ParentID}
	}

	if opts.Path != "" {
		reqBody.Inline = &struct {
			To   int    `json:"to"`
			Path string `json:"path"`
		}{To: opts.Line, Path: opts.Path}
	}

	resp, err := c.Post(ctx, path, reqBody)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PRComment](resp)
}

// UpdatePullRequest updates an existing pull request
func (c *Client) UpdatePullRequest(ctx context.Context, workspace, repoSlug string, prID int64, opts *PRCreateOptions) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", workspace, repoSlug, prID)

	// Build update body - only include fields that should be updated
	body := map[string]interface{}{}
	if opts.Title != "" {
		body["title"] = opts.Title
	}
	if opts.Description != "" {
		body["description"] = opts.Description
	}
	if opts.DestinationBranch != "" {
		body["destination"] = map[string]interface{}{
			"branch": map[string]string{
				"name": opts.DestinationBranch,
			},
		}
	}
	body["close_source_branch"] = opts.CloseSourceBranch

	if len(opts.Reviewers) > 0 {
		reviewers := make([]map[string]string, len(opts.Reviewers))
		for i, uuid := range opts.Reviewers {
			reviewers[i] = map[string]string{"uuid": uuid}
		}
		body["reviewers"] = reviewers
	}

	resp, err := c.Put(ctx, path, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// GetPullRequestStatuses retrieves build statuses for a pull request
func (c *Client) GetPullRequestStatuses(ctx context.Context, workspace, repoSlug string, prID int64) (*Paginated[CommitStatus], error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/statuses", workspace, repoSlug, prID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[CommitStatus]](resp)
}

// CommitStatus represents a build status for a commit
type CommitStatus struct {
	UUID        string    `json:"uuid"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	State       string    `json:"state"` // SUCCESSFUL, FAILED, INPROGRESS, STOPPED
	URL         string    `json:"url"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	Links       struct {
		Self   Link `json:"self"`
		Commit Link `json:"commit"`
	} `json:"links"`
}

// PullRequestJSON is a type alias for JSON marshaling with custom format
type PullRequestJSON struct {
	*PullRequest
}

// MarshalJSON provides a simplified JSON representation
func (pr PullRequestJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":                  pr.ID,
		"title":               pr.Title,
		"description":         pr.Description,
		"state":               pr.State,
		"author":              pr.Author.DisplayName,
		"source_branch":       pr.Source.Branch.Name,
		"destination_branch":  pr.Destination.Branch.Name,
		"created_on":          pr.CreatedOn,
		"updated_on":          pr.UpdatedOn,
		"url":                 pr.Links.HTML.Href,
		"close_source_branch": pr.CloseSourceBranch,
		"comment_count":       pr.CommentCount,
		"task_count":          pr.TaskCount,
	})
}
