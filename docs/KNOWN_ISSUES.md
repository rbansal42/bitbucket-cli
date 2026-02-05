# Known Issues and Future Improvements

This document tracks issues identified during code review that should be addressed in future phases.

## Phase 1 Code Review (2026-02-05)

### Important (Should Fix)

1. **`bb api` command doesn't reuse the existing API client**
   - File: `internal/cmd/api/api.go`
   - Issue: Creates a raw `http.Client` instead of using `internal/api.Client`
   - Impact: Duplicates HTTP handling logic, inconsistent User-Agent headers
   - Fix: Refactor to use `api.NewClient()` with the `api.Request` struct

2. **Missing tests for command packages**
   - Files: `internal/cmd/api/`, `internal/cmd/browse/`, `internal/cmd/config/`
   - Issue: No test files for any command packages
   - Fix: Add tests that invoke the RunE functions with mock IOStreams

3. **`bb api --json` flag parses all values as strings**
   - File: `internal/cmd/api/api.go:100-107`
   - Issue: `jsonBody[parts[0]] = parts[1]` treats everything as strings
   - Impact: API calls needing booleans or numbers will fail
   - Fix: Attempt JSON unmarshaling of the value first, fall back to string

4. **No context timeout propagation in `handlePagination`**
   - File: `internal/cmd/api/api.go:279-291`
   - Issue: Pagination loop creates requests without context
   - Fix: Pass context through and use `http.NewRequestWithContext()`

### Minor (Nice to Have)

1. **`--host` flag declared but unused in config commands**
   - Files: `internal/cmd/config/get.go`, `internal/cmd/config/set.go`
   - Fix: Either implement per-host config or remove the flag

2. **Config list skips zero/empty values**
   - File: `internal/cmd/config/list.go`
   - Fix: Show all keys with "(default)" annotation for unset values

3. **`bb api` pagination assumes all responses have `values` array**
   - File: `internal/cmd/api/api.go`
   - Fix: Print a warning if `--paginate` was specified but response lacks pagination fields

4. **Form data encoding in api command is simplistic**
   - File: `internal/cmd/api/api.go`
   - Issue: Raw field values aren't URL-encoded
   - Fix: Use `url.Values` and proper encoding

## Resolved in Phase 1

- ~~`bb browse` code duplication~~ - Now uses shared git package
- ~~`bb browse` hardcodes "main" branch~~ - Now detects current branch
- ~~Missing BITBUCKET_TOKEN check in api command~~ - Added fallback
