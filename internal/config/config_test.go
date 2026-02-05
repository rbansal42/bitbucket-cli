package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDir_WithBBConfigDir(t *testing.T) {
	// Set BB_CONFIG_DIR environment variable
	testDir := "/tmp/bb-test-config"
	t.Setenv("BB_CONFIG_DIR", testDir)
	t.Setenv("XDG_CONFIG_HOME", "") // Clear XDG to ensure BB_CONFIG_DIR takes precedence

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() returned error: %v", err)
	}

	if dir != testDir {
		t.Errorf("ConfigDir() = %q, want %q", dir, testDir)
	}
}

func TestConfigDir_WithXDGConfigHome(t *testing.T) {
	// Clear BB_CONFIG_DIR, set XDG_CONFIG_HOME
	t.Setenv("BB_CONFIG_DIR", "")
	xdgDir := "/tmp/xdg-config"
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() returned error: %v", err)
	}

	expected := filepath.Join(xdgDir, "bb")
	if dir != expected {
		t.Errorf("ConfigDir() = %q, want %q", dir, expected)
	}
}

func TestConfigDir_BBConfigDirTakesPrecedence(t *testing.T) {
	// Set both variables - BB_CONFIG_DIR should take precedence
	bbDir := "/tmp/bb-priority"
	xdgDir := "/tmp/xdg-config"
	t.Setenv("BB_CONFIG_DIR", bbDir)
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() returned error: %v", err)
	}

	if dir != bbDir {
		t.Errorf("ConfigDir() = %q, want %q (BB_CONFIG_DIR should take precedence)", dir, bbDir)
	}
}

func TestConfigDir_DefaultsToHomeConfig(t *testing.T) {
	// Clear both environment variables
	t.Setenv("BB_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() returned error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Could not get home directory: %v", err)
	}

	expected := filepath.Join(home, ".config", "bb")
	if dir != expected {
		t.Errorf("ConfigDir() = %q, want %q", dir, expected)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	if config == nil {
		t.Fatal("defaultConfig() returned nil")
	}

	// Check GitProtocol default
	if config.GitProtocol != "ssh" {
		t.Errorf("defaultConfig().GitProtocol = %q, want %q", config.GitProtocol, "ssh")
	}

	// Check Prompt default
	if config.Prompt != "enabled" {
		t.Errorf("defaultConfig().Prompt = %q, want %q", config.Prompt, "enabled")
	}

	// Check HTTPTimeout default
	if config.HTTPTimeout != 30 {
		t.Errorf("defaultConfig().HTTPTimeout = %d, want %d", config.HTTPTimeout, 30)
	}

	// Verify unset fields are empty/zero
	if config.Editor != "" {
		t.Errorf("defaultConfig().Editor = %q, want empty string", config.Editor)
	}
	if config.Pager != "" {
		t.Errorf("defaultConfig().Pager = %q, want empty string", config.Pager)
	}
	if config.Browser != "" {
		t.Errorf("defaultConfig().Browser = %q, want empty string", config.Browser)
	}
}

func TestHostsConfig_SetActiveUser_NewHost(t *testing.T) {
	hosts := make(HostsConfig)

	hosts.SetActiveUser("bitbucket.org", "testuser")

	// Verify host was created
	if _, ok := hosts["bitbucket.org"]; !ok {
		t.Fatal("SetActiveUser did not create host entry")
	}

	// Verify user was set
	if hosts["bitbucket.org"].User != "testuser" {
		t.Errorf("SetActiveUser set User = %q, want %q", hosts["bitbucket.org"].User, "testuser")
	}

	// Verify user was added to Users map
	if hosts["bitbucket.org"].Users == nil {
		t.Fatal("SetActiveUser did not initialize Users map")
	}
	if _, ok := hosts["bitbucket.org"].Users["testuser"]; !ok {
		t.Error("SetActiveUser did not add user to Users map")
	}
}

func TestHostsConfig_SetActiveUser_ExistingHost(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{
		Users:       map[string]*UserConfig{"olduser": {}},
		User:        "olduser",
		GitProtocol: "https",
	}

	hosts.SetActiveUser("bitbucket.org", "newuser")

	// Verify user was updated
	if hosts["bitbucket.org"].User != "newuser" {
		t.Errorf("SetActiveUser set User = %q, want %q", hosts["bitbucket.org"].User, "newuser")
	}

	// Verify GitProtocol was preserved
	if hosts["bitbucket.org"].GitProtocol != "https" {
		t.Errorf("SetActiveUser changed GitProtocol to %q, should preserve existing value", hosts["bitbucket.org"].GitProtocol)
	}

	// Verify both users exist in Users map
	if _, ok := hosts["bitbucket.org"].Users["olduser"]; !ok {
		t.Error("SetActiveUser removed existing user from Users map")
	}
	if _, ok := hosts["bitbucket.org"].Users["newuser"]; !ok {
		t.Error("SetActiveUser did not add new user to Users map")
	}
}

func TestHostsConfig_GetActiveUser(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{User: "activeuser"}

	user := hosts.GetActiveUser("bitbucket.org")
	if user != "activeuser" {
		t.Errorf("GetActiveUser() = %q, want %q", user, "activeuser")
	}
}

func TestHostsConfig_GetActiveUser_NonexistentHost(t *testing.T) {
	hosts := make(HostsConfig)

	user := hosts.GetActiveUser("nonexistent.org")
	if user != "" {
		t.Errorf("GetActiveUser() for nonexistent host = %q, want empty string", user)
	}
}

func TestHostsConfig_GetActiveUser_NoUserSet(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{} // Host exists but no user set

	user := hosts.GetActiveUser("bitbucket.org")
	if user != "" {
		t.Errorf("GetActiveUser() for host with no user = %q, want empty string", user)
	}
}

func TestHostsConfig_GetGitProtocol_DefaultValue(t *testing.T) {
	hosts := make(HostsConfig)

	// Test for nonexistent host - should return default "ssh"
	protocol := hosts.GetGitProtocol("bitbucket.org")
	if protocol != "ssh" {
		t.Errorf("GetGitProtocol() for nonexistent host = %q, want %q", protocol, "ssh")
	}
}

func TestHostsConfig_GetGitProtocol_EmptyProtocol(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{GitProtocol: ""} // Host exists but protocol not set

	protocol := hosts.GetGitProtocol("bitbucket.org")
	if protocol != "ssh" {
		t.Errorf("GetGitProtocol() for host with empty protocol = %q, want %q", protocol, "ssh")
	}
}

func TestHostsConfig_GetGitProtocol_CustomProtocol(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{GitProtocol: "https"}

	protocol := hosts.GetGitProtocol("bitbucket.org")
	if protocol != "https" {
		t.Errorf("GetGitProtocol() = %q, want %q", protocol, "https")
	}
}

func TestHostsConfig_AuthenticatedHosts_Empty(t *testing.T) {
	hosts := make(HostsConfig)

	result := hosts.AuthenticatedHosts()
	if len(result) != 0 {
		t.Errorf("AuthenticatedHosts() returned %d hosts, want 0", len(result))
	}
}

func TestHostsConfig_AuthenticatedHosts_NoActiveUsers(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{} // No user set
	hosts["example.org"] = &HostConfig{}   // No user set

	result := hosts.AuthenticatedHosts()
	if len(result) != 0 {
		t.Errorf("AuthenticatedHosts() returned %d hosts, want 0 (no hosts have active users)", len(result))
	}
}

func TestHostsConfig_AuthenticatedHosts_WithActiveUsers(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{User: "user1"}
	hosts["example.org"] = &HostConfig{User: "user2"}
	hosts["nouser.org"] = &HostConfig{} // No user set

	result := hosts.AuthenticatedHosts()

	// Should return 2 hosts (those with active users)
	if len(result) != 2 {
		t.Errorf("AuthenticatedHosts() returned %d hosts, want 2", len(result))
	}

	// Check that both authenticated hosts are in the result
	hasHost := func(host string) bool {
		for _, h := range result {
			if h == host {
				return true
			}
		}
		return false
	}

	if !hasHost("bitbucket.org") {
		t.Error("AuthenticatedHosts() missing bitbucket.org")
	}
	if !hasHost("example.org") {
		t.Error("AuthenticatedHosts() missing example.org")
	}
	if hasHost("nouser.org") {
		t.Error("AuthenticatedHosts() incorrectly included nouser.org (has no active user)")
	}
}

func TestHostsConfig_SetActiveUser_NilUsersMap(t *testing.T) {
	hosts := make(HostsConfig)
	hosts["bitbucket.org"] = &HostConfig{
		Users: nil, // Explicitly nil
		User:  "",
	}

	// Should not panic and should initialize Users map
	hosts.SetActiveUser("bitbucket.org", "testuser")

	if hosts["bitbucket.org"].Users == nil {
		t.Fatal("SetActiveUser did not initialize nil Users map")
	}
	if _, ok := hosts["bitbucket.org"].Users["testuser"]; !ok {
		t.Error("SetActiveUser did not add user to Users map")
	}
}
