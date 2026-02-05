package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Pipeline represents a Bitbucket pipeline run
type Pipeline struct {
	Type             string           `json:"type"`
	UUID             string           `json:"uuid"`
	BuildNumber      int              `json:"build_number"`
	Creator          *User            `json:"creator,omitempty"`
	Repository       *Repository      `json:"repository,omitempty"`
	Target           *PipelineTarget  `json:"target,omitempty"`
	Trigger          *PipelineTrigger `json:"trigger,omitempty"`
	State            *PipelineState   `json:"state,omitempty"`
	CreatedOn        time.Time        `json:"created_on"`
	CompletedOn      *time.Time       `json:"completed_on,omitempty"`
	BuildSecondsUsed int              `json:"build_seconds_used"`
	Links            *PipelineLinks   `json:"links,omitempty"`
}

// PipelineTarget represents the target of a pipeline run (branch, tag, etc.)
type PipelineTarget struct {
	Type     string            `json:"type"`
	RefType  string            `json:"ref_type,omitempty"` // branch, tag
	RefName  string            `json:"ref_name,omitempty"`
	Commit   *PipelineCommit   `json:"commit,omitempty"`
	Selector *PipelineSelector `json:"selector,omitempty"`
}

// PipelineCommit represents a commit in a pipeline
type PipelineCommit struct {
	Type string `json:"type"`
	Hash string `json:"hash"`
}

// PipelineSelector represents a pipeline selector for targeting specific pipelines
type PipelineSelector struct {
	Type    string `json:"type"`    // branches, tags, pull-requests, custom
	Pattern string `json:"pattern"`
}

// PipelineTrigger represents what triggered a pipeline run
type PipelineTrigger struct {
	Type string `json:"type"` // push, pull_request, manual, schedule
}

// PipelineState represents the current state of a pipeline
type PipelineState struct {
	Type   string               `json:"type"`
	Name   string               `json:"name"` // PENDING, IN_PROGRESS, COMPLETED, etc.
	Result *PipelineStateResult `json:"result,omitempty"`
}

// PipelineStateResult represents the result of a completed pipeline
type PipelineStateResult struct {
	Type string `json:"type"`
	Name string `json:"name"` // SUCCESSFUL, FAILED, STOPPED, EXPIRED
}

// PipelineLinks contains links related to a pipeline
type PipelineLinks struct {
	Self  *Link `json:"self,omitempty"`
	Steps *Link `json:"steps,omitempty"`
}

// PipelineStep represents a step in a pipeline
type PipelineStep struct {
	Type        string             `json:"type"`
	UUID        string             `json:"uuid"`
	Name        string             `json:"name,omitempty"`
	StartedOn   *time.Time         `json:"started_on,omitempty"`
	CompletedOn *time.Time         `json:"completed_on,omitempty"`
	State       *PipelineStepState `json:"state,omitempty"`
	Image       *PipelineImage     `json:"image,omitempty"`
}

// PipelineStepState represents the state of a pipeline step
type PipelineStepState struct {
	Type   string               `json:"type"`
	Name   string               `json:"name"`
	Result *PipelineStateResult `json:"result,omitempty"`
}

// PipelineImage represents the Docker image used in a pipeline step
type PipelineImage struct {
	Name string `json:"name"`
}

// PipelineListOptions are options for listing pipelines
type PipelineListOptions struct {
	Status string // Filter by status
	Sort   string // Sort field
}

// PipelineRunOptions are options for triggering a new pipeline run
type PipelineRunOptions struct {
	Target *PipelineTarget `json:"target"`
}

// ListPipelines lists pipelines for a repository
func (c *Client) ListPipelines(ctx context.Context, workspace, repoSlug string, opts *PipelineListOptions) (*Paginated[Pipeline], error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines", workspace, repoSlug)

	query := url.Values{}
	if opts != nil {
		if opts.Status != "" {
			query.Set("status", opts.Status)
		}
		if opts.Sort != "" {
			query.Set("sort", opts.Sort)
		}
	}

	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[Pipeline]](resp)
}

// GetPipeline gets a single pipeline by UUID
func (c *Client) GetPipeline(ctx context.Context, workspace, repoSlug, pipelineUUID string) (*Pipeline, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s", workspace, repoSlug, pipelineUUID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Pipeline](resp)
}

// RunPipeline triggers a new pipeline run
func (c *Client) RunPipeline(ctx context.Context, workspace, repoSlug string, opts *PipelineRunOptions) (*Pipeline, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines", workspace, repoSlug)

	resp, err := c.Post(ctx, path, opts)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Pipeline](resp)
}

// StopPipeline stops a running pipeline
func (c *Client) StopPipeline(ctx context.Context, workspace, repoSlug, pipelineUUID string) error {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/stopPipeline", workspace, repoSlug, pipelineUUID)

	_, err := c.Post(ctx, path, nil)
	return err
}

// ListPipelineSteps lists steps for a pipeline
func (c *Client) ListPipelineSteps(ctx context.Context, workspace, repoSlug, pipelineUUID string) (*Paginated[PipelineStep], error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps", workspace, repoSlug, pipelineUUID)

	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	return ParseResponse[*Paginated[PipelineStep]](resp)
}

// GetPipelineStepLog gets the log for a pipeline step
func (c *Client) GetPipelineStepLog(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) (string, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/%s/log", workspace, repoSlug, pipelineUUID, stepUUID)

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
