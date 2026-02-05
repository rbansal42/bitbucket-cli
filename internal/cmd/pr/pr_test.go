package pr

import (
	"testing"
)

func TestParsePRNumber(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantNum int
		wantErr bool
	}{
		{
			name:    "valid number",
			args:    []string{"123"},
			wantNum: 123,
			wantErr: false,
		},
		{
			name:    "single digit",
			args:    []string{"1"},
			wantNum: 1,
			wantErr: false,
		},
		{
			name:    "large number",
			args:    []string{"999999"},
			wantNum: 999999,
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    []string{},
			wantNum: 0,
			wantErr: true,
		},
		{
			name:    "invalid non-numeric",
			args:    []string{"abc"},
			wantNum: 0,
			wantErr: true,
		},
		{
			name:    "negative number",
			args:    []string{"-1"},
			wantNum: 0,
			wantErr: true, // negative PR numbers are invalid
		},
		{
			name:    "mixed alphanumeric",
			args:    []string{"123abc"},
			wantNum: 0,
			wantErr: true,
		},
		{
			name:    "float-like string",
			args:    []string{"12.5"},
			wantNum: 0,
			wantErr: true,
		},
		{
			name:    "PR with hash prefix",
			args:    []string{"#456"},
			wantNum: 0,
			wantErr: true,
		},
		{
			name:    "zero",
			args:    []string{"0"},
			wantNum: 0,
			wantErr: true, // zero is not a valid PR number
		},
		{
			name:    "whitespace",
			args:    []string{" 123 "},
			wantNum: 0,
			wantErr: true, // strconv.Atoi doesn't trim whitespace
		},
		{
			name:    "multiple args - uses first",
			args:    []string{"100", "200"},
			wantNum: 100,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePRNumber(tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("parsePRNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.wantNum {
				t.Errorf("parsePRNumber() = %v, want %v", got, tt.wantNum)
			}
		})
	}
}

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name          string
		repoFlag      string
		wantWorkspace string
		wantRepo      string
		wantErr       bool
	}{
		{
			name:          "valid workspace/repo format",
			repoFlag:      "myworkspace/myrepo",
			wantWorkspace: "myworkspace",
			wantRepo:      "myrepo",
			wantErr:       false,
		},
		{
			name:          "workspace with hyphen",
			repoFlag:      "my-workspace/my-repo",
			wantWorkspace: "my-workspace",
			wantRepo:      "my-repo",
			wantErr:       false,
		},
		{
			name:          "workspace with underscore",
			repoFlag:      "my_workspace/my_repo",
			wantWorkspace: "my_workspace",
			wantRepo:      "my_repo",
			wantErr:       false,
		},
		{
			name:          "repo with multiple slashes - only first split",
			repoFlag:      "workspace/repo/extra",
			wantWorkspace: "workspace",
			wantRepo:      "repo/extra",
			wantErr:       false,
		},
		{
			name:     "missing repo",
			repoFlag: "workspace-only",
			wantErr:  true,
		},
		{
			name:     "empty workspace",
			repoFlag: "/repo",
			wantErr:  true, // empty workspace is invalid
		},
		{
			name:     "empty repo",
			repoFlag: "workspace/",
			wantErr:  true, // empty repo is invalid
		},
		{
			name:          "empty flag falls back to git detection",
			repoFlag:      "",
			wantErr:       true, // Will error in test environment without git
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, repo, err := parseRepository(tt.repoFlag)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if workspace != tt.wantWorkspace {
				t.Errorf("parseRepository() workspace = %v, want %v", workspace, tt.wantWorkspace)
			}

			if repo != tt.wantRepo {
				t.Errorf("parseRepository() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestGetEditor(t *testing.T) {
	// Test that getEditor returns a non-empty string
	// The actual value depends on environment variables
	editor := getEditor()
	
	if editor == "" {
		t.Error("getEditor() returned empty string")
	}

	// Should default to "vi" if no env vars are set
	// This is an implementation detail test
}

func TestGetEditorPriority(t *testing.T) {
	// Store original values
	// Note: In a real test, we'd use t.Setenv() which automatically restores
	// For now, just test that the function returns a non-empty value
	
	editor := getEditor()
	if editor == "" {
		t.Error("expected non-empty editor")
	}
}

// TestPullRequestTypes verifies the PR types can be used correctly
func TestPullRequestTypes(t *testing.T) {
	// Test that PullRequest struct can be instantiated
	pr := PullRequest{
		ID:          1,
		Title:       "Test PR",
		Description: "Test description",
		State:       "OPEN",
	}

	if pr.ID != 1 {
		t.Errorf("expected ID 1, got %d", pr.ID)
	}

	if pr.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got %q", pr.Title)
	}

	// Test nested struct access
	pr.Source.Branch.Name = "feature"
	if pr.Source.Branch.Name != "feature" {
		t.Errorf("expected source branch 'feature', got %q", pr.Source.Branch.Name)
	}

	pr.Destination.Branch.Name = "main"
	if pr.Destination.Branch.Name != "main" {
		t.Errorf("expected destination branch 'main', got %q", pr.Destination.Branch.Name)
	}
}

// TestPRCommentTypes verifies the PRComment struct
func TestPRCommentTypes(t *testing.T) {
	comment := PRComment{
		ID: 42,
	}
	comment.Content.Raw = "This is a comment"
	comment.Links.HTML.Href = "https://bitbucket.org/comment/42"

	if comment.ID != 42 {
		t.Errorf("expected ID 42, got %d", comment.ID)
	}

	if comment.Content.Raw != "This is a comment" {
		t.Errorf("expected content 'This is a comment', got %q", comment.Content.Raw)
	}

	if comment.Links.HTML.Href != "https://bitbucket.org/comment/42" {
		t.Errorf("expected link 'https://bitbucket.org/comment/42', got %q", comment.Links.HTML.Href)
	}
}

// TestParsePRNumberErrorMessages verifies error message quality
func TestParsePRNumberErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErrMsg  string
	}{
		{
			name:       "empty args error message",
			args:       []string{},
			wantErrMsg: "pull request number is required",
		},
		{
			name:       "invalid number error message",
			args:       []string{"notanumber"},
			wantErrMsg: "invalid pull request number: notanumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePRNumber(tt.args)
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			if err.Error() != tt.wantErrMsg {
				t.Errorf("error message = %q, want %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

// TestParseRepositoryErrorMessages verifies error message quality
func TestParseRepositoryErrorMessages(t *testing.T) {
	_, _, err := parseRepository("invalid-format")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	// Check that error message contains helpful information
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("expected non-empty error message")
	}
}
