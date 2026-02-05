package repo

import (
	"testing"
)

func TestDetectDefaultBranch(t *testing.T) {
	tests := []struct {
		name        string
		mainBranch  string
		wantBranch  string
	}{
		{
			name:       "explicit main branch",
			mainBranch: "main",
			wantBranch: "main",
		},
		{
			name:       "explicit master branch",
			mainBranch: "master",
			wantBranch: "master",
		},
		{
			name:       "custom branch",
			mainBranch: "develop",
			wantBranch: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch := detectDefaultBranch(tt.mainBranch, "")
			if branch != tt.wantBranch {
				t.Errorf("detectDefaultBranch() = %v, want %v", branch, tt.wantBranch)
			}
		})
	}
}

func TestDetectDefaultBranchWithFlagOverride(t *testing.T) {
	tests := []struct {
		name       string
		mainBranch string
		flagBranch string
		wantBranch string
	}{
		{
			name:       "flag overrides main branch",
			mainBranch: "main",
			flagBranch: "feature",
			wantBranch: "feature",
		},
		{
			name:       "empty flag uses main branch",
			mainBranch: "main",
			flagBranch: "",
			wantBranch: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch := detectDefaultBranch(tt.mainBranch, tt.flagBranch)
			if branch != tt.wantBranch {
				t.Errorf("detectDefaultBranch() = %v, want %v", branch, tt.wantBranch)
			}
		})
	}
}

func TestGetUpstreamRemoteName(t *testing.T) {
	// Test that a reasonable remote name is generated for the upstream
	name := getUpstreamRemoteName()
	if name == "" {
		t.Error("getUpstreamRemoteName() returned empty string")
	}
	if name != "upstream" {
		t.Errorf("getUpstreamRemoteName() = %v, want 'upstream'", name)
	}
}

func TestBuildFetchRefspec(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		wantRefspec string
	}{
		{
			name:        "main branch",
			branch:      "main",
			wantRefspec: "refs/heads/main:refs/remotes/upstream/main",
		},
		{
			name:        "master branch",
			branch:      "master",
			wantRefspec: "refs/heads/master:refs/remotes/upstream/master",
		},
		{
			name:        "feature branch",
			branch:      "feature/test",
			wantRefspec: "refs/heads/feature/test:refs/remotes/upstream/feature/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refspec := buildFetchRefspec("upstream", tt.branch)
			if refspec != tt.wantRefspec {
				t.Errorf("buildFetchRefspec() = %v, want %v", refspec, tt.wantRefspec)
			}
		})
	}
}
