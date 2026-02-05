package repo

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseRepoArg(t *testing.T) {
	tests := []struct {
		name          string
		arg           string
		wantWorkspace string
		wantRepo      string
		wantErr       bool
	}{
		{
			name:          "valid workspace/repo format",
			arg:           "myworkspace/myrepo",
			wantWorkspace: "myworkspace",
			wantRepo:      "myrepo",
			wantErr:       false,
		},
		{
			name:          "workspace with hyphen",
			arg:           "my-workspace/my-repo",
			wantWorkspace: "my-workspace",
			wantRepo:      "my-repo",
			wantErr:       false,
		},
		{
			name:          "workspace with underscore",
			arg:           "my_workspace/my_repo",
			wantWorkspace: "my_workspace",
			wantRepo:      "my_repo",
			wantErr:       false,
		},
		{
			name:    "missing repo",
			arg:     "workspace-only",
			wantErr: true,
		},
		{
			name:    "empty workspace",
			arg:     "/repo",
			wantErr: true,
		},
		{
			name:    "empty repo",
			arg:     "workspace/",
			wantErr: true,
		},
		{
			name:    "empty string",
			arg:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, repo, err := parseRepoArg(tt.arg)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepoArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if workspace != tt.wantWorkspace {
				t.Errorf("parseRepoArg() workspace = %v, want %v", workspace, tt.wantWorkspace)
			}

			if repo != tt.wantRepo {
				t.Errorf("parseRepoArg() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestGetRepoName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extracts repo name",
			input:    "myworkspace/myrepo",
			expected: "myrepo",
		},
		{
			name:     "repo with hyphen",
			input:    "workspace/my-repo",
			expected: "my-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, repo, err := parseRepoArg(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo != tt.expected {
				t.Errorf("got repo = %v, want %v", repo, tt.expected)
			}
		})
	}
}

func TestConfirmDeletion(t *testing.T) {
	tests := []struct {
		name         string
		repoName     string
		input        string
		wantConfirm  bool
	}{
		{
			name:        "correct confirmation",
			repoName:    "myrepo",
			input:       "myrepo\n",
			wantConfirm: true,
		},
		{
			name:        "wrong input",
			repoName:    "myrepo",
			input:       "wrongname\n",
			wantConfirm: false,
		},
		{
			name:        "empty input",
			repoName:    "myrepo",
			input:       "\n",
			wantConfirm: false,
		},
		{
			name:        "extra whitespace in input",
			repoName:    "myrepo",
			input:       "  myrepo  \n",
			wantConfirm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			confirmed := confirmDeletion(tt.repoName, reader)

			if confirmed != tt.wantConfirm {
				t.Errorf("confirmDeletion() = %v, want %v", confirmed, tt.wantConfirm)
			}
		})
	}
}

func TestDeleteWarningMessage(t *testing.T) {
	var buf bytes.Buffer
	printDeleteWarning(&buf)

	output := buf.String()
	
	// Should contain warning about permanent deletion
	if !strings.Contains(output, "cannot be undone") {
		t.Errorf("warning should mention deletion cannot be undone, got: %s", output)
	}
}
