# Phase 2: Pull Request Commands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement full pull request workflow commands for bb CLI - create, list, view, checkout, merge, review, and more.

**Architecture:** PR commands use the shared API client. Repository context is detected from git remotes. Output supports both human-readable tables and JSON.

**Tech Stack:** Go 1.25+, Cobra, internal/api client, internal/git for context detection

---

## Task 1: Add PR API Types and Client Methods

**Files:**
- Create: `internal/api/pullrequests.go`

**Implementation:**

```go
package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"` // OPEN, MERGED, DECLINED, SUPERSEDED
	Author      User      `json:"author"`
	Source      PRRef     `json:"source"`
	Destination PRRef     `json:"destination"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	CloseSource bool      `json:"close_source_branch"`
	MergeCommit *Commit   `json:"merge_commit,omitempty"`
	Links       PRLinks   `json:"links"`
	Reviewers   []User    `json:"reviewers"`
	Participants []Participant `json:"participants"`
	CommentCount int      `json:"comment_count"`
	TaskCount    int      `json:"task_count"`
}

type PRRef struct {
	Branch     Branch     `json:"branch"`
	Commit     Commit     `json:"commit"`
	Repository Repository `json:"repository"`
}

type Branch struct {
	Name string `json:"name"`
}

type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message,omitempty"`
}

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	UUID     string `json:"uuid"`
}

type PRLinks struct {
	HTML     Link `json:"html"`
	Diff     Link `json:"diff"`
	Commits  Link `json:"commits"`
	Comments Link `json:"comments"`
	Approve  Link `json:"approve"`
	Merge    Link `json:"merge"`
}

type Link struct {
	Href string `json:"href"`
}

type Participant struct {
	User     User   `json:"user"`
	Role     string `json:"role"` // PARTICIPANT, REVIEWER
	Approved bool   `json:"approved"`
	State    string `json:"state,omitempty"` // approved, changes_requested
}

// PRListOptions for filtering pull requests
type PRListOptions struct {
	State  string // OPEN, MERGED, DECLINED, SUPERSEDED
	Author string
	Q      string // query string
	Sort   string
	Page   int
	Limit  int
}

// ListPullRequests lists pull requests for a repository
func (c *Client) ListPullRequests(ctx context.Context, workspace, repoSlug string, opts *PRListOptions) (*Paginated[PullRequest], error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repoSlug)
	
	query := url.Values{}
	if opts != nil {
		if opts.State != "" {
			query.Set("state", opts.State)
		}
		if opts.Q != "" {
			query.Set("q", opts.Q)
		}
		if opts.Sort != "" {
			query.Set("sort", opts.Sort)
		}
		if opts.Limit > 0 {
			query.Set("pagelen", strconv.Itoa(opts.Limit))
		}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
	}

	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[PullRequest]](resp)
}

// GetPullRequest gets a single pull request by ID
func (c *Client) GetPullRequest(ctx context.Context, workspace, repoSlug string, prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", workspace, repoSlug, prID)
	
	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// PRCreateOptions for creating a pull request
type PRCreateOptions struct {
	Title             string   `json:"title"`
	Description       string   `json:"description,omitempty"`
	SourceBranch      string   `json:"-"`
	DestinationBranch string   `json:"-"`
	CloseSourceBranch bool     `json:"close_source_branch"`
	Reviewers         []string `json:"-"` // UUIDs
}

// CreatePullRequest creates a new pull request
func (c *Client) CreatePullRequest(ctx context.Context, workspace, repoSlug string, opts *PRCreateOptions) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repoSlug)

	body := map[string]interface{}{
		"title":               opts.Title,
		"close_source_branch": opts.CloseSourceBranch,
		"source": map[string]interface{}{
			"branch": map[string]string{"name": opts.SourceBranch},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{"name": opts.DestinationBranch},
		},
	}

	if opts.Description != "" {
		body["description"] = opts.Description
	}

	if len(opts.Reviewers) > 0 {
		reviewers := make([]map[string]string, len(opts.Reviewers))
		for i, uuid := range opts.Reviewers {
			reviewers[i] = map[string]string{"uuid": uuid}
		}
		body["reviewers"] = reviewers
	}

	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// MergePullRequest merges a pull request
func (c *Client) MergePullRequest(ctx context.Context, workspace, repoSlug string, prID int, message string, closeSource bool, mergeStrategy string) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge", workspace, repoSlug, prID)

	body := map[string]interface{}{
		"close_source_branch": closeSource,
	}
	if message != "" {
		body["message"] = message
	}
	if mergeStrategy != "" {
		body["merge_strategy"] = mergeStrategy // merge_commit, squash, fast_forward
	}

	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// DeclinePullRequest declines (closes) a pull request
func (c *Client) DeclinePullRequest(ctx context.Context, workspace, repoSlug string, prID int) (*PullRequest, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline", workspace, repoSlug, prID)

	resp, err := c.Post(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PullRequest](resp)
}

// ApprovePullRequest approves a pull request
func (c *Client) ApprovePullRequest(ctx context.Context, workspace, repoSlug string, prID int) error {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve", workspace, repoSlug, prID)

	_, err := c.Post(ctx, path, nil)
	return err
}

// UnapprovePullRequest removes approval from a pull request
func (c *Client) UnapprovePullRequest(ctx context.Context, workspace, repoSlug string, prID int) error {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve", workspace, repoSlug, prID)

	_, err := c.Delete(ctx, path)
	return err
}

// RequestChanges requests changes on a pull request
func (c *Client) RequestChanges(ctx context.Context, workspace, repoSlug string, prID int) error {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/request-changes", workspace, repoSlug, prID)

	_, err := c.Post(ctx, path, nil)
	return err
}

// GetPullRequestDiff gets the diff for a pull request
func (c *Client) GetPullRequestDiff(ctx context.Context, workspace, repoSlug string, prID int) (string, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diff", workspace, repoSlug, prID)

	resp, err := c.Do(ctx, &Request{
		Method: "GET",
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

// PRComment represents a comment on a pull request
type PRComment struct {
	ID        int       `json:"id"`
	Content   Content   `json:"content"`
	User      User      `json:"user"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Inline    *Inline   `json:"inline,omitempty"`
	Parent    *Parent   `json:"parent,omitempty"`
}

type Content struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

type Inline struct {
	Path string `json:"path"`
	From int    `json:"from,omitempty"`
	To   int    `json:"to"`
}

type Parent struct {
	ID int `json:"id"`
}

// ListPRComments lists comments on a pull request
func (c *Client) ListPRComments(ctx context.Context, workspace, repoSlug string, prID int) (*Paginated[PRComment], error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, prID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[PRComment]](resp)
}

// AddPRComment adds a comment to a pull request
func (c *Client) AddPRComment(ctx context.Context, workspace, repoSlug string, prID int, content string) (*PRComment, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", workspace, repoSlug, prID)

	body := map[string]interface{}{
		"content": map[string]string{"raw": content},
	}

	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*PRComment](resp)
}
```

**Verify:** `go build ./...`

**Commit:**
```bash
git add internal/api/pullrequests.go
git commit -m "feat(api): add pull request types and client methods"
```

---

## Task 2: Implement bb pr list Command

**Files:**
- Create: `internal/cmd/pr/pr.go`
- Create: `internal/cmd/pr/list.go`
- Modify: `internal/cmd/root.go`

**Implementation for pr.go:**

```go
package pr

import (
	"github.com/spf13/cobra"
	"github.com/rbansal42/bb/internal/iostreams"
)

func NewCmdPR(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr <command>",
		Short: "Manage pull requests",
		Long:  "Work with Bitbucket pull requests.",
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdView(streams))
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdCheckout(streams))
	cmd.AddCommand(NewCmdMerge(streams))
	cmd.AddCommand(NewCmdClose(streams))
	cmd.AddCommand(NewCmdReview(streams))
	cmd.AddCommand(NewCmdDiff(streams))
	cmd.AddCommand(NewCmdComment(streams))

	return cmd
}
```

**Implementation for list.go:** (full command with table output, filtering, JSON support)

**Wire up in root.go:**
```go
import "github.com/rbansal42/bb/internal/cmd/pr"
// In init():
rootCmd.AddCommand(pr.NewCmdPR(GetStreams()))
```

**Commit:**
```bash
git add internal/cmd/pr/pr.go internal/cmd/pr/list.go internal/cmd/root.go
git commit -m "feat(pr): add bb pr list command"
```

---

## Task 3: Implement bb pr view Command

**Files:**
- Create: `internal/cmd/pr/view.go`

**Features:**
- View PR details (title, description, author, reviewers, status)
- Show approval status
- Show comment count
- `--web` flag to open in browser
- `--json` flag for JSON output

**Commit:**
```bash
git add internal/cmd/pr/view.go
git commit -m "feat(pr): add bb pr view command"
```

---

## Task 4: Implement bb pr create Command

**Files:**
- Create: `internal/cmd/pr/create.go`

**Features:**
- Create PR from current branch
- `--title`, `--body`, `--base` flags
- `--reviewer` flag (multiple)
- `--draft` flag
- `--fill` flag to use commit messages
- Interactive mode if title not provided
- `--web` flag to open in browser after creation

**Commit:**
```bash
git add internal/cmd/pr/create.go
git commit -m "feat(pr): add bb pr create command"
```

---

## Task 5: Implement bb pr checkout Command

**Files:**
- Create: `internal/cmd/pr/checkout.go`

**Features:**
- Checkout PR branch locally
- Create local branch tracking PR
- Handle detached HEAD for closed PRs

**Commit:**
```bash
git add internal/cmd/pr/checkout.go
git commit -m "feat(pr): add bb pr checkout command"
```

---

## Task 6: Implement bb pr merge Command

**Files:**
- Create: `internal/cmd/pr/merge.go`

**Features:**
- Merge PR with various strategies (merge_commit, squash, fast_forward)
- `--delete-branch` flag
- `--squash` and `--merge` flags
- Custom commit message

**Commit:**
```bash
git add internal/cmd/pr/merge.go
git commit -m "feat(pr): add bb pr merge command"
```

---

## Task 7: Implement bb pr close and bb pr review Commands

**Files:**
- Create: `internal/cmd/pr/close.go`
- Create: `internal/cmd/pr/review.go`

**close features:**
- Decline a PR
- Optional comment when closing

**review features:**
- `--approve` flag
- `--request-changes` flag
- `--comment` flag with body

**Commit:**
```bash
git add internal/cmd/pr/close.go internal/cmd/pr/review.go
git commit -m "feat(pr): add bb pr close and review commands"
```

---

## Task 8: Implement bb pr diff and bb pr comment Commands

**Files:**
- Create: `internal/cmd/pr/diff.go`
- Create: `internal/cmd/pr/comment.go`

**diff features:**
- Show PR diff
- Support color output
- Pipe-friendly

**comment features:**
- Add comment to PR
- Reply to existing comment with `--reply-to`

**Commit:**
```bash
git add internal/cmd/pr/diff.go internal/cmd/pr/comment.go
git commit -m "feat(pr): add bb pr diff and comment commands"
```

---

## Task 9: Add Tests for PR Commands

**Files:**
- Create: `internal/api/pullrequests_test.go`
- Create: `internal/cmd/pr/list_test.go`

**Test coverage:**
- API client methods with mock HTTP server
- PR list command output formatting
- Flag parsing and validation

**Commit:**
```bash
git add internal/api/pullrequests_test.go internal/cmd/pr/list_test.go
git commit -m "test: add tests for PR API and commands"
```

---

## Summary

After completing all tasks:
- Full PR workflow: list, view, create, checkout, merge, close, review, diff, comment
- API types for Bitbucket PR endpoints
- Unit tests for API and commands
- Ready for Phase 3 (Repository commands)
