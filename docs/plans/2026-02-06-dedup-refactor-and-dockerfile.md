# Code Deduplication, Dockerfile & Cleanup - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate all code duplication across command packages by extracting shared utilities into `cmdutil`, remove duplicate type definitions in the `pr` package, and add a production-ready Dockerfile.

**Architecture:** Three phases executed sequentially. Phase 1 extracts 5 shared utility functions + 2 shared output helpers into `cmdutil`. Phase 2 eliminates duplicate type definitions in `internal/cmd/pr/shared.go` by reusing `api` package types. Phase 3 adds a multi-stage Dockerfile.

**Tech Stack:** Go 1.25.7, Docker (multi-stage build), existing `cmdutil` package

---

## Phase 1: Extract Shared Utilities into `cmdutil`

### Task 1: Add `TimeAgo` and `FormatTimeAgoString` to `cmdutil`

**Files:**
- Create: `internal/cmdutil/time.go`
- Create: `internal/cmdutil/time_test.go`

**What:** Extract the `timeAgo(t time.Time) string` function that is duplicated in:
- `internal/cmd/pr/view.go:279-316`
- `internal/cmd/issue/shared.go:100-137`
- `internal/cmd/pipeline/shared.go:102-143`
- `internal/cmd/repo/list.go:169-211`

The canonical version should handle both `time.Time` input and string input (for `snippet/list.go`'s `formatTime`), and include the `.IsZero()` guard from the pipeline version.

```go
// internal/cmdutil/time.go
package cmdutil

import (
	"fmt"
	"time"
)

// TimeAgo returns a human-readable relative time string for a time.Time value.
// Returns "-" for zero time values.
func TimeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 30*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 365*24*time.Hour:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(duration.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// TimeAgoFromString parses an ISO 8601 / RFC3339 timestamp string and returns
// a human-readable relative time. Returns the raw string on parse failure.
func TimeAgoFromString(isoTime string) string {
	if isoTime == "" {
		return "-"
	}

	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000000-07:00", isoTime)
		if err != nil {
			return isoTime
		}
	}

	return TimeAgo(t)
}
```

### Task 2: Add `GetUserDisplayName` to `cmdutil`

**Files:**
- Create: `internal/cmdutil/user.go`

**What:** Extract the `getUserDisplayName` function duplicated in:
- `internal/cmd/pr/view.go:265-276` (takes `PRUser` value)
- `internal/cmd/issue/shared.go:151-162` (takes `*api.User` pointer)

The canonical version uses `*api.User` since that's the API-level type.

```go
// internal/cmdutil/user.go
package cmdutil

import "github.com/rbansal42/bitbucket-cli/internal/api"

// GetUserDisplayName returns the best available display name for a user.
// Returns "-" if user is nil, falls back to Username, then "unknown".
func GetUserDisplayName(user *api.User) string {
	if user == nil {
		return "-"
	}
	if user.DisplayName != "" {
		return user.DisplayName
	}
	if user.Username != "" {
		return user.Username
	}
	return "unknown"
}
```

### Task 3: Add `PrintJSON` and `PrintTableHeader` to `cmdutil`

**Files:**
- Create: `internal/cmdutil/output.go`

**What:** Extract the repeated JSON output pattern (10+ copies) and table header pattern (7 copies).

```go
// internal/cmdutil/output.go
package cmdutil

import (
	"encoding/json"
	"fmt"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// PrintJSON marshals v as indented JSON and writes it to streams.Out.
func PrintJSON(streams *iostreams.IOStreams, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}

// PrintTableHeader writes a bold header line if color is enabled.
func PrintTableHeader(streams *iostreams.IOStreams, w *tabwriter.Writer, header string) {
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}
}
```

### Task 4: Add `ConfirmPrompt` to `cmdutil`

**Files:**
- Modify: `internal/cmdutil/output.go` (add to same file)

**What:** Extract the confirmation prompt duplicated in:
- `internal/cmd/issue/shared.go:165-172`
- `internal/cmd/snippet/delete.go:86-93` (inline)

```go
// ConfirmPrompt reads a line from reader and returns true if user typed y/yes.
func ConfirmPrompt(reader io.Reader) bool {
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return input == "y" || input == "yes"
	}
	return false
}
```

### Task 5: Replace all local `truncateString` with `cmdutil.TruncateString`

**Files to modify:**
- `internal/cmd/pr/list.go` -- remove `truncateString`, use `cmdutil.TruncateString`
- `internal/cmd/issue/shared.go` -- remove `truncateString`, use `cmdutil.TruncateString`
- `internal/cmd/pipeline/shared.go` -- remove `truncateString`, use `cmdutil.TruncateString`
- `internal/cmd/repo/list.go` -- remove `truncateString`, use `cmdutil.TruncateString`
- `internal/cmd/branch/list.go` -- remove `truncateMessage`, use `cmdutil.TruncateString`

### Task 6: Replace all local `timeAgo`/`formatTimeAgo`/`formatUpdated`/`formatTime` with `cmdutil.TimeAgo`

**Files to modify:**
- `internal/cmd/pr/view.go` -- remove `timeAgo`, use `cmdutil.TimeAgo`
- `internal/cmd/issue/shared.go` -- remove `timeAgo`, use `cmdutil.TimeAgo`
- `internal/cmd/pipeline/shared.go` -- remove `formatTimeAgo`, use `cmdutil.TimeAgo`
- `internal/cmd/repo/list.go` -- remove `formatUpdated`, use `cmdutil.TimeAgo`
- `internal/cmd/snippet/list.go` -- remove `formatTime`, use `cmdutil.TimeAgoFromString`

### Task 7: Replace all local `getUserDisplayName` with `cmdutil.GetUserDisplayName`

**Files to modify:**
- `internal/cmd/issue/shared.go` -- remove `getUserDisplayName`, use `cmdutil.GetUserDisplayName`
- NOTE: `internal/cmd/pr/view.go` uses `PRUser` type -- this is fixed in Phase 2

### Task 8: Replace all JSON output boilerplate with `cmdutil.PrintJSON`

**Files to modify (list commands):**
- `internal/cmd/pr/list.go` -- use `cmdutil.PrintJSON`
- `internal/cmd/branch/list.go` -- use `cmdutil.PrintJSON`
- `internal/cmd/repo/list.go` -- use `cmdutil.PrintJSON`
- `internal/cmd/snippet/list.go` -- use `cmdutil.PrintJSON`
- `internal/cmd/pr/view.go` -- use `cmdutil.PrintJSON`

### Task 9: Replace all table header boilerplate with `cmdutil.PrintTableHeader`

**Files to modify:**
- `internal/cmd/pr/list.go`
- `internal/cmd/issue/list.go`
- `internal/cmd/pipeline/list.go`
- `internal/cmd/repo/list.go`
- `internal/cmd/branch/list.go`
- `internal/cmd/workspace/list.go`
- `internal/cmd/snippet/list.go`

---

## Phase 2: Eliminate Duplicate Types in PR Package

### Task 10: Remove duplicate `PRUser`, `PRParticipant`, `PullRequest`, `PRComment` from `internal/cmd/pr/shared.go`

**Files to modify:**
- `internal/cmd/pr/shared.go` -- Remove types `PRUser` (lines 98-112), `PRParticipant` (lines 114-120), `PullRequest` (lines 122-163), `PRComment` (lines 166-176), and `getPullRequest` (lines 178-187)
- `internal/cmd/pr/view.go` -- Update to use `api.PullRequest`, `api.User`, `api.Participant`, replace `getUserDisplayName(PRUser)` with `cmdutil.GetUserDisplayName(&api.User)`, fix `time.Parse` of `CreatedOn` (api type uses `time.Time` not `string`)

---

## Phase 3: Dockerfile

### Task 11: Create multi-stage Dockerfile

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_DATE

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/rbansal42/bitbucket-cli/internal/cmd.Version=${VERSION} -X github.com/rbansal42/bitbucket-cli/internal/cmd.BuildDate=${BUILD_DATE}" \
    -o /bin/bb ./cmd/bb

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache git ca-certificates

COPY --from=builder /bin/bb /usr/local/bin/bb

ENTRYPOINT ["bb"]
```

```
# .dockerignore
.git
.worktrees
bin/
cover.out
*.md
!README.md
docs/
.github/
```
