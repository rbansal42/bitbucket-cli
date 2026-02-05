package repo

import (
	"testing"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/cmdutil"
)

func TestParseRepositoryFormat(t *testing.T) {
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
			name:          "workspace with numbers",
			repoFlag:      "workspace123/repo456",
			wantWorkspace: "workspace123",
			wantRepo:      "repo456",
			wantErr:       false,
		},
		{
			name:          "repo with dots",
			repoFlag:      "myworkspace/my.repo.name",
			wantWorkspace: "myworkspace",
			wantRepo:      "my.repo.name",
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
			name:     "missing repo - no slash",
			repoFlag: "workspace-only",
			wantErr:  true,
		},
		{
			name:     "empty workspace",
			repoFlag: "/repo",
			wantErr:  true,
		},
		{
			name:     "empty repo",
			repoFlag: "workspace/",
			wantErr:  true,
		},
		{
			name:     "empty flag falls back to git detection (fails in test)",
			repoFlag: "",
			wantErr:  true,
		},
		{
			name:     "just a slash",
			repoFlag: "/",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, repo, err := cmdutil.ParseRepository(tt.repoFlag)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if workspace != tt.wantWorkspace {
				t.Errorf("parseRepository() workspace = %q, want %q", workspace, tt.wantWorkspace)
			}

			if repo != tt.wantRepo {
				t.Errorf("parseRepository() repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestGetCloneURL(t *testing.T) {
	tests := []struct {
		name     string
		links    api.RepositoryLinks
		protocol string
		want     string
	}{
		{
			name: "select HTTPS protocol",
			links: api.RepositoryLinks{
				Clone: []api.CloneLink{
					{Href: "https://bitbucket.org/workspace/repo.git", Name: "https"},
					{Href: "git@bitbucket.org:workspace/repo.git", Name: "ssh"},
				},
			},
			protocol: "https",
			want:     "https://bitbucket.org/workspace/repo.git",
		},
		{
			name: "select SSH protocol",
			links: api.RepositoryLinks{
				Clone: []api.CloneLink{
					{Href: "https://bitbucket.org/workspace/repo.git", Name: "https"},
					{Href: "git@bitbucket.org:workspace/repo.git", Name: "ssh"},
				},
			},
			protocol: "ssh",
			want:     "git@bitbucket.org:workspace/repo.git",
		},
		{
			name: "fallback when protocol not found",
			links: api.RepositoryLinks{
				Clone: []api.CloneLink{
					{Href: "https://bitbucket.org/workspace/repo.git", Name: "https"},
				},
			},
			protocol: "ssh",
			want:     "https://bitbucket.org/workspace/repo.git", // falls back to first available
		},
		{
			name: "empty clone links returns empty string",
			links: api.RepositoryLinks{
				Clone: []api.CloneLink{},
			},
			protocol: "https",
			want:     "",
		},
		{
			name: "protocol case sensitive match",
			links: api.RepositoryLinks{
				Clone: []api.CloneLink{
					{Href: "https://bitbucket.org/workspace/repo.git", Name: "https"},
					{Href: "git@bitbucket.org:workspace/repo.git", Name: "ssh"},
				},
			},
			protocol: "HTTPS", // uppercase - won't match
			want:     "https://bitbucket.org/workspace/repo.git", // falls back to first
		},
		{
			name: "single clone link",
			links: api.RepositoryLinks{
				Clone: []api.CloneLink{
					{Href: "git@bitbucket.org:workspace/repo.git", Name: "ssh"},
				},
			},
			protocol: "https",
			want:     "git@bitbucket.org:workspace/repo.git", // only option available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCloneURL(tt.links, tt.protocol)
			if got != tt.want {
				t.Errorf("getCloneURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "bytes",
			bytes: 500,
			want:  "500 B",
		},
		{
			name:  "one kilobyte",
			bytes: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "kilobytes",
			bytes: 2560,
			want:  "2.5 KB",
		},
		{
			name:  "one megabyte",
			bytes: 1048576,
			want:  "1.0 MB",
		},
		{
			name:  "megabytes",
			bytes: 5242880,
			want:  "5.0 MB",
		},
		{
			name:  "one gigabyte",
			bytes: 1073741824,
			want:  "1.0 GB",
		},
		{
			name:  "gigabytes",
			bytes: 2684354560,
			want:  "2.5 GB",
		},
		{
			name:  "large gigabytes",
			bytes: 10737418240,
			want:  "10.0 GB",
		},
		{
			name:  "terabyte range",
			bytes: 1099511627776,
			want:  "1.0 TB", // function supports TB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestParseRepositoryErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		repoFlag       string
		wantErrContain string
	}{
		{
			name:           "missing slash shows format hint",
			repoFlag:       "invalid-format",
			wantErrContain: "workspace/repo",
		},
		{
			name:           "empty workspace shows validation error",
			repoFlag:       "/myrepo",
			wantErrContain: "empty",
		},
		{
			name:           "empty repo shows validation error",
			repoFlag:       "myworkspace/",
			wantErrContain: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := cmdutil.ParseRepository(tt.repoFlag)
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			errMsg := err.Error()
			if errMsg == "" {
				t.Error("expected non-empty error message")
			}

			// Error should contain helpful information
			if tt.wantErrContain != "" && !containsSubstring(errMsg, tt.wantErrContain) {
				t.Errorf("expected error to contain %q, got %q", tt.wantErrContain, errMsg)
			}
		})
	}
}

// containsSubstring checks if s contains substr (case-insensitive)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		len(substr) == 0 || 
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test that the repo command is properly set up
func TestNewCmdRepo(t *testing.T) {
	// Test that NewCmdRepo returns a valid command
	cmd := NewCmdRepo(nil)

	if cmd == nil {
		t.Fatal("NewCmdRepo returned nil")
	}

	if cmd.Use != "repo <command>" {
		t.Errorf("expected Use 'repo <command>', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}

	// Check for expected aliases
	hasRepositoryAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "repository" {
			hasRepositoryAlias = true
			break
		}
	}
	if !hasRepositoryAlias {
		t.Error("expected 'repository' alias")
	}
}

// Test getPreferredProtocol returns a valid protocol
func TestGetPreferredProtocol(t *testing.T) {
	protocol := getPreferredProtocol()

	// Should return either "https" or "ssh"
	if protocol != "https" && protocol != "ssh" {
		t.Errorf("expected protocol to be 'https' or 'ssh', got %q", protocol)
	}
}

// Test that RepositoryLinks type is properly accessible
func TestRepositoryLinksType(t *testing.T) {
	links := api.RepositoryLinks{
		Self:   api.Link{Href: "https://api.bitbucket.org/2.0/repositories/workspace/repo"},
		HTML:   api.Link{Href: "https://bitbucket.org/workspace/repo"},
		Avatar: api.Link{Href: "https://bitbucket.org/workspace/repo/avatar"},
		Clone: []api.CloneLink{
			{Href: "https://bitbucket.org/workspace/repo.git", Name: "https"},
			{Href: "git@bitbucket.org:workspace/repo.git", Name: "ssh"},
		},
	}

	if links.Self.Href == "" {
		t.Error("expected Self link to be set")
	}

	if links.HTML.Href == "" {
		t.Error("expected HTML link to be set")
	}

	if len(links.Clone) != 2 {
		t.Errorf("expected 2 clone links, got %d", len(links.Clone))
	}
}

// Test RepositoryFull type parsing
func TestRepositoryFullType(t *testing.T) {
	repo := api.RepositoryFull{
		UUID:        "{test-uuid}",
		Name:        "test-repo",
		Slug:        "test-repo",
		FullName:    "workspace/test-repo",
		Description: "A test repository",
		IsPrivate:   true,
		ForkPolicy:  "allow_forks",
		Language:    "go",
		Size:        1024000,
	}

	if repo.UUID != "{test-uuid}" {
		t.Errorf("expected UUID '{test-uuid}', got %q", repo.UUID)
	}

	if repo.Name != "test-repo" {
		t.Errorf("expected Name 'test-repo', got %q", repo.Name)
	}

	if !repo.IsPrivate {
		t.Error("expected IsPrivate to be true")
	}

	if repo.Size != 1024000 {
		t.Errorf("expected Size 1024000, got %d", repo.Size)
	}
}
