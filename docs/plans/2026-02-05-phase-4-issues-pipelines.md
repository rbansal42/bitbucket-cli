# Phase 4: Issues and Pipelines Commands

## Overview

Add issue tracking and pipeline management commands to the bb CLI.

## Commands to Implement

### Issues (8 commands)
- `bb issue list` - List issues in a repository
- `bb issue view <id>` - View issue details
- `bb issue create` - Create a new issue
- `bb issue close <id>` - Close an issue
- `bb issue reopen <id>` - Reopen an issue
- `bb issue comment <id>` - Add comment to an issue
- `bb issue edit <id>` - Edit an issue
- `bb issue delete <id>` - Delete an issue

### Pipelines (6 commands)
- `bb pipeline list` - List pipelines
- `bb pipeline view <id>` - View pipeline details
- `bb pipeline run` - Run a pipeline
- `bb pipeline stop <id>` - Stop a running pipeline
- `bb pipeline logs <id>` - View pipeline step logs
- `bb pipeline steps <id>` - List pipeline steps

## API Endpoints

### Issues
- GET /repositories/{workspace}/{repo_slug}/issues
- POST /repositories/{workspace}/{repo_slug}/issues
- GET /repositories/{workspace}/{repo_slug}/issues/{issue_id}
- PUT /repositories/{workspace}/{repo_slug}/issues/{issue_id}
- DELETE /repositories/{workspace}/{repo_slug}/issues/{issue_id}
- GET /repositories/{workspace}/{repo_slug}/issues/{issue_id}/comments
- POST /repositories/{workspace}/{repo_slug}/issues/{issue_id}/comments

### Pipelines
- GET /repositories/{workspace}/{repo_slug}/pipelines
- POST /repositories/{workspace}/{repo_slug}/pipelines
- GET /repositories/{workspace}/{repo_slug}/pipelines/{pipeline_uuid}
- GET /repositories/{workspace}/{repo_slug}/pipelines/{pipeline_uuid}/steps
- GET /repositories/{workspace}/{repo_slug}/pipelines/{pipeline_uuid}/steps/{step_uuid}/log
- POST /repositories/{workspace}/{repo_slug}/pipelines/{pipeline_uuid}/stopPipeline

## Implementation Tasks

### Task 1: API Types and Client - Issues
**Files:** `internal/api/issues.go`
- Issue struct with fields: ID, Title, Content, State, Kind, Priority, Reporter, Assignee, etc.
- IssueComment struct
- IssueListOptions (state, kind, priority, assignee filters)
- Methods: ListIssues, GetIssue, CreateIssue, UpdateIssue, DeleteIssue
- Methods: ListIssueComments, CreateIssueComment

### Task 2: API Types and Client - Pipelines
**Files:** `internal/api/pipelines.go`
- Pipeline struct with fields: UUID, BuildNumber, State, Target, Creator, etc.
- PipelineStep struct
- PipelineState enum (PENDING, IN_PROGRESS, COMPLETED, FAILED, etc.)
- Methods: ListPipelines, GetPipeline, RunPipeline, StopPipeline
- Methods: ListPipelineSteps, GetPipelineLog

### Task 3: Issue Commands - List/View
**Files:** `internal/cmd/issue/issue.go`, `internal/cmd/issue/list.go`, `internal/cmd/issue/view.go`
- Parent issue command
- List with filters (--state, --kind, --priority, --assignee)
- View with --web flag
- JSON output support

### Task 4: Issue Commands - Create/Edit
**Files:** `internal/cmd/issue/create.go`, `internal/cmd/issue/edit.go`
- Create with --title, --body, --kind, --priority, --assignee
- Edit with same flags
- Interactive mode for create

### Task 5: Issue Commands - State Management
**Files:** `internal/cmd/issue/close.go`, `internal/cmd/issue/reopen.go`, `internal/cmd/issue/delete.go`
- Close with optional comment
- Reopen command
- Delete with confirmation

### Task 6: Issue Command - Comment
**Files:** `internal/cmd/issue/comment.go`
- Add comment to issue
- Optional --body flag (else use editor)

### Task 7: Pipeline Commands - List/View
**Files:** `internal/cmd/pipeline/pipeline.go`, `internal/cmd/pipeline/list.go`, `internal/cmd/pipeline/view.go`
- Parent pipeline command
- List with filters (--status, --branch)
- View with step summary
- JSON output support

### Task 8: Pipeline Commands - Run/Stop
**Files:** `internal/cmd/pipeline/run.go`, `internal/cmd/pipeline/stop.go`
- Run on branch/commit with --branch, --commit, --custom
- Stop a running pipeline with confirmation

### Task 9: Pipeline Commands - Logs/Steps
**Files:** `internal/cmd/pipeline/logs.go`, `internal/cmd/pipeline/steps.go`
- List steps for a pipeline
- View logs for a step

### Task 10: Tests
**Files:** `internal/api/issues_test.go`, `internal/api/pipelines_test.go`, `internal/cmd/issue/issue_test.go`, `internal/cmd/pipeline/pipeline_test.go`
- API tests for all methods
- Command tests for parsing, shared utilities

### Task 11: Root Command Integration
**Files:** `internal/cmd/root.go`
- Add issue and pipeline commands to root

## Execution Plan

**Batch 1 (Parallel):** Tasks 1, 2 - API layer
**Batch 2 (Parallel):** Tasks 3, 4, 5, 6, 7, 8, 9 - Commands
**Batch 3:** Task 10 - Tests
**Batch 4:** Task 11 - Integration

## Success Criteria

- All commands functional
- Tests pass
- Builds cleanly
- Follows existing patterns from PR and Repo commands
