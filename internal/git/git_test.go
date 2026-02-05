package git

import (
	"testing"
)

func TestParseBitbucketURL_SSHWithGit(t *testing.T) {
	url := "git@bitbucket.org:myworkspace/myrepo.git"
	remote, err := ParseBitbucketURL(url)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if remote.Workspace != "myworkspace" {
		t.Errorf("expected workspace 'myworkspace', got '%s'", remote.Workspace)
	}
	if remote.RepoSlug != "myrepo" {
		t.Errorf("expected repo 'myrepo', got '%s'", remote.RepoSlug)
	}
}

func TestParseBitbucketURL_SSHWithoutGit(t *testing.T) {
	url := "git@bitbucket.org:myworkspace/myrepo"
	remote, err := ParseBitbucketURL(url)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if remote.Workspace != "myworkspace" {
		t.Errorf("expected workspace 'myworkspace', got '%s'", remote.Workspace)
	}
	if remote.RepoSlug != "myrepo" {
		t.Errorf("expected repo 'myrepo', got '%s'", remote.RepoSlug)
	}
}

func TestParseBitbucketURL_HTTPSWithGit(t *testing.T) {
	url := "https://bitbucket.org/myworkspace/myrepo.git"
	remote, err := ParseBitbucketURL(url)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if remote.Workspace != "myworkspace" {
		t.Errorf("expected workspace 'myworkspace', got '%s'", remote.Workspace)
	}
	if remote.RepoSlug != "myrepo" {
		t.Errorf("expected repo 'myrepo', got '%s'", remote.RepoSlug)
	}
}

func TestParseBitbucketURL_HTTPSWithoutGit(t *testing.T) {
	url := "https://bitbucket.org/myworkspace/myrepo"
	remote, err := ParseBitbucketURL(url)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if remote.Workspace != "myworkspace" {
		t.Errorf("expected workspace 'myworkspace', got '%s'", remote.Workspace)
	}
	if remote.RepoSlug != "myrepo" {
		t.Errorf("expected repo 'myrepo', got '%s'", remote.RepoSlug)
	}
}

func TestParseBitbucketURL_HTTPSWithUsername(t *testing.T) {
	url := "https://username@bitbucket.org/myworkspace/myrepo.git"
	remote, err := ParseBitbucketURL(url)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if remote.Workspace != "myworkspace" {
		t.Errorf("expected workspace 'myworkspace', got '%s'", remote.Workspace)
	}
	if remote.RepoSlug != "myrepo" {
		t.Errorf("expected repo 'myrepo', got '%s'", remote.RepoSlug)
	}
}

func TestParseBitbucketURL_InvalidURLs(t *testing.T) {
	invalidURLs := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"random string", "not-a-url"},
		{"github ssh", "git@github.com:user/repo.git"},
		{"github https", "https://github.com/user/repo.git"},
		{"gitlab ssh", "git@gitlab.com:user/repo.git"},
		{"gitlab https", "https://gitlab.com/user/repo.git"},
		{"http instead of https", "http://bitbucket.org/workspace/repo.git"},
		{"missing workspace", "https://bitbucket.org/repo.git"},
		{"malformed ssh", "git@bitbucket.org/workspace/repo.git"},
	}

	for _, tc := range invalidURLs {
		t.Run(tc.name, func(t *testing.T) {
			remote, err := ParseBitbucketURL(tc.url)
			if err == nil {
				t.Errorf("expected error for URL '%s', got remote: %+v", tc.url, remote)
			}
		})
	}
}

func TestIsBitbucketURL_ReturnsTrueForBitbucket(t *testing.T) {
	bitbucketURLs := []string{
		"git@bitbucket.org:workspace/repo.git",
		"https://bitbucket.org/workspace/repo.git",
		"https://username@bitbucket.org/workspace/repo.git",
		"https://bitbucket.org/workspace/repo",
	}

	for _, url := range bitbucketURLs {
		t.Run(url, func(t *testing.T) {
			if !IsBitbucketURL(url) {
				t.Errorf("expected IsBitbucketURL to return true for '%s'", url)
			}
		})
	}
}

func TestIsBitbucketURL_ReturnsFalseForOtherProviders(t *testing.T) {
	nonBitbucketURLs := []struct {
		name string
		url  string
	}{
		{"github ssh", "git@github.com:user/repo.git"},
		{"github https", "https://github.com/user/repo.git"},
		{"gitlab ssh", "git@gitlab.com:user/repo.git"},
		{"gitlab https", "https://gitlab.com/user/repo.git"},
		{"azure devops", "https://dev.azure.com/org/project/_git/repo"},
		{"empty string", ""},
	}

	for _, tc := range nonBitbucketURLs {
		t.Run(tc.name, func(t *testing.T) {
			if IsBitbucketURL(tc.url) {
				t.Errorf("expected IsBitbucketURL to return false for '%s'", tc.url)
			}
		})
	}
}

func TestParseRemotes_SingleRemote(t *testing.T) {
	output := `origin	git@bitbucket.org:myworkspace/myrepo.git (fetch)
origin	git@bitbucket.org:myworkspace/myrepo.git (push)`

	remotes, err := parseRemotes(output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(remotes))
	}
	if remotes[0].Name != "origin" {
		t.Errorf("expected name 'origin', got '%s'", remotes[0].Name)
	}
	if remotes[0].FetchURL != "git@bitbucket.org:myworkspace/myrepo.git" {
		t.Errorf("expected fetch URL 'git@bitbucket.org:myworkspace/myrepo.git', got '%s'", remotes[0].FetchURL)
	}
	if remotes[0].PushURL != "git@bitbucket.org:myworkspace/myrepo.git" {
		t.Errorf("expected push URL 'git@bitbucket.org:myworkspace/myrepo.git', got '%s'", remotes[0].PushURL)
	}
	if remotes[0].Workspace != "myworkspace" {
		t.Errorf("expected workspace 'myworkspace', got '%s'", remotes[0].Workspace)
	}
	if remotes[0].RepoSlug != "myrepo" {
		t.Errorf("expected repo 'myrepo', got '%s'", remotes[0].RepoSlug)
	}
}

func TestParseRemotes_MultipleRemotes(t *testing.T) {
	output := `origin	git@bitbucket.org:workspace1/repo1.git (fetch)
origin	git@bitbucket.org:workspace1/repo1.git (push)
upstream	https://bitbucket.org/workspace2/repo2.git (fetch)
upstream	https://bitbucket.org/workspace2/repo2.git (push)`

	remotes, err := parseRemotes(output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(remotes))
	}

	// Find origin and upstream remotes (order is not guaranteed from map)
	var origin, upstream *Remote
	for i := range remotes {
		switch remotes[i].Name {
		case "origin":
			origin = &remotes[i]
		case "upstream":
			upstream = &remotes[i]
		}
	}

	if origin == nil {
		t.Fatal("expected to find 'origin' remote")
	}
	if upstream == nil {
		t.Fatal("expected to find 'upstream' remote")
	}

	if origin.Workspace != "workspace1" {
		t.Errorf("expected origin workspace 'workspace1', got '%s'", origin.Workspace)
	}
	if origin.RepoSlug != "repo1" {
		t.Errorf("expected origin repo 'repo1', got '%s'", origin.RepoSlug)
	}
	if upstream.Workspace != "workspace2" {
		t.Errorf("expected upstream workspace 'workspace2', got '%s'", upstream.Workspace)
	}
	if upstream.RepoSlug != "repo2" {
		t.Errorf("expected upstream repo 'repo2', got '%s'", upstream.RepoSlug)
	}
}

func TestParseRemotes_MixedProviders(t *testing.T) {
	output := `origin	git@bitbucket.org:workspace/repo.git (fetch)
origin	git@bitbucket.org:workspace/repo.git (push)
github	git@github.com:user/repo.git (fetch)
github	git@github.com:user/repo.git (push)`

	remotes, err := parseRemotes(output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(remotes))
	}

	// Find origin and github remotes
	var origin, github *Remote
	for i := range remotes {
		switch remotes[i].Name {
		case "origin":
			origin = &remotes[i]
		case "github":
			github = &remotes[i]
		}
	}

	if origin == nil {
		t.Fatal("expected to find 'origin' remote")
	}
	if github == nil {
		t.Fatal("expected to find 'github' remote")
	}

	// Bitbucket remote should have workspace and repo extracted
	if origin.Workspace != "workspace" {
		t.Errorf("expected origin workspace 'workspace', got '%s'", origin.Workspace)
	}
	if origin.RepoSlug != "repo" {
		t.Errorf("expected origin repo 'repo', got '%s'", origin.RepoSlug)
	}

	// GitHub remote should not have workspace and repo
	if github.Workspace != "" {
		t.Errorf("expected github workspace to be empty, got '%s'", github.Workspace)
	}
	if github.RepoSlug != "" {
		t.Errorf("expected github repo to be empty, got '%s'", github.RepoSlug)
	}
}

func TestParseRemotes_EmptyOutput(t *testing.T) {
	output := ""

	remotes, err := parseRemotes(output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(remotes) != 0 {
		t.Errorf("expected 0 remotes, got %d", len(remotes))
	}
}

func TestParseRemotes_WhitespaceOnlyOutput(t *testing.T) {
	output := "   \n\t\n   "

	remotes, err := parseRemotes(output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(remotes) != 0 {
		t.Errorf("expected 0 remotes, got %d", len(remotes))
	}
}

func TestParseRemotes_DifferentFetchPushURLs(t *testing.T) {
	output := `origin	https://bitbucket.org/workspace/repo.git (fetch)
origin	git@bitbucket.org:workspace/repo.git (push)`

	remotes, err := parseRemotes(output)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(remotes))
	}

	if remotes[0].FetchURL != "https://bitbucket.org/workspace/repo.git" {
		t.Errorf("expected fetch URL 'https://bitbucket.org/workspace/repo.git', got '%s'", remotes[0].FetchURL)
	}
	if remotes[0].PushURL != "git@bitbucket.org:workspace/repo.git" {
		t.Errorf("expected push URL 'git@bitbucket.org:workspace/repo.git', got '%s'", remotes[0].PushURL)
	}
}

func TestParseBitbucketURL_TrimsWhitespace(t *testing.T) {
	url := "  git@bitbucket.org:workspace/repo.git  "
	remote, err := ParseBitbucketURL(url)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if remote.Workspace != "workspace" {
		t.Errorf("expected workspace 'workspace', got '%s'", remote.Workspace)
	}
	if remote.RepoSlug != "repo" {
		t.Errorf("expected repo 'repo', got '%s'", remote.RepoSlug)
	}
}
