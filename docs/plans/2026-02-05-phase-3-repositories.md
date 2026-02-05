# Phase 3: Repository Commands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement full repository management commands for bb CLI - clone, create, fork, list, view, delete, and sync.

**Architecture:** Repo commands use the shared API client. Support both SSH and HTTPS clone URLs based on user config. Integrate with git for local operations.

**Tech Stack:** Go 1.25+, Cobra, internal/api client, internal/git for operations

---

## Task 1: Add Repository API Types and Client Methods

**Files:**
- Create: `internal/api/repositories.go`

**Types to implement:**
- `Repository` - full repository representation
- `RepositoryLinks` - links (clone, html, etc.)
- `CloneLink` - clone URLs (ssh, https)
- `Project` - project reference
- `Workspace` - workspace reference
- `RepositoryListOptions` - filtering options
- `RepositoryCreateOptions` - creation options
- `ForkOptions` - fork options

**Client methods:**
- `ListRepositories(ctx, workspace, opts)` - GET /repositories/{workspace}
- `GetRepository(ctx, workspace, repoSlug)` - GET /repositories/{workspace}/{repo}
- `CreateRepository(ctx, workspace, opts)` - POST /repositories/{workspace}/{repo}
- `DeleteRepository(ctx, workspace, repoSlug)` - DELETE /repositories/{workspace}/{repo}
- `ForkRepository(ctx, workspace, repoSlug, opts)` - POST /repositories/{workspace}/{repo}/forks
- `ListForks(ctx, workspace, repoSlug)` - GET /repositories/{workspace}/{repo}/forks
- `GetMainBranch(ctx, workspace, repoSlug)` - GET /repositories/{workspace}/{repo}/main-branch

---

## Task 2: Implement bb repo list Command

**Files:**
- Create: `internal/cmd/repo/repo.go`
- Create: `internal/cmd/repo/list.go`
- Create: `internal/cmd/repo/shared.go`
- Modify: `internal/cmd/root.go`

**Features:**
- List repositories in a workspace
- `--workspace` / `-w` flag (required or from config)
- `--limit` flag (default 30)
- `--json` flag for JSON output
- Table output: NAME, DESCRIPTION, VISIBILITY, UPDATED
- `--public` / `--private` visibility filter
- `--sort` flag (name, updated_on, created_on)

---

## Task 3: Implement bb repo view Command

**Files:**
- Create: `internal/cmd/repo/view.go`

**Features:**
- View repository details
- Show name, description, visibility, size
- Show clone URLs (SSH and HTTPS)
- Show default branch
- Show project (if any)
- `--web` flag to open in browser
- `--json` flag for JSON output
- Auto-detect repo from git remote if not specified

---

## Task 4: Implement bb repo clone Command

**Files:**
- Create: `internal/cmd/repo/clone.go`

**Features:**
- Clone a repository
- Support `workspace/repo` format
- Support full URL (ssh or https)
- Respect `git_protocol` config (ssh vs https)
- Optional destination directory
- `--depth` flag for shallow clone
- Show progress during clone

---

## Task 5: Implement bb repo create Command

**Files:**
- Create: `internal/cmd/repo/create.go`

**Features:**
- Create new repository
- `--name` flag (required)
- `--description` flag
- `--private` / `--public` flags (default private)
- `--project` flag to assign to project
- `--clone` flag to clone after creation
- Interactive mode if name not provided
- Print clone URL after creation

---

## Task 6: Implement bb repo fork Command

**Files:**
- Create: `internal/cmd/repo/fork.go`

**Features:**
- Fork a repository to your workspace
- `--workspace` flag for destination workspace
- `--name` flag for fork name (default: same as original)
- `--clone` flag to clone the fork
- `--remote` flag to add fork as remote to existing clone
- Print fork URL after creation

---

## Task 7: Implement bb repo delete Command

**Files:**
- Create: `internal/cmd/repo/delete.go`

**Features:**
- Delete a repository (requires confirmation)
- `--yes` flag to skip confirmation
- Warn about permanent deletion
- Cannot be undone message

---

## Task 8: Implement bb repo sync Command

**Files:**
- Create: `internal/cmd/repo/sync.go`

**Features:**
- Sync fork with upstream
- Fetch upstream changes
- Update default branch
- `--branch` flag to specify branch
- `--force` flag to force update

---

## Task 9: Add Tests for Repository Commands

**Files:**
- Create: `internal/api/repositories_test.go`
- Create: `internal/cmd/repo/repo_test.go`

**Test coverage:**
- API client methods with mock HTTP server
- Repository URL parsing
- Clone URL selection logic
- Flag parsing and validation

---

## Summary

After completing all tasks:
- Full repo workflow: list, view, clone, create, fork, delete, sync
- API types for Bitbucket repository endpoints
- Unit tests for API and commands
- Ready for Phase 4 (Issue and Pipeline commands)
