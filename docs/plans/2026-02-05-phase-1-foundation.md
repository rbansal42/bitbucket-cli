# Phase 1: Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete the foundation layer of bb CLI - authentication, API client, git detection, and core commands.

**Architecture:** Cobra-based CLI with modular command structure. API client handles all Bitbucket REST API calls. Config stored in ~/.config/bb/ with tokens in system keychain.

**Tech Stack:** Go 1.25+, Cobra, Viper, go-keyring, golang.org/x/oauth2

---

## Task 1: Wire Up Auth Commands to Root

**Files:**
- Modify: `internal/cmd/root.go`

**Step 1: Add auth import and wire up command**

```go
// Add to imports
"github.com/rbansal42/bb/internal/cmd/auth"

// In init() function, replace comment with:
rootCmd.AddCommand(auth.NewCmdAuth(GetStreams()))
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Test command exists**

Run: `go run ./cmd/bb auth --help`
Expected: Shows auth subcommands (login, logout, status, token)

**Step 4: Commit**

```bash
git add internal/cmd/root.go
git commit -m "feat: wire up auth commands to root"
```

---

## Task 2: Implement bb api Command

**Files:**
- Create: `internal/cmd/api/api.go`
- Modify: `internal/cmd/root.go`

**Step 1: Create api command**

Create `internal/cmd/api/api.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/api"
	"github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/iostreams"
)

type apiOptions struct {
	streams    *iostreams.IOStreams
	method     string
	headers    []string
	inputFile  string
	rawOutput  bool
	hostname   string
}

func NewCmdAPI(streams *iostreams.IOStreams) *cobra.Command {
	opts := &apiOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated API request",
		Long: `Make an authenticated request to the Bitbucket API.

The endpoint argument should be a path like "/repositories/workspace/repo"
or a full URL. If only a path is given, the Bitbucket API base URL is prepended.

The default HTTP method is GET. Use -X to specify a different method.`,
		Example: `  # Get the authenticated user
  $ bb api /user

  # List repositories in a workspace  
  $ bb api /repositories/myworkspace

  # Create a pull request
  $ bb api -X POST /repositories/workspace/repo/pullrequests --input pr.json

  # Get with query parameters
  $ bb api "/repositories/workspace/repo/pullrequests?state=OPEN"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAPI(opts, args[0])
		},
	}

	cmd.Flags().StringVarP(&opts.method, "method", "X", "GET", "HTTP method")
	cmd.Flags().StringArrayVarP(&opts.headers, "header", "H", nil, "Add HTTP header (can be repeated)")
	cmd.Flags().StringVar(&opts.inputFile, "input", "", "File to read request body from (use '-' for stdin)")
	cmd.Flags().BoolVar(&opts.rawOutput, "raw", false, "Output raw response without formatting")
	cmd.Flags().StringVar(&opts.hostname, "hostname", config.DefaultHost, "Bitbucket hostname")

	return cmd
}

func runAPI(opts *apiOptions, endpoint string) error {
	// Get token
	token, err := getToken(opts.hostname)
	if err != nil {
		return err
	}

	// Build client
	client := api.NewClient(api.WithToken(token))

	// Parse endpoint
	var reqURL string
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		reqURL = endpoint
	} else {
		reqURL = endpoint
	}

	// Parse query parameters from URL
	var query url.Values
	if idx := strings.Index(reqURL, "?"); idx != -1 {
		var err error
		query, err = url.ParseQuery(reqURL[idx+1:])
		if err != nil {
			return fmt.Errorf("invalid query string: %w", err)
		}
		reqURL = reqURL[:idx]
	}

	// Read request body if provided
	var body interface{}
	if opts.inputFile != "" {
		var reader io.Reader
		if opts.inputFile == "-" {
			reader = os.Stdin
		} else {
			f, err := os.Open(opts.inputFile)
			if err != nil {
				return fmt.Errorf("could not open input file: %w", err)
			}
			defer f.Close()
			reader = f
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("could not read input: %w", err)
		}

		if err := json.Unmarshal(data, &body); err != nil {
			return fmt.Errorf("invalid JSON input: %w", err)
		}
	}

	// Build request
	req := &api.Request{
		Method:  opts.method,
		Path:    reqURL,
		Query:   query,
		Body:    body,
		Headers: make(map[string]string),
	}

	// Add custom headers
	for _, h := range opts.headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			req.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// Execute request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Do(ctx, req)
	if err != nil {
		return err
	}

	// Output response
	if opts.rawOutput || !json.Valid(resp.Body) {
		fmt.Println(string(resp.Body))
	} else {
		var prettyJSON map[string]interface{}
		if err := json.Unmarshal(resp.Body, &prettyJSON); err != nil {
			fmt.Println(string(resp.Body))
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(prettyJSON)
		}
	}

	return nil
}

func getToken(hostname string) (string, error) {
	// Check environment variable first
	if token := os.Getenv("BB_TOKEN"); token != "" {
		return token, nil
	}
	if token := os.Getenv("BITBUCKET_TOKEN"); token != "" {
		return token, nil
	}

	// Get from keyring
	hosts, err := config.LoadHostsConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	user := hosts.GetActiveUser(hostname)
	if user == "" {
		return "", fmt.Errorf("not logged in. Run 'bb auth login' first")
	}

	tokenData, _, err := config.GetTokenFromEnvOrKeyring(hostname, user)
	if err != nil {
		return "", err
	}

	// Try to parse as OAuth token JSON
	var oauthToken struct {
		AccessToken string `json:"access_token"`
	}
	if json.Unmarshal([]byte(tokenData), &oauthToken) == nil && oauthToken.AccessToken != "" {
		return oauthToken.AccessToken, nil
	}

	return tokenData, nil
}
```

**Step 2: Wire up to root command**

Add to `internal/cmd/root.go`:
- Import: `"github.com/rbansal42/bb/internal/cmd/api"`
- In init(): `rootCmd.AddCommand(api.NewCmdAPI(GetStreams()))`

**Step 3: Verify build**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/cmd/api/api.go internal/cmd/root.go
git commit -m "feat: add bb api command for raw API access"
```

---

## Task 3: Implement bb browse Command

**Files:**
- Create: `internal/cmd/browse/browse.go`
- Modify: `internal/cmd/root.go`

**Step 1: Create browse command**

Create `internal/cmd/browse/browse.go`:

```go
package browse

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/browser"
	"github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/git"
	"github.com/rbansal42/bb/internal/iostreams"
)

type browseOptions struct {
	streams   *iostreams.IOStreams
	repo      string
	branch    string
	commit    string
	noBrowser bool
}

func NewCmdBrowse(streams *iostreams.IOStreams) *cobra.Command {
	opts := &browseOptions{
		streams: streams,
	}

	cmd := &cobra.Command{
		Use:   "browse [path]",
		Short: "Open the repository in a web browser",
		Long: `Open the current repository in a web browser.

With no arguments, opens the repository home page. With a path argument,
opens that file or directory in the browser.`,
		Example: `  # Open the current repository
  $ bb browse

  # Open a specific file
  $ bb browse src/main.go

  # Open a specific branch
  $ bb browse --branch feature/auth

  # Just print the URL
  $ bb browse -n`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var path string
			if len(args) > 0 {
				path = args[0]
			}
			return runBrowse(opts, path)
		},
	}

	cmd.Flags().StringVarP(&opts.repo, "repo", "R", "", "Repository in WORKSPACE/REPO format")
	cmd.Flags().StringVarP(&opts.branch, "branch", "b", "", "Branch to browse")
	cmd.Flags().StringVarP(&opts.commit, "commit", "c", "", "Commit to browse")
	cmd.Flags().BoolVarP(&opts.noBrowser, "no-browser", "n", false, "Print URL instead of opening browser")

	return cmd
}

func runBrowse(opts *browseOptions, path string) error {
	var workspace, repoSlug string

	if opts.repo != "" {
		// Parse WORKSPACE/REPO format
		parts := splitRepo(opts.repo)
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format, expected WORKSPACE/REPO")
		}
		workspace, repoSlug = parts[0], parts[1]
	} else {
		// Detect from git remote
		remote, err := git.GetDefaultRemote()
		if err != nil {
			return fmt.Errorf("could not detect repository: %w\nUse --repo to specify manually", err)
		}
		workspace, repoSlug = remote.Workspace, remote.RepoSlug
	}

	// Build URL
	baseURL := fmt.Sprintf("https://%s/%s/%s", config.DefaultHost, workspace, repoSlug)

	var url string
	switch {
	case opts.commit != "":
		url = fmt.Sprintf("%s/commits/%s", baseURL, opts.commit)
	case opts.branch != "":
		if path != "" {
			url = fmt.Sprintf("%s/src/%s/%s", baseURL, opts.branch, path)
		} else {
			url = fmt.Sprintf("%s/branch/%s", baseURL, opts.branch)
		}
	case path != "":
		branch := opts.branch
		if branch == "" {
			var err error
			branch, err = git.GetCurrentBranch()
			if err != nil {
				branch = "main"
			}
		}
		url = fmt.Sprintf("%s/src/%s/%s", baseURL, branch, path)
	default:
		url = baseURL
	}

	if opts.noBrowser {
		fmt.Println(url)
		return nil
	}

	opts.streams.Info("Opening %s in browser...", url)
	return browser.Open(url)
}

func splitRepo(repo string) []string {
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			return []string{repo[:i], repo[i+1:]}
		}
	}
	return []string{repo}
}
```

**Step 2: Wire up to root command**

Add to `internal/cmd/root.go`:
- Import: `"github.com/rbansal42/bb/internal/cmd/browse"`
- In init(): `rootCmd.AddCommand(browse.NewCmdBrowse(GetStreams()))`

**Step 3: Verify build and test**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/cmd/browse/browse.go internal/cmd/root.go
git commit -m "feat: add bb browse command to open repo in browser"
```

---

## Task 4: Implement bb config Commands

**Files:**
- Create: `internal/cmd/config/config.go`
- Create: `internal/cmd/config/get.go`
- Create: `internal/cmd/config/set.go`
- Create: `internal/cmd/config/list.go`
- Modify: `internal/cmd/root.go`

**Step 1: Create config command files**

Create `internal/cmd/config/config.go`:

```go
package config

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

func NewCmdConfig(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <command>",
		Short: "Manage configuration for bb",
		Long: `Display or change configuration settings for bb.

Configuration values are stored in ~/.config/bb/config.yml`,
	}

	cmd.AddCommand(NewCmdConfigGet(streams))
	cmd.AddCommand(NewCmdConfigSet(streams))
	cmd.AddCommand(NewCmdConfigList(streams))

	return cmd
}
```

Create `internal/cmd/config/get.go`:

```go
package config

import (
	"fmt"

	"github.com/spf13/cobra"

	bbconfig "github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/iostreams"
)

func NewCmdConfigGet(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print the value of a configuration key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(streams, args[0])
		},
	}
	return cmd
}

func runConfigGet(streams *iostreams.IOStreams, key string) error {
	cfg, err := bbconfig.LoadConfig()
	if err != nil {
		return err
	}

	var value string
	switch key {
	case "git_protocol":
		value = cfg.GitProtocol
	case "editor":
		value = cfg.Editor
	case "pager":
		value = cfg.Pager
	case "browser":
		value = cfg.Browser
	case "prompt":
		value = cfg.Prompt
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if value != "" {
		fmt.Println(value)
	}
	return nil
}
```

Create `internal/cmd/config/set.go`:

```go
package config

import (
	"fmt"

	"github.com/spf13/cobra"

	bbconfig "github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/iostreams"
)

func NewCmdConfigSet(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Update a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(streams, args[0], args[1])
		},
	}
	return cmd
}

func runConfigSet(streams *iostreams.IOStreams, key, value string) error {
	cfg, err := bbconfig.LoadConfig()
	if err != nil {
		return err
	}

	switch key {
	case "git_protocol":
		if value != "ssh" && value != "https" {
			return fmt.Errorf("invalid git_protocol value: must be 'ssh' or 'https'")
		}
		cfg.GitProtocol = value
	case "editor":
		cfg.Editor = value
	case "pager":
		cfg.Pager = value
	case "browser":
		cfg.Browser = value
	case "prompt":
		if value != "enabled" && value != "disabled" {
			return fmt.Errorf("invalid prompt value: must be 'enabled' or 'disabled'")
		}
		cfg.Prompt = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return bbconfig.SaveConfig(cfg)
}
```

Create `internal/cmd/config/list.go`:

```go
package config

import (
	"fmt"

	"github.com/spf13/cobra"

	bbconfig "github.com/rbansal42/bb/internal/config"
	"github.com/rbansal42/bb/internal/iostreams"
)

func NewCmdConfigList(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(streams)
		},
	}
	return cmd
}

func runConfigList(streams *iostreams.IOStreams) error {
	cfg, err := bbconfig.LoadConfig()
	if err != nil {
		return err
	}

	fmt.Printf("git_protocol=%s\n", cfg.GitProtocol)
	fmt.Printf("editor=%s\n", cfg.Editor)
	fmt.Printf("prompt=%s\n", cfg.Prompt)
	fmt.Printf("pager=%s\n", cfg.Pager)
	fmt.Printf("browser=%s\n", cfg.Browser)

	return nil
}
```

**Step 2: Wire up to root command**

Add to `internal/cmd/root.go`:
- Import: `bbconfigcmd "github.com/rbansal42/bb/internal/cmd/config"`
- In init(): `rootCmd.AddCommand(bbconfigcmd.NewCmdConfig(GetStreams()))`

**Step 3: Verify build**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/cmd/config/*.go internal/cmd/root.go
git commit -m "feat: add bb config commands for configuration management"
```

---

## Task 5: Add Makefile and Build Tooling

**Files:**
- Create: `Makefile`
- Create: `.goreleaser.yml`

**Step 1: Create Makefile**

```makefile
.PHONY: build install test lint clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/rbansal42/bb/internal/cmd.Version=$(VERSION) -X github.com/rbansal42/bb/internal/cmd.BuildDate=$(BUILD_DATE)"

build:
	go build $(LDFLAGS) -o bin/bb ./cmd/bb

install:
	go install $(LDFLAGS) ./cmd/bb

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
	go clean

# Development helpers
run:
	go run ./cmd/bb $(ARGS)

fmt:
	go fmt ./...

vet:
	go vet ./...
```

**Step 2: Create .goreleaser.yml**

```yaml
version: 2

project_name: bb

before:
  hooks:
    - go mod tidy

builds:
  - id: bb
    main: ./cmd/bb
    binary: bb
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/rbansal42/bb/internal/cmd.Version={{.Version}}
      - -X github.com/rbansal42/bb/internal/cmd.BuildDate={{.Date}}

archives:
  - id: default
    format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"

release:
  github:
    owner: rbansal42
    name: bb
```

**Step 3: Add bin/ to gitignore**

```bash
echo "bin/" >> .gitignore
```

**Step 4: Verify build with Makefile**

Run: `make build`
Expected: Creates bin/bb binary

**Step 5: Commit**

```bash
git add Makefile .goreleaser.yml .gitignore
git commit -m "chore: add Makefile and goreleaser config"
```

---

## Task 6: Add Unit Tests for Config Package

**Files:**
- Create: `internal/config/config_test.go`
- Create: `internal/config/keyring_test.go`

**Step 1: Create config_test.go**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir(t *testing.T) {
	// Test with BB_CONFIG_DIR set
	t.Run("BB_CONFIG_DIR", func(t *testing.T) {
		os.Setenv("BB_CONFIG_DIR", "/custom/path")
		defer os.Unsetenv("BB_CONFIG_DIR")

		dir, err := ConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != "/custom/path" {
			t.Errorf("expected /custom/path, got %s", dir)
		}
	})

	// Test with XDG_CONFIG_HOME set
	t.Run("XDG_CONFIG_HOME", func(t *testing.T) {
		os.Unsetenv("BB_CONFIG_DIR")
		os.Setenv("XDG_CONFIG_HOME", "/xdg/config")
		defer os.Unsetenv("XDG_CONFIG_HOME")

		dir, err := ConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join("/xdg/config", "bb")
		if dir != expected {
			t.Errorf("expected %s, got %s", expected, dir)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.GitProtocol != "ssh" {
		t.Errorf("expected git_protocol=ssh, got %s", cfg.GitProtocol)
	}
	if cfg.Prompt != "enabled" {
		t.Errorf("expected prompt=enabled, got %s", cfg.Prompt)
	}
}

func TestHostsConfig(t *testing.T) {
	hosts := make(HostsConfig)

	// Test SetActiveUser
	hosts.SetActiveUser("bitbucket.org", "testuser")

	if hosts.GetActiveUser("bitbucket.org") != "testuser" {
		t.Errorf("expected testuser, got %s", hosts.GetActiveUser("bitbucket.org"))
	}

	// Test GetGitProtocol default
	if hosts.GetGitProtocol("bitbucket.org") != "ssh" {
		t.Errorf("expected default git protocol ssh")
	}

	// Test AuthenticatedHosts
	authHosts := hosts.AuthenticatedHosts()
	if len(authHosts) != 1 || authHosts[0] != "bitbucket.org" {
		t.Errorf("unexpected authenticated hosts: %v", authHosts)
	}
}
```

**Step 2: Create keyring_test.go**

```go
package config

import (
	"os"
	"testing"
)

func TestKeyringKey(t *testing.T) {
	key := keyringKey("bitbucket.org", "testuser")
	expected := "bitbucket.org:testuser"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestLookupEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	value := lookupEnv("TEST_VAR")
	if value != "test_value" {
		t.Errorf("expected test_value, got %s", value)
	}

	value = lookupEnv("NONEXISTENT_VAR")
	if value != "" {
		t.Errorf("expected empty string, got %s", value)
	}
}

func TestGetEnvToken(t *testing.T) {
	// Test BB_TOKEN
	os.Setenv("BB_TOKEN", "bb_token_value")
	defer os.Unsetenv("BB_TOKEN")

	token := getEnvToken()
	if token != "bb_token_value" {
		t.Errorf("expected bb_token_value, got %s", token)
	}

	// Test BITBUCKET_TOKEN fallback
	os.Unsetenv("BB_TOKEN")
	os.Setenv("BITBUCKET_TOKEN", "bitbucket_token_value")
	defer os.Unsetenv("BITBUCKET_TOKEN")

	token = getEnvToken()
	if token != "bitbucket_token_value" {
		t.Errorf("expected bitbucket_token_value, got %s", token)
	}
}
```

**Step 3: Run tests**

Run: `go test -v ./internal/config/...`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/config/*_test.go
git commit -m "test: add unit tests for config package"
```

---

## Task 7: Add Unit Tests for Git Package

**Files:**
- Create: `internal/git/git_test.go`

**Step 1: Create git_test.go**

```go
package git

import (
	"testing"
)

func TestParseBitbucketURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		workspace string
		repo      string
		wantErr   bool
	}{
		{
			name:      "SSH URL",
			url:       "git@bitbucket.org:myworkspace/myrepo.git",
			workspace: "myworkspace",
			repo:      "myrepo",
		},
		{
			name:      "SSH URL without .git",
			url:       "git@bitbucket.org:myworkspace/myrepo",
			workspace: "myworkspace",
			repo:      "myrepo",
		},
		{
			name:      "HTTPS URL",
			url:       "https://bitbucket.org/myworkspace/myrepo.git",
			workspace: "myworkspace",
			repo:      "myrepo",
		},
		{
			name:      "HTTPS URL without .git",
			url:       "https://bitbucket.org/myworkspace/myrepo",
			workspace: "myworkspace",
			repo:      "myrepo",
		},
		{
			name:      "HTTPS URL with username",
			url:       "https://user@bitbucket.org/myworkspace/myrepo.git",
			workspace: "myworkspace",
			repo:      "myrepo",
		},
		{
			name:    "Invalid URL",
			url:     "https://github.com/user/repo",
			wantErr: true,
		},
		{
			name:    "Not a URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote, err := ParseBitbucketURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if remote.Workspace != tt.workspace {
				t.Errorf("workspace: expected %s, got %s", tt.workspace, remote.Workspace)
			}
			if remote.RepoSlug != tt.repo {
				t.Errorf("repo: expected %s, got %s", tt.repo, remote.RepoSlug)
			}
		})
	}
}

func TestIsBitbucketURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"git@bitbucket.org:workspace/repo.git", true},
		{"https://bitbucket.org/workspace/repo", true},
		{"git@github.com:user/repo.git", false},
		{"https://gitlab.com/user/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsBitbucketURL(tt.url)
			if got != tt.want {
				t.Errorf("IsBitbucketURL(%s) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestParseRemotes(t *testing.T) {
	output := `origin	git@bitbucket.org:myworkspace/myrepo.git (fetch)
origin	git@bitbucket.org:myworkspace/myrepo.git (push)
upstream	https://bitbucket.org/otherworkspace/otherrepo.git (fetch)
upstream	https://bitbucket.org/otherworkspace/otherrepo.git (push)
`
	remotes, err := parseRemotes(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(remotes))
	}

	// Find origin remote
	var origin *Remote
	for _, r := range remotes {
		if r.Name == "origin" {
			origin = &r
			break
		}
	}

	if origin == nil {
		t.Fatal("origin remote not found")
	}

	if origin.Workspace != "myworkspace" {
		t.Errorf("expected workspace myworkspace, got %s", origin.Workspace)
	}
	if origin.RepoSlug != "myrepo" {
		t.Errorf("expected repo myrepo, got %s", origin.RepoSlug)
	}
}
```

**Step 2: Run tests**

Run: `go test -v ./internal/git/...`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/git/git_test.go
git commit -m "test: add unit tests for git package"
```

---

## Task 8: Add Unit Tests for API Client

**Files:**
- Create: `internal/api/client_test.go`

**Step 1: Create client_test.go**

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected base URL %s, got %s", DefaultBaseURL, client.baseURL)
	}
}

func TestWithToken(t *testing.T) {
	client := NewClient(WithToken("test-token"))
	if client.token != "test-token" {
		t.Errorf("expected token test-token, got %s", client.token)
	}
}

func TestWithBaseURL(t *testing.T) {
	client := NewClient(WithBaseURL("https://custom.api.com/"))
	if client.baseURL != "https://custom.api.com" {
		t.Errorf("expected base URL https://custom.api.com, got %s", client.baseURL)
	}
}

func TestClientDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header with Bearer token")
		}
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("expected User-Agent %s", UserAgent)
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithToken("test-token"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Get(ctx, "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestClientDoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Repository not found",
				"detail":  "The repository does not exist",
			},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Get(ctx, "/nonexistent", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Repository not found" {
		t.Errorf("expected message 'Repository not found', got '%s'", apiErr.Message)
	}
}

func TestParseResponse(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	resp := &Response{
		Body: []byte(`{"name":"test","value":42}`),
	}

	result, err := ParseResponse[TestStruct](resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", result.Name)
	}
	if result.Value != 42 {
		t.Errorf("expected value 42, got %d", result.Value)
	}
}
```

**Step 2: Run tests**

Run: `go test -v ./internal/api/...`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/api/client_test.go
git commit -m "test: add unit tests for API client"
```

---

## Summary

After completing all tasks:

1. All foundation commands wired up and working
2. Build tooling in place (Makefile, goreleaser)
3. Unit tests for core packages (config, git, api)
4. Project ready for Phase 2 (Pull Request commands)

Run full test suite: `make test`
Build binary: `make build`
